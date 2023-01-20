package scheduler

import (
	"sync"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"github.com/wavesplatform/gowaves/pkg/util/common"
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

type Default struct {
	seeder       seeder
	mine         chan Emit
	cancel       []func()
	settings     *settings.BlockchainSettings
	mu           sync.Mutex
	internal     internal
	emits        []Emit
	storage      state.State
	tm           types.Time
	consensus    types.MinerConsensus
	obsolescence time.Duration
}

type internal interface {
	schedule(state state.StateInfo, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, MinBlockTime float64, DelayDelta uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) ([]Emit, error)
}

type internalImpl struct {
}

func (a internalImpl) schedule(storage state.StateInfo, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, MinBlockTime float64, DelayDelta uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) ([]Emit, error) {
	vrfActivated, err := storage.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return nil, errors.Wrap(err, "failed get vrfActivated")
	}
	if vrfActivated {
		return a.scheduleWithVrf(storage, keyPairs, schema, AverageBlockDelaySeconds, MinBlockTime, DelayDelta, confirmedBlock, confirmedBlockHeight)
	}
	return a.scheduleWithoutVrf(storage, keyPairs, schema, AverageBlockDelaySeconds, MinBlockTime, DelayDelta, confirmedBlock, confirmedBlockHeight)
}

func (a internalImpl) scheduleWithVrf(storage state.StateInfo, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, MinBlockTime float64, DelayDelta uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) ([]Emit, error) {
	var greatGrandParentTimestamp proto.Timestamp = 0
	if confirmedBlockHeight > 2 {
		greatGrandParentHeight := confirmedBlockHeight - 2
		greatGrandParent, err := storage.BlockByHeight(greatGrandParentHeight)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to get blockID by height %d: %v", greatGrandParentHeight, err)
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
	pos := consensus.NXTPosCalculator
	if fairPosActivated {
		if blockV5Activated {
			pos = consensus.NewFairPosCalculator(DelayDelta, MinBlockTime)
		} else {
			pos = consensus.FairPosCalculatorV1
		}
	}
	gsp := consensus.NXTGenerationSignatureProvider
	if blockV5Activated {
		gsp = consensus.VRFGenerationSignatureProvider
	}

	heightForHit := pos.HeightForHit(confirmedBlockHeight)

	zap.S().Debugf("Scheduler: topBlock: id %s, gensig: %s, topBlockHeight: %d",
		confirmedBlock.BlockID().String(), confirmedBlock.GenSignature, confirmedBlockHeight,
	)

	var out []Emit
	for _, keyPair := range keyPairs {
		sk := keyPair.Secret
		hitSourceAtHeight, err := storage.HitSourceAtHeight(heightForHit)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to get hit source at height %d: %v", heightForHit, err)
			continue
		}
		genSig, err := gsp.GenerationSignature(sk, hitSourceAtHeight)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to schedule mining, can't get generation signature at height %d: %v",
				heightForHit, err,
			)
			continue
		}
		source, err := gsp.HitSource(sk, hitSourceAtHeight)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to schedule mining, failed to get hit source at height %d: %v",
				heightForHit, err,
			)
			continue
		}
		var vrf []byte
		if blockV5Activated {
			vrf = source
		}
		hit, err := consensus.GenHit(source)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to schedule mining, failed to generate hit from source: %v", err)
			continue
		}

		addr, err := keyPair.Addr(schema)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to schedule mining, failed to create address from PK: %v", err)
			continue
		}
		var startHeight proto.Height = 1
		if confirmedBlockHeight > 1000 {
			startHeight = confirmedBlockHeight - 1000 + 1
		}
		effectiveBalance, err := storage.EffectiveBalance(proto.NewRecipientFromAddress(addr), startHeight, confirmedBlockHeight)
		if err != nil {
			zap.S().Debugf(
				"Scheduler: Failed to schedule mining for address %q, failed to calculate effective balance with startHeight=%d: %v",
				addr.String(), startHeight, err)
			continue
		}

		delay, err := pos.CalculateDelay(hit, confirmedBlock.BlockHeader.BaseTarget, effectiveBalance)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to schedule mining for address %q, failed to calculate delay: %v", addr.String(), err)
			continue
		}

		baseTarget, err := pos.CalculateBaseTarget(
			AverageBlockDelaySeconds,
			confirmedBlockHeight,
			confirmedBlock.BlockHeader.BaseTarget,
			confirmedBlock.Timestamp,
			greatGrandParentTimestamp,
			delay+confirmedBlock.Timestamp,
		)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to schedule mining for address %q, failed to calculate base target: %v",
				addr.String(), err,
			)
			continue
		}
		zap.S().Debugf("Scheduled generation by address '%s' at %s", addr.String(),
			time.UnixMilli(int64(confirmedBlock.Timestamp+delay)).Format("2006-01-02 15:04:05.000 MST"))
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

