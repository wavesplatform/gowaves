package node

import (
	"math/big"

	"github.com/pkg/errors"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type BlockWithBytes struct {
	Block *proto.Block
	Bytes []byte
}

type innerBlockApplier struct {
	state state.State
}

func (a *innerBlockApplier) apply(bb BlockWithBytes) (*proto.Block, proto.Height, error) {
	block := bb.Block
	// check if such block already exists
	_, err := a.state.Block(block.BlockSignature)
	if err == nil {
		return nil, 0, errors.New("block exists")
	}
	if !state.IsNotFound(err) {
		return nil, 0, err
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
		return nil, 0, err
	}

	// if new block has highest score apply it
	score, err := state.CalculateScore(block.NxtConsensus.BaseTarget)
	if err != nil {
		return nil, 0, err
	}
	parentScore, err := a.state.ScoreAtHeight(parentHeight)
	if err != nil {
		return nil, 0, err
	}
	sumScore := score.Add(score, parentScore)
	if curScore.Cmp(sumScore) >= 0 { // same height, or current height is higher
		return nil, 0, errors.New("low score")
	}

	// so, new block has highest score, try apply it.
	// Do we need to rollback blocks?
	if parentHeight == curHeight {
		// no, don't rollback, just add block
		newBlock, err := a.state.AddBlock(bb.Bytes)
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
	blocks := make([][]byte, 0, deltaHeight)
	for i := proto.Height(1); i <= deltaHeight; i++ {
		block, err := a.state.BlockByHeight(parentHeight + i)
		if err != nil {
			return nil, 0, err
		}

		// TODO remove this on new api
		bts, err := block.MarshalBinary()
		if err != nil {
			return nil, 0, err
		}
		blocks = append(blocks, bts)
	}

	err = a.state.RollbackToHeight(parentHeight)
	if err != nil {
		return nil, 0, err
	}

	newBlock, err := a.state.AddBlock(bb.Bytes)
	if err != nil {
		// return back saved blocks
		err2 := a.state.AddNewBlocks(blocks)
		if err2 != nil {
			return nil, 0, err2
		}
		return nil, 0, err
	}

	return newBlock, parentHeight + 1, nil
}

type BlockApplier struct {
	state       state.State
	peer        PeerManager
	scheduler   types.Scheduler
	interrupter types.MinerInterrupter
	inner       innerBlockApplier
}

func NewBlockApplier(state state.State, peer PeerManager, scheduler types.Scheduler, minerInterrupter types.MinerInterrupter) *BlockApplier {
	return &BlockApplier{
		state:       state,
		peer:        peer,
		scheduler:   scheduler,
		interrupter: minerInterrupter,

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
	return a.Apply(BlockWithBytes{Block: block, Bytes: b})
}

func (a *BlockApplier) Apply(bb BlockWithBytes) error {
	a.interrupter.Interrupt()
	m := a.state.Mutex()
	m.Lock()

	block, height, err := a.inner.apply(bb)
	if err != nil {
		m.Unlock()
		return err
	}
	m.Unlock()

	go a.scheduler.Reschedule(a.state, block, height)

	cur, err := a.state.CurrentScore()
	if err == nil {
		a.peer.EachConnected(func(peer Peer, i *big.Int) {
			peer.SendMessage(&proto.ScoreMessage{
				Score: cur.Bytes(),
			})
		})
	}

	return nil
}
