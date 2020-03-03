package scheduler

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"github.com/wavesplatform/gowaves/pkg/wallet"
	"go.uber.org/zap"
)

type Emit struct {
	Timestamp            uint64
	KeyPair              proto.KeyPair
	GenSignature         []byte
	BaseTarget           types.BaseTarget
	ParentBlockSignature crypto.Signature
}

type SchedulerImpl struct {
	seeder     seeder
	mine       chan Emit
	cancel     []func()
	settings   *settings.BlockchainSettings
	mu         sync.Mutex
	internal   internal
	emits      []Emit
	state      state.State
	tm         types.Time
	consensus  types.MinerConsensus
	minerDelay proto.Timestamp
}

type internal interface {
	schedule(state state.State, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) []Emit
}

type internalImpl struct {
}

func (a internalImpl) schedule(state state.State, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) []Emit {
	var greatGrandParentTimestamp proto.Timestamp = 0
	if confirmedBlockHeight > 2 {
		greatGrandParent, err := state.BlockByHeight(confirmedBlockHeight - 2)
		if err != nil {
			zap.S().Error(err)
			return nil
		}
		greatGrandParentTimestamp = greatGrandParent.Timestamp
	}

	fairPosActivated, vrfActivated, err := func() (bool, bool, error) {
		defer state.Mutex().RLock().Unlock()
		fairPosActivated, err := state.IsActiveAtHeight(int16(settings.FairPoS), confirmedBlockHeight)
		if err != nil {
			return false, false, errors.Wrap(err, "failed get fairPosActivated")
		}
		vrfActivated, err := state.IsActivated(int16(settings.BlockV5))
		if err != nil {
			return false, false, errors.Wrap(err, "failed get vrfActivated")
		}
		return fairPosActivated, vrfActivated, nil
	}()
	if err != nil {
		zap.S().Error(err)
		return nil
	}

	var pos consensus.PosCalculator = &consensus.NxtPosCalculator{}
	if fairPosActivated {
		pos = &consensus.FairPosCalculator{}
	}

	var gsp consensus.GenerationSignatureProvider = &consensus.NXTGenerationSignatureProvider{}
	if vrfActivated {
		gsp = &consensus.VRFGenerationSignatureProvider{}
	}
	hitSourceHeader, err := state.HeaderByHeight(pos.HeightForHit(confirmedBlockHeight))
	if err != nil {
		zap.S().Error(err)
		return nil
	}

	zap.S().Infof("Scheduler: confirmedBlock: sig %s, gensig: %s, confirmedHeight: %d", confirmedBlock.BlockSignature, confirmedBlock.GenSignature, confirmedBlockHeight)

	var out []Emit
	for _, keyPair := range keyPairs {
		var key [crypto.KeySize]byte = keyPair.Public
		genSigBlock := confirmedBlock.BlockHeader
		if vrfActivated {
			key = keyPair.Secret
			genSigBlock = *hitSourceHeader
		}
		genSig, err := gsp.GenerationSignature(key, genSigBlock.GenSignature)
		if err != nil {
			zap.S().Error(err)
			continue
		}
		source, err := gsp.HitSource(key, hitSourceHeader.GenSignature)
		if err != nil {
			zap.S().Error(err)
			continue
		}
		hit, err := consensus.GenHit(source)
		if err != nil {
			zap.S().Error(err)
			continue
		}

		addr, err := keyPair.Addr(schema)
		if err != nil {
			zap.S().Error(err)
			continue
		}
		locked := state.Mutex().RLock()
		effectiveBalance, err := state.EffectiveBalanceStable(proto.NewRecipientFromAddress(addr), confirmedBlockHeight-1000, confirmedBlockHeight)
		locked.Unlock()
		if err != nil {
			zap.S().Error(err)
			continue
		}

		delay, err := pos.CalculateDelay(hit, confirmedBlock.BlockHeader.BaseTarget, effectiveBalance)
		if err != nil {
			zap.S().Error(err)
			continue
		}

		baseTarget, err := pos.CalculateBaseTarget(AverageBlockDelaySeconds, confirmedBlockHeight, confirmedBlock.BlockHeader.BaseTarget, confirmedBlock.Timestamp, greatGrandParentTimestamp, delay+confirmedBlock.Timestamp)
		if err != nil {
			zap.S().Error(err)
			continue
		}

		out = append(out, Emit{
			Timestamp:            confirmedBlock.Timestamp + delay + 10,
			KeyPair:              keyPair,
			GenSignature:         genSig,
			BaseTarget:           baseTarget,
			ParentBlockSignature: confirmedBlock.BlockSignature,
		})
	}
	return out
}