func (a internalImpl) scheduleWithoutVrf(storage state.StateInfo, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, MinBlockTime float64, DelayDelta uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) ([]Emit, error) {
	var greatGrandParentTimestamp proto.Timestamp = 0
	if confirmedBlockHeight > 2 {
		greatGrandParentHeight := confirmedBlockHeight - 2
		greatGrandParent, err := storage.BlockByHeight(greatGrandParentHeight)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to get blockID by height %d: %v", greatGrandParentHeight, err)
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
	pos := consensus.NXTPosCalculator
	if fairPosActivated {
		if blockV5Activated {
			pos = consensus.NewFairPosCalculator(DelayDelta, MinBlockTime)
		} else {
			pos = consensus.FairPosCalculatorV1
		}
	}

	gsp := consensus.NXTGenerationSignatureProvider

	heightForHit := pos.HeightForHit(confirmedBlockHeight)
	hitSourceHeader, err := storage.HeaderByHeight(heightForHit)
	if err != nil {
		zap.S().Errorf("Scheduler: Failed to get header by height %d for hit: %v", heightForHit, err)
		return nil, err
	}

	zap.S().Debugf("Scheduling generation on top of block (%d) '%s'", confirmedBlockHeight, confirmedBlock.BlockID().String())
	zap.S().Debugf("  block timestamp: %d (%s)", confirmedBlock.Timestamp, common.UnixMillisToTime(int64(confirmedBlock.Timestamp)).String())
	zap.S().Debugf("  block base target: %d", confirmedBlock.BaseTarget)
	zap.S().Debug("Generation accounts:")
	var out []Emit
	for _, keyPair := range keyPairs {
		pk := keyPair.Public
		genSigBlock := confirmedBlock.BlockHeader
		genSig, err := gsp.GenerationSignature(pk, genSigBlock.GenSignature)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to get generation signature for PK %q: %v", pk.String(), err)
			continue
		}
		source, err := gsp.HitSource(pk, hitSourceHeader.GenSignature)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to generate hit source for PK %q: %v", pk.String(), err)
			continue
		}
		hit, err := consensus.GenHit(source)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to generate hit for PK %q: %v", pk.String(), err)
			continue
		}

		addr, err := proto.NewAddressFromPublicKey(schema, pk)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to create new address from PK %q: %v", pk.String(), err)
			continue
		}
		var startHeight proto.Height = 1
		if confirmedBlockHeight > 1000 {
			startHeight = confirmedBlockHeight - 1000 + 1
		}
		effectiveBalance, err := storage.EffectiveBalance(proto.NewRecipientFromAddress(addr), startHeight, confirmedBlockHeight)
		if err != nil {
			zap.S().Debugf("Scheduler: Failed to get effective balance for address %q from startHeight=%d: %v",
				addr.String(), startHeight, err,
			)
			continue
		}

		delay, err := pos.CalculateDelay(hit, confirmedBlock.BlockHeader.BaseTarget, effectiveBalance)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to calculate delay for address %q with effective balance %d: %v",
				addr, effectiveBalance, err,
			)
			continue
		}

		baseTarget, err := pos.CalculateBaseTarget(
			AverageBlockDelaySeconds,
			confirmedBlockHeight,
			confirmedBlock.BlockHeader.BaseTarget,
			confirmedBlock.Timestamp,
			greatGrandParentTimestamp,
			delay+confirmedBlock.Timestamp,
		)
		if err != nil {
			zap.S().Errorf("Scheduler: Failed to calculate base target for address %q: %v", addr.String(), err)
			continue
		}
		ts := confirmedBlock.Timestamp + delay
		zap.S().Debugf("  %s (%s): ", addr.String(), pk.String())
		zap.S().Debugf("    Hit: %s (%s)", hit.String(), base58.Encode(source))
		zap.S().Debugf("    Generation Balance: %d", int(effectiveBalance))
		zap.S().Debugf("    Delay: %d", int(delay))
		zap.S().Debugf("    Timestamp: %d (%s)", int(ts), common.UnixMillisToTime(int64(ts)).String())
		out = append(out, Emit{
			Timestamp:    ts,
			KeyPair:      keyPair,
			GenSignature: genSig,
			VRF:          nil, // because without VRF
			BaseTarget:   baseTarget,
			Parent:       confirmedBlock.BlockID(),
		})
	}
	return out, nil
}

