package blocks_applier

import (
	stderrors "errors"
	"math/big"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

const maxRollbackDeltaHeight = 100

type innerBlocksApplier struct {
}

type innerState interface {
	Block(blockID proto.BlockID) (*proto.Block, error)
	Height() (proto.Height, error)
	ScoreAtHeight(height proto.Height) (*big.Int, error)
	BlockIDToHeight(blockID proto.BlockID) (proto.Height, error)
	AddDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error)
	AddDeserializedBlocksWithSnapshots(blocks []*proto.Block, snapshots []*proto.BlockSnapshot) (*proto.Block, error)
	BlockByHeight(height proto.Height) (*proto.Block, error)
	RollbackToHeight(height proto.Height) error
	SnapshotsAtHeight(height proto.Height) (proto.BlockSnapshot, error)
}

func (a *innerBlocksApplier) exists(storage innerState, block *proto.Block) (bool, error) {
	_, err := storage.Block(block.BlockID())
	if err == nil {
		return true, nil
	}
	if state.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func (a *innerBlocksApplier) apply(
	storage innerState,
	blocks []*proto.Block,
) (proto.Height, error) {
	if len(blocks) == 0 {
		return 0, errors.New("empty blocks")
	}
	currentHeight, parentHeight, err := a.getParentAndCurrentHeight(storage, blocks)
	if err != nil {
		return 0, err
	}

	// so, new blocks has higher score, try to apply it.
	// Do we need rollback?
	if parentHeight == currentHeight {
		// no, don't rollback, just add blocks
		_, err = storage.AddDeserializedBlocks(blocks)
		if err != nil {
			return 0, err
		}
		return currentHeight + proto.Height(len(blocks)), nil
	}

	deltaHeight := currentHeight - parentHeight
	if deltaHeight > maxRollbackDeltaHeight { // max number that we can rollback
		return 0, errors.Errorf(
			"can't apply new blocks, rollback more than %d blocks, %d", maxRollbackDeltaHeight, deltaHeight)
	}

	// save previously added blocks. If new firstBlock failed to add, then return them back
	rollbackBlocks, err := a.getRollbackBlocks(storage, deltaHeight, parentHeight)
	if err != nil {
		return 0, err
	}

	err = storage.RollbackToHeight(parentHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to rollback to height %d", parentHeight)
	}
	// applying new blocks
	_, err = storage.AddDeserializedBlocks(blocks)
	if err != nil {
		// return back saved blocks
		_, err2 := storage.AddDeserializedBlocks(rollbackBlocks)
		if err2 != nil {
			return 0, errors.Wrap(err2, "failed rollback deserialized blocks")
		}

		return 0, errors.Wrapf(stderrors.Join(err, err2),
			"failed add deserialized blocks, first block id %s", blocks[0].BlockID().String())
	}
	return parentHeight + proto.Height(len(blocks)), nil
}

func (a *innerBlocksApplier) getParentAndCurrentHeight(
	storage innerState,
	blocks []*proto.Block,
) (proto.Height, proto.Height, error) {
	firstBlock := blocks[0]
	// check first block if exists
	_, err := storage.Block(firstBlock.BlockID())
	if err == nil {
		return 0, 0, proto.NewInfoMsg(errors.Errorf("first block %s exists", firstBlock.BlockID().String()))
	}
	if !state.IsNotFound(err) {
		return 0, 0, errors.Wrap(err, "unknown error")
	}
	currentHeight, err := storage.Height()
	if err != nil {
		return 0, 0, err
	}
	// current score. Main idea is to find parent block, and check if score
	// of all passed blocks higher than currentScore. If yes, we can add blocks
	currentScore, err := storage.ScoreAtHeight(currentHeight)
	if err != nil {
		return 0, 0, err
	}
	// try to find parent. If not - we can't add blocks, skip it
	parentHeight, err := storage.BlockIDToHeight(firstBlock.Parent)
	if err != nil {
		return 0, 0, proto.NewInfoMsg(errors.Wrapf(err, "failed get parent height, firstBlock id %s, for firstBlock %s",
			firstBlock.Parent.String(), firstBlock.BlockID().String()))
	}
	// calculate score of all passed blocks
	forkScore, err := calcMultipleScore(blocks)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed calculate score of passed blocks")
	}
	parentScore, err := storage.ScoreAtHeight(parentHeight)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "failed get score at %d", parentHeight)
	}
	cumulativeScore := forkScore.Add(forkScore, parentScore)
	if currentScore.Cmp(cumulativeScore) >= 0 { // current score is higher or the same as fork score - do not apply blocks
		return 0, 0, proto.NewInfoMsg(errors.Errorf(
			"low fork score: current blockchain score (%s) is higher than or equal to fork's score (%s)",
			currentScore.String(), cumulativeScore.String()))
	}
	return currentHeight, parentHeight, nil
}

