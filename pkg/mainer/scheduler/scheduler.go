package scheduler

import (
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"go.uber.org/zap"
	"sync"
	"time"
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
}

type internal interface {
	schedule(state state.State, keyPairs []proto.KeyPair, schema proto.Schema, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) []Emit
}

type internalImpl struct {
}

func (a internalImpl) schedule(state state.State, keyPairs []proto.KeyPair, schema proto.Schema, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) []Emit {

	var greatGrandParentTimestamp proto.Timestamp = 0
	if confirmedBlockHeight > 2 {
		greatGrandParent, err := state.BlockByHeight(confirmedBlockHeight - 2)
		if err != nil {
			zap.S().Error(err)
			return nil
		}
		greatGrandParentTimestamp = greatGrandParent.Timestamp
	}

	out := []Emit{}
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

		c := &consensus.NxtPosCalculator{}
		//c := &consensus.FairPosCalculator{}

		effectiveBalance, err := state.EffectiveBalance(keyPair.Addr(schema), confirmedBlockHeight-1000, confirmedBlockHeight)
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

		//now := proto.NewTimestampFromTime(time.Now())
		//if confirmedBlock.Timestamp+delay > now { // timestamp in future
		// delta from now to future
		//timeout := confirmedBlock.Timestamp + delay - now
		// start timeout before emit mine
		//keyPair_ := keyPair // ensure passed correct value
		//cancel := cancellable.After(time.Duration(timeout)*time.Millisecond, func() {
		out = append(out, Emit{
			Timestamp:            confirmedBlock.Timestamp + delay,
			KeyPair:              keyPair,
			GenSignature:         genSig,
			BaseTarget:           baseTarget,
			ParentBlockSignature: confirmedBlock.BlockSignature,
		})
		//})
		//a.cancel = append(a.cancel, cancel)
		//} else {
		//	out = <- Emit{
		//		Timestamp:            confirmedBlock.Timestamp + delay,
		//		KeyPair:              keyPair,
		//		GenSignature:         genSig,
		//		BaseTarget:           baseTarget,
		//		ParentBlockSignature: confirmedBlock.BlockSignature,
		//	}
		//}
	}
	return out
}

func NewScheduler(pairs []proto.KeyPair, settings *settings.BlockchainSettings) *SchedulerImpl {
	return &SchedulerImpl{
		keyPairs: pairs,
		mine:     make(chan Emit),
		settings: settings,
		internal: internalImpl{},
	}
}

func newScheduler(internal internal, pairs []proto.KeyPair, settings *settings.BlockchainSettings) *SchedulerImpl {
	return &SchedulerImpl{
		keyPairs: pairs,
		mine:     make(chan Emit),
		settings: settings,
		internal: internal,
	}
}

func (a *SchedulerImpl) Mine() chan Emit {
	return a.mine
}

func (a *SchedulerImpl) Init(state state.State) {
	mu := state.Mutex()
	mu.RLock()

	h, err := state.Height()
	if err != nil {
		zap.S().Error(err)
		mu.RUnlock()
		return
	}

	block, err := state.BlockByHeight(h)
	if err != nil {
		zap.S().Error(err)
		mu.RUnlock()
		return
	}

	mu.RUnlock()
	go a.Reschedule(state, block, h)
}

func (a *SchedulerImpl) Reschedule(state state.State, confirmedBlock *proto.Block, confirmedBlockHeight uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.keyPairs) == 0 {
		return
	}

	// stop previous timeouts
	for _, cancel := range a.cancel {
		cancel()
	}
	a.cancel = nil

	emits := a.internal.schedule(state, a.keyPairs, a.settings.Type, a.settings.AverageBlockDelaySeconds, confirmedBlock, confirmedBlockHeight)
	a.emits = emits

	now := proto.NewTimestampFromTime(time.Now())
	for _, emit := range emits {
		if emit.Timestamp > now { // timestamp in future
			timeout := emit.Timestamp - now
			emit_ := emit
			cancel := cancellable.After(time.Duration(timeout)*time.Millisecond, func() {
				a.mine <- emit_
			})
			a.cancel = append(a.cancel, cancel)
		} else {
			a.mine <- emit
		}
	}

	//if confirmedBlock.Timestamp+delay > now { // timestamp in future
	// delta from now to future
	//timeout := confirmedBlock.Timestamp + delay - now
	// start timeout before emit mine
	//keyPair_ := keyPair // ensure passed correct value
	//cancel := cancellable.After(time.Duration(timeout)*time.Millisecond, func() {
	//out = append(out, Emit{
	//	Timestamp:            confirmedBlock.Timestamp + delay,
	//	KeyPair:              keyPair,
	//	GenSignature:         genSig,
	//	BaseTarget:           baseTarget,
	//	ParentBlockSignature: confirmedBlock.BlockSignature,
	//})
	//})
	//a.cancel = append(a.cancel, cancel)
	//} else {
	//	out = <- Emit{
	//		Timestamp:            confirmedBlock.Timestamp + delay,
	//		KeyPair:              keyPair,
	//		GenSignature:         genSig,
	//		BaseTarget:           baseTarget,
	//		ParentBlockSignature: confirmedBlock.BlockSignature,
	//	}
	//}

	////
	//parentBlock, err := state.BlockByHeight(height - 1)
	//if err != nil {
	//	zap.S().Error(err)
	//	return
	//}

}
