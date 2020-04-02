package state_fsm

import (
	"github.com/pkg/errors"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

type NGFsm12 struct {
	BaseInfo
}

func (a *NGFsm12) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	err := a.utx.Add(t)
	return a, nil, err
}

func (a *NGFsm12) Task(task AsyncTask) (FSM, Async, error) {
	switch task.TaskType {
	case PING:
		return noop(a)
	case ASK_PEERS:
		a.peers.AskPeers()
		return a, nil, nil
	// TODO handle this
	//case MINE_MICRO:
	//	t := task.Data.(MineMicroTaskData)
	//	return a.mineMicro(t.Block, t.Limits, a.blocks, t.KeyPair)
	default:
		return a, nil, errors.Errorf("NGFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

func (a *NGFsm12) Halt() (FSM, Async, error) {
	return HaltTransition(a.BaseInfo)
}

func NewNGFsm12(info BaseInfo) *NGFsm12 {
	return &NGFsm12{
		BaseInfo: info,
	}
}

func (a *NGFsm12) NewPeer(p peer.Peer) (FSM, Async, error) {
	fsm, as, err := newPeer(a, p, a.peers)
	sendScore(p, a.storage)
	return fsm, as, err
}

func (a *NGFsm12) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return peerError(a, p, a.peers, e)
}

func (a *NGFsm12) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	return handleScore(a, a.BaseInfo, p, score)
}

func (a *NGFsm12) Block(peer peer.Peer, block *proto.Block) (FSM, Async, error) {
	err := a.blocksApplier.Apply(a.storage, []*proto.Block{block})
	if err != nil {
		return a, nil, err
	}
	a.Scheduler.Reschedule()
	return NewNGFsm12(a.BaseInfo), nil, nil
}

func (a *NGFsm12) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair) (FSM, Async, error) {
	return noop(a)
	//err := a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	//if err != nil {
	//	return a, nil, err
	//}
	//a.baseInfo.Reschedule()
	//a.baseInfo.actions.SendBlock(block)
	//rlocked := a.baseInfo.storage.Mutex().RLock()
	//score, err := a.baseInfo.storage.CurrentScore()
	//rlocked.Unlock()
	//if err != nil {
	//	return NewIdleFsm(a.baseInfo), nil, err
	//}
	//a.baseInfo.actions.SendScore(score)
	//a.blocks = a.blocks.ForceAddBlock(block)
	//return NewNGFsm12(a.baseInfo), Tasks(NewMineMicroTask(5*time.Second, block, limits, keyPair)), nil
}

func (a *NGFsm12) BlockIDs(peer peer.Peer, sigs []proto.BlockID) (FSM, Async, error) {
	return noop(a)
}

func (a *NGFsm12) GetPeers(peer peer.Peer) (FSM, Async, error) {
	return sendPeers(a, peer, a.peers)
}

func (a *NGFsm12) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
	// TODO check if it really need
	defer a.Reschedule()
	if micro.Reference.IsSignature() {
		return a.microBlockBySignature(micro)
	} else {
		return a.microBlockByID(micro)
	}
}

//func (a *NGFsm12) mineMicro(minedBlock *proto.Block, rest proto.MiningLimits, blocks ng.Blocks, keyPair proto.KeyPair) (FSM, Async, error) {
//	block, micro, rest, newBlocks, err := a.baseInfo.microMiner.Micro(minedBlock, rest, blocks, keyPair)
//	if err == miner.NoTransactionsErr {
//		return a, Tasks(NewMineMicroTask(5*time.Second, minedBlock, rest, keyPair)), nil
//	}
//	defer a.baseInfo.Reschedule()
//	if err != nil {
//		return a, nil, errors.Wrap(err, "NGFsm.mineMicro")
//	}
//	err = a.baseInfo.storage.Mutex().Map(func() error {
//		return a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
//	})
//	if err != nil {
//		return a, nil, err
//	}
//	inv := proto.NewUnsignedMicroblockInv(micro.SenderPK, micro.TotalBlockID, micro.Reference)
//	err = inv.Sign(keyPair.Secret, a.baseInfo.scheme)
//	if err != nil {
//		return a, nil, err
//	}
//	invBts, err := inv.MarshalBinary()
//	if err != nil {
//		return a, nil, err
//	}
//	a.baseInfo.MicroBlockCache.Add(micro)
//	a.baseInfo.peers.EachConnected(func(p peer.Peer, score *proto.Score) {
//		p.SendMessage(
//			&proto.MicroBlockInvMessage{
//				Body: invBts,
//			},
//		)
//	})
//	a.blocks = newBlocks
//	return a, Tasks(NewMineMicroTask(5*time.Second, block, rest, keyPair)), nil
//}