func (a *innerBlocksApplier) applyWithSnapshots(
	storage innerState,
	blocks []*proto.Block,
	snapshots []*proto.BlockSnapshot,
) (proto.Height, error) {
	if len(blocks) == 0 {
		return 0, errors.New("empty blocks")
	}
	currentHeight, parentHeight, err := a.getParentAndCurrentHeight(storage, blocks)
	if err != nil {
		return 0, err
	}

	// so, new blocks has higher score, try to apply it.
	// Do we need rollback?
	if parentHeight == currentHeight {
		// no, don't rollback, just add blocks
		_, err = storage.AddDeserializedBlocksWithSnapshots(blocks, snapshots)
		if err != nil {
			return 0, err
		}
		return currentHeight + proto.Height(len(blocks)), nil
	}

	deltaHeight := currentHeight - parentHeight
	if deltaHeight > maxRollbackDeltaHeight { // max number that we can rollback
		return 0, errors.Errorf(
			"can't apply new blocks, rollback more than %d blocks, %d", maxRollbackDeltaHeight, deltaHeight)
	}

	// save previously added blocks. If new firstBlock failed to add, then return them back
	rollbackBlocks, rollbackBlocksSnapshots, err := a.getRollbackBlocksAndSnapshots(storage, deltaHeight, parentHeight)
	if err != nil {
		return 0, err
	}

	err = storage.RollbackToHeight(parentHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to rollback to height %d", parentHeight)
	}
	// applying new blocks
	_, err = storage.AddDeserializedBlocksWithSnapshots(blocks, snapshots)
	if err != nil {
		// return back saved blocks
		_, errDeserialized := storage.AddDeserializedBlocksWithSnapshots(rollbackBlocks, rollbackBlocksSnapshots)
		if errDeserialized != nil {
			return 0, errors.Wrap(errDeserialized, "failed rollback deserialized blocks")
		}
		return 0, errors.Wrapf(stderrors.Join(err, errDeserialized),
			"failed add deserialized blocks, first block id %s", blocks[0].BlockID().String())
	}
	return parentHeight + proto.Height(len(blocks)), nil
}

func (a *innerBlocksApplier) getRollbackBlocksAndSnapshots(
	storage innerState,
	deltaHeight proto.Height,
	parentHeight proto.Height,
) ([]*proto.Block, []*proto.BlockSnapshot, error) {
	rollbackBlocks := make([]*proto.Block, 0, deltaHeight)
	rollbackBlocksSnapshots := make([]*proto.BlockSnapshot, 0, deltaHeight)
	for i := proto.Height(1); i <= deltaHeight; i++ {
		block, err := storage.BlockByHeight(parentHeight + i)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get firstBlock by height %d", parentHeight+i)
		}
		rollbackBlocks = append(rollbackBlocks, block)

		snapshot, err := storage.SnapshotsAtHeight(parentHeight + i)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get snapshot by height %d", parentHeight+i)
		}
		rollbackBlocksSnapshots = append(rollbackBlocksSnapshots, &snapshot)
	}
	return rollbackBlocks, rollbackBlocksSnapshots, nil
}

func (a *innerBlocksApplier) getRollbackBlocks(
	storage innerState,
	deltaHeight proto.Height,
	parentHeight proto.Height,
) ([]*proto.Block, error) {
	rollbackBlocks := make([]*proto.Block, 0, deltaHeight)
	for i := proto.Height(1); i <= deltaHeight; i++ {
		block, err := storage.BlockByHeight(parentHeight + i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get firstBlock by height %d", parentHeight+i)
		}
		rollbackBlocks = append(rollbackBlocks, block)
	}
	return rollbackBlocks, nil
}

