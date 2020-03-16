package state_fsm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/ng"
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
	return NewNGFsm(a.baseInfo)
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
	default:
		return a, nil, errors.Errorf("IdleFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

func (a *NGFsm) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
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
