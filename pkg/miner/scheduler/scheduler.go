package scheduler

import (
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"go.uber.org/zap"
)

type Emit struct {
	Timestamp            uint64
	KeyPair              proto.KeyPair
	GenSignature         crypto.Digest
	BaseTarget           consensus.BaseTarget
	ParentBlockSignature crypto.Signature
}

type SchedulerImpl struct {
	keyPairs []proto.KeyPair
	mine     chan Emit
	cancel   []func()
	settings *settings.BlockchainSettings
	mu       sync.Mutex
	internal internal
	emits    []Emit
	state    state.State
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

	var out []Emit
	for _, keyPair := range keyPairs {
		genSig, err := consensus.GeneratorSignature(confirmedBlock.BlockHeader.GenSignature, keyPair.Public())
		if err != nil {
			zap.S().Error(err)
			continue
		}

		hit, err := consensus.GenHit(genSig[:])
		if err != nil {
			zap.S().Error(err)
			continue
		}

		// TODO
		c := &consensus.NxtPosCalculator{}
		//c := &consensus.FairPosCalculator{}

		locked := state.Mutex().RLock()
		effectiveBalance, err := state.EffectiveBalance(keyPair.Addr(schema), confirmedBlockHeight-1000, confirmedBlockHeight)
		locked.Unlock()
		if err != nil {
			zap.S().Error(err)
			continue
		}

		delay, err := c.CalculateDelay(hit, confirmedBlock.BlockHeader.BaseTarget, effectiveBalance)
		if err != nil {
			zap.S().Error(err)
			continue
		}

		baseTarget, err := c.CalculateBaseTarget(AverageBlockDelaySeconds, confirmedBlockHeight, confirmedBlock.BlockHeader.BaseTarget, confirmedBlock.Timestamp, greatGrandParentTimestamp, delay+confirmedBlock.Timestamp)
		if err != nil {
			zap.S().Error(err)
			continue
		}

		out = append(out, Emit{
			Timestamp:            confirmedBlock.Timestamp + delay,
			KeyPair:              keyPair,
			GenSignature:         genSig,
			BaseTarget:           baseTarget,
			ParentBlockSignature: confirmedBlock.BlockSignature,
		})
	}
	return out
}

func NewScheduler(state state.State, pairs []proto.KeyPair, settings *settings.BlockchainSettings) *SchedulerImpl {
	return newScheduler(internalImpl{}, state, pairs, settings)
}

func newScheduler(internal internal, state state.State, pairs []proto.KeyPair, settings *settings.BlockchainSettings) *SchedulerImpl {
	return &SchedulerImpl{
		keyPairs: pairs,
		mine:     make(chan Emit, 1),
		settings: settings,
		internal: internal,
		state:    state,
		mu:       sync.Mutex{},
	}
}

func (a *SchedulerImpl) Mine() chan Emit {
	return a.mine
}

func (a *SchedulerImpl) Reschedule() {
	if len(a.keyPairs) == 0 {
		return
	}
	state := a.state

	mu := state.Mutex()
	locked := mu.RLock()

	h, err := state.Height()
	if err != nil {
		zap.S().Error(err)
		locked.Unlock()
		return
	}

	block, err := state.BlockByHeight(h)
	if err != nil {
		zap.S().Error(err)
		locked.Unlock()
		return
	}
	locked.Unlock()

	a.reschedule(state, block, h)
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

	now := proto.NewTimestampFromTime(time.Now())
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
				zap.S().Debug("cannot emit a.mine, chan is full")
			}

		}
	}
}

func (a *SchedulerImpl) Emits() []Emit {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.emits
}