type seeder interface {
	AccountSeeds() [][]byte
}

func NewScheduler(
	state state.State,
	seeder seeder,
	settings *settings.BlockchainSettings,
	tm types.Time,
	consensus types.MinerConsensus,
	minerDelay time.Duration) (*Default, error) {
	if minerDelay <= 0 {
		return nil, errors.New("minerDelay must be positive")
	}
	return newScheduler(internalImpl{}, state, seeder, settings, tm, consensus, minerDelay), nil
}

func newScheduler(internal internal, state state.State, seeder seeder, settings *settings.BlockchainSettings,
	tm types.Time, consensus types.MinerConsensus, minerDelay time.Duration) *Default {
	if seeder == nil {
		seeder = wallet.NewWallet()
	}
	return &Default{
		seeder:       seeder,
		mine:         make(chan Emit, 1),
		settings:     settings,
		internal:     internal,
		storage:      state,
		mu:           sync.Mutex{},
		tm:           tm,
		consensus:    consensus,
		obsolescence: minerDelay,
	}
}

func (a *Default) Mine() chan Emit {
	return a.mine
}

func (a *Default) Reschedule() {
	if len(a.seeder.AccountSeeds()) == 0 {
		zap.S().Debug("Scheduler: Mining is not possible because no seeds registered")
		return
	}

	zap.S().Debugf("Scheduler: Trying to mine with %d seeds", len(a.seeder.AccountSeeds()))

	if !a.consensus.IsMiningAllowed() {
		zap.S().Debug("Scheduler: Mining is not allowed because of lack of connected nodes")
		return
	}

	now := a.tm.Now()
	obsolescenceTime := now.Add(-a.obsolescence)
	lastBlock := a.storage.TopBlock()
	lastBlockTime := time.UnixMilli(int64(lastBlock.Timestamp))
	if obsolescenceTime.After(lastBlockTime) {
		zap.S().Debugf("Scheduler: Mining is not allowed because last block (ID: %s) time %s is before the obsolesence time %s",
			lastBlock.ID, lastBlockTime, obsolescenceTime)
		return
	}

	h, err := a.storage.Height()
	if err != nil {
		zap.S().Errorf("Scheduler: Failed to get state height: %v", err)
		return
	}

	block, err := a.storage.BlockByHeight(h)
	if err != nil {
		zap.S().Errorf("Scheduler: Failed to get block by height %d: %v", h, err)
		return
	}

	a.reschedule(block, h)
}

func (a *Default) reschedule(confirmedBlock *proto.Block, confirmedBlockHeight uint64) {
	if len(a.seeder.AccountSeeds()) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	// stop previous timeouts
	for _, cancel := range a.cancel {
		cancel()
	}
	a.cancel = nil
	a.emits = nil

	keyPairs, err := makeKeyPairs(a.seeder.AccountSeeds())
	if err != nil {
		zap.S().Errorf("Scheduler: Failed to make key pairs from seeds: %v", err)
		return
	}

	rs, err := a.storage.MapR(func(info state.StateInfo) (i interface{}, err error) {
		return a.internal.schedule(info, keyPairs, a.settings.AddressSchemeCharacter, a.settings.AverageBlockDelaySeconds, a.settings.MinBlockTime, a.settings.DelayDelta, confirmedBlock, confirmedBlockHeight)
	})
	if err != nil {
		zap.S().Errorf("Scheduler: Failed to schedule: %v", err)
	}
	emits := rs.([]Emit)

	a.emits = emits
	now := proto.NewTimestampFromTime(a.tm.Now())
	for _, emit := range emits {
		if emit.Timestamp > now { // timestamp in future
			timeout := emit.Timestamp - now
			emit_ := emit
			cancel := cancellable.After(time.Duration(timeout)*time.Millisecond, func() {
				// hack for integrations tests
				common.EnsureTimeout(a.tm, emit_.Timestamp)
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

func (a *Default) Emits() []Emit {
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
