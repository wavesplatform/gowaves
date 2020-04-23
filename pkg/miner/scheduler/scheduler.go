package scheduler

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"github.com/wavesplatform/gowaves/pkg/wallet"
	"go.uber.org/zap"
)

type Emit struct {
	Timestamp    uint64
	KeyPair      proto.KeyPair
	GenSignature []byte
	VRF          []byte
	BaseTarget   types.BaseTarget
	Parent       proto.BlockID
}

type SchedulerImpl struct {
	seeder        seeder
	mine          chan Emit
	cancel        []func()
	settings      *settings.BlockchainSettings
	mu            sync.Mutex
	internal      internal
	emits         []Emit
	storage       state.State
	tm            types.Time
	consensus     types.MinerConsensus
	outdatePeriod proto.Timestamp
}

type internal interface {
	schedule(state state.StateInfo, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) ([]Emit, error)
}

type internalImpl struct {
}

func (a internalImpl) schedule(storage state.StateInfo, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) ([]Emit, error) {
	vrfActivated, err := storage.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return nil, errors.Wrap(err, "failed get vrfActivated")
	}
	if vrfActivated {
		return a.scheduleWithVrf(storage, keyPairs, schema, AverageBlockDelaySeconds, confirmedBlock, confirmedBlockHeight)
	}
	return a.scheduleWithoutVrf(storage, keyPairs, schema, AverageBlockDelaySeconds, confirmedBlock, confirmedBlockHeight)
}

func (a internalImpl) scheduleWithVrf(storage state.StateInfo, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) ([]Emit, error) {
	var greatGrandParentTimestamp proto.Timestamp = 0
	if confirmedBlockHeight > 2 {
		greatGrandParent, err := storage.BlockByHeight(confirmedBlockHeight - 2)
		if err != nil {
			zap.S().Error(err)
			return nil, err
		}
		greatGrandParentTimestamp = greatGrandParent.Timestamp
	}

	fairPosActivated, err := storage.IsActiveAtHeight(int16(settings.FairPoS), confirmedBlockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "failed get fairPosActivated")
	}
	blockV5Activated, err := storage.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return nil, errors.Wrap(err, "failed get blockV5Activated")
	}
	var pos consensus.PosCalculator = &consensus.NxtPosCalculator{}
	if fairPosActivated {
		if blockV5Activated {
			pos = &consensus.FairPosCalculatorV2{}
		} else {
			pos = &consensus.FairPosCalculatorV1{}
		}
	}
	var gsp consensus.GenerationSignatureProvider = &consensus.NXTGenerationSignatureProvider{}
	if blockV5Activated {
		gsp = &consensus.VRFGenerationSignatureProvider{}
	}

	heightForHit := pos.HeightForHit(confirmedBlockHeight)

	zap.S().Debugf("Scheduler: topBlock: id %s, gensig: %s, topBlockHeight: %d", confirmedBlock.BlockID().String(), confirmedBlock.GenSignature, confirmedBlockHeight)

	var out []Emit
	for _, keyPair := range keyPairs {
		key := keyPair.Secret
		HitSourceAtHeight, err := storage.HitSourceAtHeight(heightForHit)
		if err != nil {
			zap.S().Error("scheduler, internalImpl", err)
			continue
		}
		genSig, err := gsp.GenerationSignature(key, HitSourceAtHeight)
		if err != nil {
			zap.S().Error("Scheduler: Failed to schedule mining: %v", err)
			continue
		}
		source, err := gsp.HitSource(key, HitSourceAtHeight)
		if err != nil {
			zap.S().Error("Scheduler: Failed to schedule mining: %v", err)
			continue
		}
		var vrf []byte = nil
		if blockV5Activated {
			vrf = source
		}
		hit, err := consensus.GenHit(source)
		if err != nil {
			zap.S().Error("Scheduler: Failed to schedule mining: %v", err)
			continue
		}

		addr, err := keyPair.Addr(schema)
		if err != nil {
			zap.S().Error("Scheduler: Failed to schedule mining: %v", err)
			continue
		}
		var startHeight proto.Height = 1
		if confirmedBlockHeight > 1000 {
			startHeight = confirmedBlockHeight - 1000
		}
		effectiveBalance, err := storage.EffectiveBalanceStable(proto.NewRecipientFromAddress(addr), startHeight, confirmedBlockHeight)
		if err != nil {
			zap.S().Debug("Scheduler: Failed to schedule mining for address '%s': %v", addr.String(), err)
			continue
		}

		delay, err := pos.CalculateDelay(hit, confirmedBlock.BlockHeader.BaseTarget, effectiveBalance)
		if err != nil {
			zap.S().Error("Scheduler: Failed to schedule mining: %v", err)
			continue
		}

		baseTarget, err := pos.CalculateBaseTarget(AverageBlockDelaySeconds, confirmedBlockHeight, confirmedBlock.BlockHeader.BaseTarget, confirmedBlock.Timestamp, greatGrandParentTimestamp, delay+confirmedBlock.Timestamp)
		if err != nil {
			zap.S().Error("Scheduler: Failed to schedule mining: %v", err)
			continue
		}

		out = append(out, Emit{
			Timestamp:    confirmedBlock.Timestamp + delay,
			KeyPair:      keyPair,
			GenSignature: genSig,
			VRF:          vrf,
			BaseTarget:   baseTarget,
			Parent:       confirmedBlock.BlockID(),
		})
	}
	return out, nil
}

