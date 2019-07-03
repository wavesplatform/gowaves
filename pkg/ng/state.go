package ng

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type State struct {
	storage        *storage
	prevAddedBlock *proto.Block
	applier        types.BlockApplier
	state          state.State
	mu             sync.Mutex
	historySync    types.StateHistorySynchronizer
}

func NewState(validator validator, applier types.BlockApplier, state state.State) *State {
	return &State{
		mu:      sync.Mutex{},
		storage: newStorage(validator),
		applier: applier,
		state:   state,
	}
}

func (a *State) AddBlock(block *proto.Block) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// same block
	if a.prevAddedBlock != nil && a.prevAddedBlock.BlockSignature == block.BlockSignature {
		return
	}

	err := a.storage.PushBlock(block)
	if err != nil {
		zap.S().Error(err)
		return
	}

	mu := a.state.Mutex()
	mu.Lock()
	err = a.state.RollbackTo(block.Parent)
	if err != nil {
		zap.S().Error(err)
		a.storage.Pop()
		mu.Unlock()
		return
	}
	mu.Unlock()

	err = a.applier.Apply(block)
	if err != nil {
		zap.S().Error(err)
		a.storage.Pop()

		// return prev block, if possible
		if a.prevAddedBlock != nil {
			err := a.applier.Apply(a.prevAddedBlock)
			if err != nil { // can't apply previous added block, maybe broken state
				zap.S().Error(err)
				a.historySync.Sync()
			}
		}
		return
	}
	a.prevAddedBlock = block
}

func (a *State) AddMicroblock(micro *proto.MicroBlock) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.prevAddedBlock == nil {
		return
	}

	err := a.storage.PushMicro(micro)
	if err != nil {
		return
	}

	block, err := a.storage.Block()
	if err != nil {
		zap.S().Error(err)
		return
	}

	if a.prevAddedBlock.Parent != block.Parent {
		zap.S().Errorf("parents not equal expected %q actual %q", a.prevAddedBlock.Parent, block.Parent)
		return
	}

	err = a.applier.Apply(block)
	if err != nil {
		zap.S().Error(err)
		// remove prev added micro
		a.storage.Pop()
		return
	}

	a.prevAddedBlock = block
}

func (a *State) BlockApplied(block *proto.Block) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.prevAddedBlock == nil {
		a.prevAddedBlock = block
		a.storage = a.storage.newFromBlock(block)
		return
	}

	if a.prevAddedBlock.BlockSignature == block.BlockSignature {
		return
	}

	a.prevAddedBlock = block
	a.storage = a.storage.newFromBlock(block)
}
