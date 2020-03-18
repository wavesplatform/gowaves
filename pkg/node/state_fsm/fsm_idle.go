package state_fsm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type InvRequester interface {
	Request(p types.MessageSender, inv *proto.MicroBlockInv)
}

type IdleFsm struct {
	baseInfo BaseInfo
}

func (a *IdleFsm) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *IdleFsm) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *IdleFsm) Task(task AsyncTask) (FSM, Async, error) {
	zap.S().Debugf("IdleFsm Task: got task type %d, data %+v", task.TaskType, task.Data)
	switch task.TaskType {
	case ASK_PEERS:
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	default:
		return a, nil, errors.Errorf("IdleFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

func (a *IdleFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return a.baseInfo.d.PeerError(a, p, a.baseInfo, e)
}

func (a *IdleFsm) Signatures(_ peer.Peer, _ []crypto.Signature) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func NewIdleFsm(info BaseInfo) *IdleFsm {
	return &IdleFsm{
		baseInfo: info,
	}
}

func (a *IdleFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	return a.baseInfo.d.NewPeer(a, p, a.baseInfo)
}

func (a *IdleFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	////zap.S().Debug("*IdleFsm Score ", p, score)
	//err := a.baseInfo.peers.UpdateScore(p, score)
	////zap.S().Debug("a.baseInfo.peers.UpdateScore ", err)
	//if err != nil {
	//	return a, nil, err
	//}
	//
	////return a, nil, nil
	//
	//defer a.baseInfo.storage.Mutex().Lock().Unlock()
	//myScore, err := a.baseInfo.storage.CurrentScore()
	//if err != nil {
	//	return a, nil, err
	//}
	//
	//if score.Cmp(myScore) == 1 { // remote score > my score
	//	return NewIdleToSyncTransition(a.baseInfo, p)
	//}
	//return a, nil, nil
	return handleScore(a, a.baseInfo, p, score)
}

func (a *IdleFsm) Block(peer peer.Peer, block *proto.Block) (FSM, Async, error) {
	return noop(a)
}