func (a internalImpl) scheduleWithoutVrf(storage state.StateInfo, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) ([]Emit, error) {
	var greatGrandParentTimestamp proto.Timestamp = 0
	if confirmedBlockHeight > 2 {
		greatGrandParent, err := storage.BlockByHeight(confirmedBlockHeight - 2)
		if err != nil {
			zap.S().Error(err)
			return nil, err
		}
		greatGrandParentTimestamp = greatGrandParent.Timestamp
	}

	fairPosActivated, err := storage.IsActiveAtHeight(int16(settings.FairPoS), confirmedBlockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "failed get fairPosActivated")
	}
	blockV5Activated, err := storage.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return nil, errors.Wrap(err, "failed get blockV5Activated")
	}
	var pos consensus.PosCalculator = &consensus.NxtPosCalculator{}
	if fairPosActivated {
		if blockV5Activated {
			pos = &consensus.FairPosCalculatorV2{}
		} else {
			pos = &consensus.FairPosCalculatorV1{}
		}
	}
	var gsp consensus.GenerationSignatureProvider = &consensus.NXTGenerationSignatureProvider{}
	hitSourceHeader, err := storage.HeaderByHeight(pos.HeightForHit(confirmedBlockHeight))
	if err != nil {
		zap.S().Error("scheduler, internalImpl HeaderByHeight", err)
		return nil, err
	}

	zap.S().Infof("Scheduler: topBlock: id %s, gensig: %s, topBlockHeight: %d", confirmedBlock.BlockID().String(), confirmedBlock.GenSignature, confirmedBlockHeight)

	var out []Emit
	for _, keyPair := range keyPairs {
		genSigBlock := confirmedBlock.BlockHeader
		genSig, err := gsp.GenerationSignature(keyPair.Public, genSigBlock.GenSignature)
		if err != nil {
			zap.S().Error("scheduler, internalImpl", err)
			continue
		}
		source, err := gsp.HitSource(keyPair.Public, hitSourceHeader.GenSignature)
		if err != nil {
			zap.S().Error("scheduler, internalImpl HitSource", err)
			continue
		}
		var vrf []byte = nil
		hit, err := consensus.GenHit(source)
		if err != nil {
			zap.S().Error("scheduler, internalImpl GenHit", err)
			continue
		}

		addr, err := keyPair.Addr(schema)
		if err != nil {
			zap.S().Error("scheduler, internalImpl keyPair.Addr", err)
			continue
		}
		var startHeight proto.Height = 1
		if confirmedBlockHeight > 1000 {
			startHeight = confirmedBlockHeight - 1000
		}
		effectiveBalance, err := storage.EffectiveBalanceStable(proto.NewRecipientFromAddress(addr), startHeight, confirmedBlockHeight)
		if err != nil {
			zap.S().Debug("scheduler, internalImpl effectiveBalance, err", effectiveBalance, err, addr.String())
			continue
		}

		delay, err := pos.CalculateDelay(hit, confirmedBlock.BlockHeader.BaseTarget, effectiveBalance)
		if err != nil {
			zap.S().Error("scheduler, internalImpl pos.CalculateDelay", err)
			continue
		}

		baseTarget, err := pos.CalculateBaseTarget(AverageBlockDelaySeconds, confirmedBlockHeight, confirmedBlock.BlockHeader.BaseTarget, confirmedBlock.Timestamp, greatGrandParentTimestamp, delay+confirmedBlock.Timestamp)
		if err != nil {
			zap.S().Error("scheduler, internalImpl pos.CalculateBaseTarget", err)
			continue
		}

		out = append(out, Emit{
			Timestamp:    confirmedBlock.Timestamp + delay,
			KeyPair:      keyPair,
			GenSignature: genSig,
			VRF:          vrf,
			BaseTarget:   baseTarget,
			Parent:       confirmedBlock.BlockID(),
		})
	}
	return out, nil
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
		seeder:        seeder,
		mine:          make(chan Emit, 1),
		settings:      settings,
		internal:      internal,
		storage:       state,
		mu:            sync.Mutex{},
		tm:            tm,
		consensus:     consensus,
		outdatePeriod: minerDelay,
	}
}

func (a *SchedulerImpl) Mine() chan Emit {
	return a.mine
}

func (a *SchedulerImpl) Reschedule() {
	if len(a.seeder.Seeds()) == 0 {
		zap.S().Debug("Scheduler: Mining is not possible because no seeds registered")
		return
	}

	zap.S().Debugf("Scheduler: Trying to mine with %d seeds", len(a.seeder.Seeds()))

	if !a.consensus.IsMiningAllowed() {
		zap.S().Debug("Scheduler: Mining is not allowed because of lack of connected nodes")
		return
	}

	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.storage.TopBlock()
	if currentTimestamp-lastKnownBlock.Timestamp > a.outdatePeriod {
		zap.S().Debug("Scheduler: Mining is not allowed because blockchain is too old")
		return
	}

	h, err := a.storage.Height()
	if err != nil {
		zap.S().Error(err)
		return
	}

	block, err := a.storage.BlockByHeight(h)
	if err != nil {
		zap.S().Error(err)
		return
	}

	a.reschedule(block, h)
}

func (a *SchedulerImpl) reschedule(confirmedBlock *proto.Block, confirmedBlockHeight uint64) {
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

	rs, err := a.storage.MapR(func(info state.StateInfo) (i interface{}, err error) {
		return a.internal.schedule(info, keyPairs, a.settings.AddressSchemeCharacter, a.settings.AverageBlockDelaySeconds, confirmedBlock, confirmedBlockHeight)
	})
	if err != nil {
		zap.S().Error(err)
	}
	emits := rs.([]Emit)

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
					zap.S().Debug("Scheduler: cannot emit a.mine, chan is full")
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
