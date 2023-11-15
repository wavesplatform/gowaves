package node

import (
	"math/big"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

const maxRollbackLength = 100

type applier struct {
	st state.State
}

func newApplier(st state.State) *applier {
	return &applier{st: st}
}
func (a *applier) exists(block *proto.Block) (bool, error) {
	_, err := a.st.Block(block.BlockID())
	if err == nil {
		return true, nil
	}
	if state.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func (a *applier) applyBlocks(blocks []*proto.Block) (*proto.Block, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no blocks to apply")
	}
	firstBlock := blocks[0]
	// check first block if exists
	_, err := a.st.Block(firstBlock.BlockID())
	if err == nil {
		return nil, errors.Errorf("first block '%s' alredy exists", firstBlock.BlockID().String())
	}
	if !state.IsNotFound(err) {
		return nil, err
	}
	currentHeight, err := a.st.Height()
	if err != nil {
		return nil, err
	}
	// current score. Main idea is to find parent block, and check if score
	// of all passed blocks higher than currentScore. If yes, we can add blocks
	currentScore, err := a.st.ScoreAtHeight(currentHeight)
	if err != nil {
		return nil, err
	}
	// try to find parent. If not - we can't add blocks, skip it
	parentHeight, err := a.st.BlockIDToHeight(firstBlock.Parent)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get height of parent ID '%s' of block '%s'",
			firstBlock.Parent.String(), firstBlock.BlockID().String())
	}
	// calculate score of all passed blocks
	forkScore, err := calcMultipleScore(blocks)
	if err != nil {
		return nil, errors.Wrap(err, "failed calculate score of passed blocks")
	}
	parentScore, err := a.st.ScoreAtHeight(parentHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "failed get score at %d", parentHeight)
	}
	cumulativeScore := forkScore.Add(forkScore, parentScore)
	if currentScore.Cmp(cumulativeScore) >= 0 {
		// current score is higher or the same as fork score - do not apply blocks
		return nil, errors.Errorf(
			"low fork score: current blockchain score (%s) is higher than or equal to fork's score (%s)",
			currentScore.String(), cumulativeScore.String())
	}

	// so, new blocks has higher score, try to apply it.
	// Do we need rollback?
	if parentHeight == currentHeight {
		// no, don't rollback, just add blocks
		return a.st.AddDeserializedBlocks(blocks)
	}
	rollbackBlocks, err := a.rollback(parentHeight, currentHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to rollback")
	}
	// applying new blocks
	b, err := a.st.AddDeserializedBlocks(blocks)
	if err != nil {
		// return back saved blocks
		_, err2 := a.st.AddDeserializedBlocks(rollbackBlocks)
		if err2 != nil {
			return nil, errors.Wrap(err2, "failed rollback deserialized blocks")
		}
		return nil, errors.Wrapf(err, "failed add deserialized blocks, first block id %s",
			firstBlock.BlockID().String())
	}
	return b, nil
}

func (a *applier) rollback(parentHeight, currentHeight uint64) ([]*proto.Block, error) {
	count := currentHeight - parentHeight
	if count > maxRollbackLength {
		return nil, errors.Errorf("attempt to rollback on %d blocks, while only up to %d is allowed",
			count, maxRollbackLength)
	}
	// save previously added blocks. If new firstBlock failed to add, then return them back
	backup := make([]*proto.Block, count)
	for i := uint64(0); i < count; i++ {
		h := parentHeight + i + 1
		b, err := a.st.BlockByHeight(h)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get block at height %d", h)
		}
		backup[i] = b
	}
	if err := a.st.RollbackToHeight(parentHeight); err != nil {
		return nil, errors.Wrapf(err, "rollback to height %d failed", parentHeight)
	}
	return backup, nil
}

func (a *applier) applyMicroBlock(block *proto.Block) (proto.Height, error) {
	_, err := a.st.Block(block.BlockID())
	if err == nil {
		return 0, errors.Errorf("block '%s' already exist", block.BlockID().String())
	}
	if !state.IsNotFound(err) {
		return 0, errors.Wrap(err, "unexpected error")
	}

	currentHeight, err := a.st.Height()
	if err != nil {
		return 0, err
	}
	parentHeight, err := a.st.BlockIDToHeight(block.Parent)
	if err != nil {
		return 0, errors.Wrapf(err, "failed get height of parent block '%s'", block.Parent.String())
	}

	if currentHeight-parentHeight != 1 {
		return 0, errors.Errorf("invalid parent height %d", parentHeight)
	}

	currentBlock, err := a.st.BlockByHeight(currentHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get current block by height %d", currentHeight)
	}

	err = a.st.RollbackToHeight(parentHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to rollback to height %d", parentHeight)
	}

	// applying new blocks
	_, err = a.st.AddDeserializedBlocks([]*proto.Block{block})
	if err != nil {
		// return back saved blocks
		_, err2 := a.st.AddDeserializedBlocks([]*proto.Block{currentBlock})
		if err2 != nil {
			return 0, errors.Wrap(err2, "failed rollback block")
		}
		return 0, errors.Wrapf(err, "failed apply new block '%s'", block.BlockID().String())
	}
	return currentHeight, nil
}

func calcMultipleScore(blocks []*proto.Block) (*big.Int, error) {
	score := big.NewInt(0)
	for _, block := range blocks {
		s, err := state.CalculateScore(block.NxtConsensus.BaseTarget)
		if err != nil {
			return nil, errors.Wrap(err, "failed calculate score")
		}
		score = score.Add(score, s)
	}
	return score, nil
}