func (a *innerBlocksApplier) applyMicro(
	storage innerState,
	block *proto.Block,
) (proto.Height, error) {
	_, err := storage.Block(block.BlockID())
	if err == nil {
		return 0, errors.Errorf("block '%s' already exist", block.BlockID().String())
	}
	if !state.IsNotFound(err) {
		return 0, errors.Wrap(err, "unexpected error")
	}

	currentHeight, err := storage.Height()
	if err != nil {
		return 0, err
	}
	parentHeight, err := storage.BlockIDToHeight(block.Parent)
	if err != nil {
		return 0, errors.Wrapf(err, "failed get height of parent block '%s'", block.Parent.String())
	}

	if currentHeight-parentHeight != 1 {
		return 0, errors.Errorf("invalid parent height %d", parentHeight)
	}

	currentBlock, err := storage.BlockByHeight(currentHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get current block by height %d", currentHeight)
	}

	err = storage.RollbackToHeight(parentHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to rollback to height %d", parentHeight)
	}

	// applying new blocks
	_, err = storage.AddDeserializedBlocks([]*proto.Block{block})
	if err != nil {
		// return back saved blocks
		_, errAdd := storage.AddDeserializedBlocks([]*proto.Block{currentBlock})
		if errAdd != nil {
			return 0, errors.Wrap(errAdd, "failed rollback block")
		}
		return 0, errors.Wrapf(stderrors.Join(err, errAdd),
			"failed apply new block '%s'", block.BlockID().String())
	}
	return currentHeight, nil
}

func (a *innerBlocksApplier) applyMicroWithSnapshot(
	storage innerState,
	block *proto.Block,
	snapshot *proto.BlockSnapshot,
) (proto.Height, error) {
	_, err := storage.Block(block.BlockID())
	if err == nil {
		return 0, errors.Errorf("block '%s' already exist", block.BlockID().String())
	}
	if !state.IsNotFound(err) {
		return 0, errors.Wrap(err, "unexpected error")
	}

	currentHeight, err := storage.Height()
	if err != nil {
		return 0, err
	}
	parentHeight, err := storage.BlockIDToHeight(block.Parent)
	if err != nil {
		return 0, errors.Wrapf(err, "failed get height of parent block '%s'", block.Parent.String())
	}

	if currentHeight-parentHeight != 1 {
		return 0, errors.Errorf("invalid parent height %d", parentHeight)
	}

	currentBlock, err := storage.BlockByHeight(currentHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get current block by height %d", currentHeight)
	}
	curSnapshot, errSAT := storage.SnapshotsAtHeight(currentHeight)
	if errSAT != nil {
		return 0, errSAT
	}
	currentSnapshotsToApply := []*proto.BlockSnapshot{&curSnapshot}

	err = storage.RollbackToHeight(parentHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to rollback to height %d", parentHeight)
	}

	// applying new blocks
	_, err = storage.AddDeserializedBlocksWithSnapshots([]*proto.Block{block}, []*proto.BlockSnapshot{snapshot})
	if err != nil {
		// return back saved blocks
		_, errAdd := storage.AddDeserializedBlocksWithSnapshots(
			[]*proto.Block{currentBlock},
			currentSnapshotsToApply,
		)
		if errAdd != nil {
			return 0, errors.Wrap(errAdd, "failed rollback block")
		}
		return 0, errors.Wrapf(stderrors.Join(err, errAdd),
			"failed apply new block '%s'", block.BlockID().String())
	}
	return currentHeight, nil
}

type BlocksApplier struct {
	inner innerBlocksApplier
}

func NewBlocksApplier() *BlocksApplier {
	return &BlocksApplier{
		inner: innerBlocksApplier{},
	}
}

func (a *BlocksApplier) BlockExists(state state.State, block *proto.Block) (bool, error) {
	return a.inner.exists(state, block)
}

func (a *BlocksApplier) Apply(
	state state.State,
	blocks []*proto.Block,
) (proto.Height, error) {
	return a.inner.apply(state, blocks)
}

func (a *BlocksApplier) ApplyMicro(
	state state.State,
	block *proto.Block,
) (proto.Height, error) {
	return a.inner.applyMicro(state, block)
}

func (a *BlocksApplier) ApplyWithSnapshots(
	state state.State,
	blocks []*proto.Block,
	snapshots []*proto.BlockSnapshot,
) (proto.Height, error) {
	return a.inner.applyWithSnapshots(state, blocks, snapshots)
}

func (a *BlocksApplier) ApplyMicroWithSnapshots(
	state state.State,
	block *proto.Block,
	snapshot *proto.BlockSnapshot,
) (proto.Height, error) {
	return a.inner.applyMicroWithSnapshot(state, block, snapshot)
}

func calcMultipleScore(blocks []*proto.Block) (*big.Int, error) {
	score := big.NewInt(0)
	for _, block := range blocks {
		s, err := state.CalculateScore(block.BaseTarget)
		if err != nil {
			return nil, errors.Wrap(err, "failed calculate score")
		}
		score = score.Add(score, s)
	}
	return score, nil
}
