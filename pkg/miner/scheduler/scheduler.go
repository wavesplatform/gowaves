package scheduler

import (
	"bytes"
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
	keyPairs   []proto.KeyPair
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

func NewScheduler(state state.State, pairs []proto.KeyPair, settings *settings.BlockchainSettings, tm types.Time, consensus types.MinerConsensus, minerDelay proto.Timestamp) *SchedulerImpl {
	return newScheduler(internalImpl{}, state, pairs, settings, tm, consensus, minerDelay)
}

func newScheduler(internal internal, state state.State, pairs []proto.KeyPair, settings *settings.BlockchainSettings, tm types.Time, consensus types.MinerConsensus, minerDelay proto.Timestamp) *SchedulerImpl {
	return &SchedulerImpl{
		keyPairs:   pairs,
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
	if len(a.keyPairs) == 0 {
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
	if len(a.keyPairs) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	// stop previous timeouts
	for _, cancel := range a.cancel {
		cancel()
	}
	a.cancel = nil

	emits := a.internal.schedule(state, a.keyPairs, a.settings.AddressSchemeCharacter, a.settings.AverageBlockDelaySeconds, confirmedBlock, confirmedBlockHeight)
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

// TODO: this function should be moved to wallet module, as well as keyPairs.
// Private keys should only be accessible from wallet module.
// All the other modules that need them, e.g. miner, api should call wallet's methods
// to sign what is needed.
// For now let's keep keys *only* in Scheduler.
func (a *SchedulerImpl) SignTransactionWith(pk crypto.PublicKey, tx proto.Transaction) error {
	for _, kp := range a.keyPairs {
		if bytes.Equal(kp.Public.Bytes(), pk.Bytes()) {
			return tx.Sign(a.settings.AddressSchemeCharacter, kp.Secret)
		}
	}
	return errors.New("public key not found")
}

func (a *SchedulerImpl) Emits() []Emit {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.emits
}