type seeder interface {
	Seeds() [][]byte
}

func NewScheduler(state state.State, seeder seeder, settings *settings.BlockchainSettings, tm types.Time, consensus types.MinerConsensus, minerDelay proto.Timestamp) *SchedulerImpl {
	return newScheduler(internalImpl{}, state, seeder, settings, tm, consensus, minerDelay)
}

func newScheduler(internal internal, state state.State, seeder seeder, settings *settings.BlockchainSettings, tm types.Time, consensus types.MinerConsensus, minerDelay proto.Timestamp) *SchedulerImpl {
	if seeder == nil {
		seeder = wallet.NewWallet()
	}
	return &SchedulerImpl{
		seeder:     seeder,
		mine:       make(chan Emit, 1),
		settings:   settings,
		internal:   internal,
		state:      state,
		mu:         sync.Mutex{},
		tm:         tm,
		consensus:  consensus,
		minerDelay: minerDelay,
	}
}

func (a *SchedulerImpl) Mine() chan Emit {
	return a.mine
}

func (a *SchedulerImpl) Reschedule() {
	if len(a.seeder.Seeds()) == 0 {
		return
	}

	if !a.consensus.IsMiningAllowed() {
		return
	}

	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()
	if currentTimestamp-lastKnownBlock.Timestamp > a.minerDelay {
		return
	}

	mu := a.state.Mutex()
	locked := mu.RLock()

	h, err := a.state.Height()
	if err != nil {
		zap.S().Error(err)
		locked.Unlock()
		return
	}

	block, err := a.state.BlockByHeight(h)
	if err != nil {
		zap.S().Error(err)
		locked.Unlock()
		return
	}
	locked.Unlock()

	a.reschedule(a.state, block, h)
}

func (a *SchedulerImpl) reschedule(state state.State, confirmedBlock *proto.Block, confirmedBlockHeight uint64) {
	if len(a.seeder.Seeds()) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	// stop previous timeouts
	for _, cancel := range a.cancel {
		cancel()
	}
	a.cancel = nil

	keyPairs, err := makeKeyPairs(a.seeder.Seeds())
	if err != nil {
		zap.S().Error(err)
		return
	}

	emits := a.internal.schedule(state, keyPairs, a.settings.AddressSchemeCharacter, a.settings.AverageBlockDelaySeconds, confirmedBlock, confirmedBlockHeight)
	a.emits = emits
	now := proto.NewTimestampFromTime(a.tm.Now())
	for _, emit := range emits {
		if emit.Timestamp > now { // timestamp in future
			timeout := emit.Timestamp - now
			emit_ := emit
			cancel := cancellable.After(time.Duration(timeout)*time.Millisecond, func() {
				select {
				case a.mine <- emit_:
				default:
					zap.S().Debug("cannot emit a.mine, chan is full")
				}

			})
			a.cancel = append(a.cancel, cancel)
		} else {
			select {
			case a.mine <- emit:
			default:
				zap.S().Debug("Scheduler: cannot emit a.mine, chan is full")
			}

		}
	}
}

func (a *SchedulerImpl) Emits() []Emit {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.emits
}

func makeKeyPairs(seeds [][]byte) ([]proto.KeyPair, error) {
	var out []proto.KeyPair
	for _, bts := range seeds {
		kp, err := proto.NewKeyPair(bts)
		if err != nil {
			return nil, err
		}
		out = append(out, kp)
	}
	return out, nil
}