func (a *NGFsm12) microBlockByID(micro *proto.MicroBlock) (FSM, Async, error) {
	top := a.storage.TopBlock()
	if top.BlockID() != micro.Reference {
		return a, nil, errors.New("micro reference not found")
	}
	b, err := a.storage.Block(micro.Reference)
	if err != nil {
		return a, nil, err
	}
	newTrs := b.Transactions.Join(micro.Transactions)
	//b.Transactions = newTrs
	//err = b.GenerateBlockID(a.scheme)
	//if err != nil {
	//	return a, nil, err
	//}
	newBlock, err := proto.CreateBlock(newTrs, b.Timestamp, b.Parent, b.GenPublicKey, b.NxtConsensus, b.Version, b.Features, b.RewardVote, a.scheme)
	if err != nil {
		return a, nil, err
	}
	newBlock.BlockSignature = micro.TotalResBlockSigField
	ok, err := newBlock.VerifySignature(a.scheme)
	if err != nil {
		return a, nil, err
	}
	if !ok {
		return a, nil, errors.New("incorrect signature for applied microblock")
	}
	err = a.storage.Map(func(state state.State) error {
		return a.blocksApplier.Apply(state, []*proto.Block{b})
	})
	if err != nil {
		return a, nil, errors.Wrap(err, "failed to apply created from micro block")
	}
	return a, nil, nil

	//
	//block
	//
	//
	//micro.Reference

	//blocks, err := a.blocks.AddMicro(micro)
	//if err != nil {
	//	return a, nil, errors.Wrap(err, "failed add micro to row")
	//}
	//block, err := a.blockCreater.FromMicroblockRow(blocks.Row())
	//if err != nil {
	//	return a, nil, errors.Wrap(err, "failed create block from row")
	//}
	//ok, err := block.VerifySignature(a.baseInfo.scheme)
	//if err != nil {
	//	return a, nil, errors.Wrap(err, "failed to verify signature")
	//}
	//if !ok {
	//	return a, nil, errors.New("IdleFsm MicroBlock: failed to validate created block sig")
	//}
	//err = a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	//if err != nil {
	//	return a, nil, errors.Wrap(err, "failed apply block to storage")
	//}
	//a.blocks = blocks
	//return a, nil, nil
}

func (a *NGFsm12) microBlockBySignature(micro *proto.MicroBlock) (FSM, Async, error) {
	//top := a.storage.TopBlock()
	//if top.BlockID() != micro.Reference {
	//	return a, nil, errors.New("micro reference not found")
	//}
	b, err := a.storage.Block(a.storage.TopBlock().BlockID())
	if err != nil {
		return a, nil, err
	}
	newTrs := b.Transactions.Join(micro.Transactions)
	newBlock, err := proto.CreateBlock(newTrs, b.Timestamp, b.Parent, b.GenPublicKey, b.NxtConsensus, b.Version, b.Features, b.RewardVote, a.scheme)
	if err != nil {
		return a, nil, err
	}
	newBlock.BlockSignature = micro.TotalResBlockSigField
	ok, err := newBlock.VerifySignature(a.scheme)
	if err != nil {
		return a, nil, err
	}
	if !ok {
		return a, nil, errors.New("incorrect signature for applied microblock")
	}
	err = a.storage.Map(func(state state.NonThreadSafeState) error {
		return a.blocksApplier.Apply(state, []*proto.Block{b})
	})
	if err != nil {
		return a, nil, errors.Wrap(err, "failed to apply created from micro block")
	}
	return a, nil, nil
}

func (a *NGFsm12) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error) {
	zap.S().Info("got inv, requesting microblock")
	a.invRequester.Request(p, inv.TotalBlockID.Bytes())
	return a, nil, nil
}

func MinedBlockNgTransition(info BaseInfo, block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair) (FSM, Async, error) {
	return NewNGFsm12(info).MinedBlock(block, limits, keyPair)
}
