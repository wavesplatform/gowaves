package blocks_applier

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	//"github.com/wavesplatform/gowaves/pkg/types"
)

type innerBlocksApplier struct {
	//state state.State
}

func (a *innerBlocksApplier) apply(storage state.State, blocks []*proto.Block) (*proto.Block, proto.Height, error) {
	if len(blocks) == 0 {
		return nil, 0, errors.New("empty blocks")
	}
	firstBlock := blocks[0]
	// check first block if exists
	_, err := storage.Block(firstBlock.BlockSignature)
	if err == nil {
		return nil, 0, errors.Errorf("first block %s exists", firstBlock.BlockSignature.String())
	}
	if !state.IsNotFound(err) {
		return nil, 0, errors.Wrap(err, "unknown error")
	}
	curHeight, err := storage.Height()
	if err != nil {
		return nil, 0, err
	}
	// current score. Main idea is to find parent block, and check if score
	// of all passed blocks higher than curScore. If yes, we can add blocks
	curScore, err := storage.ScoreAtHeight(curHeight)
	if err != nil {
		return nil, 0, err
	}

	// try to find parent. If not - we can't add blocks, skip it
	parentHeight, err := storage.BlockIDToHeight(firstBlock.Parent)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "BlockApplier: failed get parent height, firstBlock sig %s, for firstBlock %s", firstBlock.Parent, firstBlock.BlockSignature)
	}
	// calculate score of all passed blocks
	score, err := calcMultipleScore(blocks)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed calculate score of passed blocks")
	}
	parentScore, err := storage.ScoreAtHeight(parentHeight)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed get score at %d", parentHeight)
	}
	sumScore := score.Add(score, parentScore)
	if curScore.Cmp(sumScore) > 0 { // current height is higher
		return nil, 0, errors.New("BlockApplier: low score: current score is higher than firstBlock")
	}

	// so, new blocks has higher score, try apply it.
	// Do we need rollback?
	if parentHeight == curHeight {
		// no, don't rollback, just add blocks
		newBlock, err := storage.AddNewDeserializedBlocks(blocks)
		if err != nil {
			return nil, 0, err
		}
		return newBlock, curHeight + proto.Height(len(blocks)), nil
	}

	deltaHeight := curHeight - parentHeight
	if deltaHeight > 100 { // max number that we can rollback
		return nil, 0, errors.Errorf("can't apply new blocks, rollback more than 100 blocks, %d", deltaHeight)
	}

	// save previously added blocks. If new firstBlock failed to add, then return them back
	rollbackBlocks := make([]*proto.Block, 0, deltaHeight)
	for i := proto.Height(1); i <= deltaHeight; i++ {
		block, err := storage.BlockByHeight(parentHeight + i)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed to get firstBlock by height %d", parentHeight+i)
		}
		rollbackBlocks = append(rollbackBlocks, block)
	}

	err = storage.RollbackToHeight(parentHeight)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to rollback to height %d", parentHeight)
	}
	// applying new blocks
	newBlock, err := storage.AddNewDeserializedBlocks(blocks)
	if err != nil {
		// return back saved blocks
		_, err2 := storage.AddNewDeserializedBlocks(rollbackBlocks)
		if err2 != nil {
			return nil, 0, errors.Wrap(err2, "failed rollback deserialized blocks")
		}
		return nil, 0, errors.Wrapf(err, "failed add deserialized blocks, first block sig %q", firstBlock.BlockSignature)
	}
	return newBlock, parentHeight + proto.Height(len(blocks)), nil
}

type BlocksApplier struct {
	//state state.State
	inner innerBlocksApplier
	//tm    types.Time
}

func NewBlocksApplier() *BlocksApplier {
	return &BlocksApplier{
		//state: state,
		//tm:    tm,
		inner: innerBlocksApplier{
			//state: state,
		},
	}
}

// 1) notify peers about score
// 2) reshedule
func (a *BlocksApplier) Apply(state state.State, blocks []*proto.Block) error {
	//locked := a.state.Mutex().Lock()
	//defer locked.Unlock()

	_, _, err := a.inner.apply(state, blocks)
	if err != nil {
		return err
	}
	// TODO extended api
	//if err := maybeEnableExtendedApi(a.state, lastBlock, proto.NewTimestampFromTime(a.tm.Now())); err != nil {
	//	panic(fmt.Sprintf("[*] BlockDownloader: MaybeEnableExtendedApi(): %v. Failed to persist address transactions for API after successfully applying valid blocks.", err))
	//}
	return nil
}

func calcMultipleScore(blocks []*proto.Block) (*big.Int, error) {
	score, err := state.CalculateScore(blocks[0].NxtConsensus.BaseTarget)
	if err != nil {
		return nil, errors.Wrap(err, "failed calculate score")
	}
	for _, block := range blocks[1:] {
		s, err := state.CalculateScore(block.NxtConsensus.BaseTarget)
		if err != nil {
			return nil, errors.Wrap(err, "failed calculate score")
		}
		score.Add(score, s)
	}
	return score, nil
}
