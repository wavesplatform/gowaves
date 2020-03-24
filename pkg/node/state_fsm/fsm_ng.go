package state_fsm

import (
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/ng"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type NGFsm struct {
	baseInfo BaseInfo

	blocks ng.Blocks
}

func NewNGFsm(info BaseInfo) (FSM, Async, error) {
	return &NGFsm{
		blocks:   ng.NewBlocksFromBlock(info.storage.TopBlock()),
		baseInfo: info,
	}, nil, nil
}

func (a *NGFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	return newPeer(a, p, a.baseInfo.peers)
}

func (a *NGFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return peerError(a, p, a.baseInfo.peers, e)
}

func (a *NGFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	return handleScore(a, a.baseInfo, p, score)
}

func (a *NGFsm) Block(peer peer.Peer, block *proto.Block) (FSM, Async, error) {
	err := a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	if err != nil {
		return NewIdleFsm(a.baseInfo), nil, err
	}
	a.baseInfo.Scheduler.Reschedule()
	return NewNGFsm(a.baseInfo)
}

func (a *NGFsm) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair) (FSM, Async, error) {
	err := a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	if err != nil {
		return a, nil, err
	}
	a.baseInfo.Reschedule()
	return a, Tasks(NewMineMicroTask(5*time.Second, block, limits, keyPair)), nil
}

func (a *NGFsm) Signatures(peer peer.Peer, sigs []crypto.Signature) (FSM, Async, error) {
	return noop(a)
}

func (a *NGFsm) GetPeers(peer peer.Peer) (FSM, Async, error) {
	return sendPeers(a, peer, a.baseInfo.peers)
}

func (a *NGFsm) Task(task AsyncTask) (FSM, Async, error) {
	zap.S().Debugf("NGFsm Task: got task type %d, data %+v", task.TaskType, task.Data)
	switch task.TaskType {
	case ASK_PEERS:
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case MINE_MICRO:
		t := task.Data.(MineMicroTaskData)
		return a.mineMicro(t.Block, t.Limits, a.blocks, t.KeyPair)
	default:
		return a, nil, errors.Errorf("IdleFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

func (a *NGFsm) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
	return a.microBlock(micro)
}

func (a *NGFsm) mineMicro(minedBlock *proto.Block, rest proto.MiningLimits, blocks ng.Blocks, keyPair proto.KeyPair) (FSM, Async, error) {
	block, micro, rest, err := a.baseInfo.microMiner.Micro(minedBlock, rest, blocks, keyPair)
	if err != nil {
		return a, nil, err
	}
	// TODO send micro block inv
	err = a.baseInfo.storage.Mutex().Map(func() error {
		return a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	})
	if err != nil {
		return a, nil, err
	}
	inv := proto.NewUnsignedMicroblockInv(micro.SenderPK, micro.TotalResBlockSigField, micro.PrevResBlockSigField)
	err = inv.Sign(keyPair.Secret, a.baseInfo.scheme)
	if err != nil {
		return a, nil, err
	}
	invBts, err := inv.MarshalBinary()
	if err != nil {
		return a, nil, err
	}
	a.baseInfo.MicroBlockCache.Add(micro)
	a.baseInfo.peers.EachConnected(func(p peer.Peer, score *proto.Score) {
		p.SendMessage(
			&proto.MicroBlockInvMessage{
				Body: invBts,
			},
		)
	})
	return a, Tasks(NewMineMicroTask(5*time.Second, block, rest, keyPair)), nil
}

func (a *NGFsm) microBlock(micro *proto.MicroBlock) (FSM, Async, error) {
	blocks, err := a.blocks.AddMicro(micro)
	if err != nil {
		return a, nil, err
	}
	block, err := a.baseInfo.blockCreater.FromMicroblockRow(blocks.Row())
	if err != nil {
		return a, nil, err
	}
	ok, err := block.VerifySignature(a.baseInfo.scheme)
	if err != nil {
		return a, nil, err
	}
	if !ok {
		return a, nil, errors.New("IdleFsm MicroBlock: failed to validate created block sig")
	}
	err = a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	if err != nil {
		return a, nil, err
	}
	a.blocks = blocks
	return a, nil, nil
}

func (a *NGFsm) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error) {
	a.baseInfo.invRequester.Request(p, inv)
	return a, nil, nil
}
