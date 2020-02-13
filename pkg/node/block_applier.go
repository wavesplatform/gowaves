package node

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type innerBlockApplier struct {
	state state.State
}

func (a *innerBlockApplier) apply(block *proto.Block) (*proto.Block, proto.Height, error) {
	// check if such block already exists
	_, err := a.state.Block(block.BlockSignature)
	if err == nil {
		return nil, 0, errors.Errorf("block %s exists", block.BlockSignature.String())
	}
	if !state.IsNotFound(err) {
		return nil, 0, errors.Wrap(err, "unknown error")
	}

	curHeight, err := a.state.Height()
	if err != nil {
		return nil, 0, err
	}
	curScore, err := a.state.ScoreAtHeight(curHeight)
	if err != nil {
		return nil, 0, err
	}

	// try to find parent. If not - we can't add block, skip it
	parentHeight, err := a.state.BlockIDToHeight(block.Parent)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "BlockApplier: failed get parent height, block sig %s, for block %s", block.Parent, block.BlockSignature)
	}

	// if new block has highest score apply it
	score, err := state.CalculateScore(block.NxtConsensus.BaseTarget)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed calculate score")
	}
	parentScore, err := a.state.ScoreAtHeight(parentHeight)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed get score at %d", parentHeight)
	}
	sumScore := score.Add(score, parentScore)
	if curScore.Cmp(sumScore) > 0 { // current height is higher
		return nil, 0, errors.New("BlockApplier: low score: current score is higher than block")
	}

	// so, new block has highest score, try apply it.
	// Do we need to rollback blocks?
	if parentHeight == curHeight {
		// no, don't rollback, just add block
		newBlock, err := a.state.AddDeserializedBlock(block)
		if err != nil {
			return nil, 0, err
		}
		return newBlock, curHeight + 1, nil
	}

	deltaHeight := curHeight - parentHeight
	if deltaHeight > 100 { // max number that we can rollback
		return nil, 0, errors.Errorf("can't apply new block, rollback more than 100 block, %d", deltaHeight)
	}

	// save previously added blocks. If new block failed to add, then return them back
	blocks := make([]*proto.Block, 0, deltaHeight)
	for i := proto.Height(1); i <= deltaHeight; i++ {
		block, err := a.state.BlockByHeight(parentHeight + i)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed to get block by height %d", parentHeight+i)
		}
		blocks = append(blocks, block)
	}

	err = a.state.RollbackToHeight(parentHeight)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to rollback to height %d", parentHeight)
	}

	newBlock, err := a.state.AddDeserializedBlock(block)
	if err != nil {
		// return back saved blocks
		_, err2 := a.state.AddNewDeserializedBlocks(blocks)
		if err2 != nil {
			return nil, 0, errors.Wrap(err2, "failed add new deserialized blocks")
		}
		return nil, 0, errors.Wrapf(err, "failed add deserialized block %q", block.BlockSignature)
	}

	return newBlock, parentHeight + 1, nil
}

type BlockApplier struct {
	state              state.State
	scheduler          types.Scheduler
	inner              innerBlockApplier
	blockAddedNotifier types.Handler
}

func NewBlockApplier(state state.State, blockAddedNotifier types.Handler, scheduler types.Scheduler) *BlockApplier {
	return &BlockApplier{
		state:              state,
		scheduler:          scheduler,
		blockAddedNotifier: blockAddedNotifier,

		inner: innerBlockApplier{
			state: state,
		},
	}
}

func (a *BlockApplier) ApplyBytes(b []byte) error {
	block := &proto.Block{}
	err := block.UnmarshalBinary(b)
	if err != nil {
		return err
	}
	return a.Apply(block)
}

// 1) interrupt miner
// 2) notify peers about score
// 3) reshedule
func (a *BlockApplier) Apply(block *proto.Block) error {
	m := a.state.Mutex()
	locked := m.Lock()

	_, _, err := a.inner.apply(block)
	if err != nil {
		locked.Unlock()
		return err
	}
	locked.Unlock()
	go a.blockAddedNotifier.Handle()
	return nil
}
