package state_fsm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type InvRequester interface {
	Request(p types.MessageSender, id []byte)
}

type IdleFsm struct {
	baseInfo BaseInfo
}

func (a *IdleFsm) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	err := a.baseInfo.utx.Add(t)
	if err == nil {
		a.baseInfo.BroadcastTransaction(t, p)
	}
	return a, nil, err
}

func (a *IdleFsm) Halt() (FSM, Async, error) {
	return HaltTransition(a.baseInfo)
}

func (a *IdleFsm) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	return MinedBlockNgTransition(a.baseInfo, block, limits, keyPair, vrf)
}

func (a *IdleFsm) MicroBlock(_ peer.Peer, _ *proto.MicroBlock) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *IdleFsm) MicroBlockInv(_ peer.Peer, _ *proto.MicroBlockInv) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *IdleFsm) Task(task AsyncTask) (FSM, Async, error) {
	zap.S().Debugf("IdleFsm Task: got task type %d, data %+v", task.TaskType, task.Data)
	switch task.TaskType {
	case Ping:
		return noop(a)
	case AskPeers:
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case MineMicro: // Do nothing
		return a, nil, nil
	default:
		return a, nil, errors.Errorf("IdleFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

func (a *IdleFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return a.baseInfo.d.PeerError(a, p, a.baseInfo, e)
}

func (a *IdleFsm) BlockIDs(_ peer.Peer, _ []proto.BlockID) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func NewIdleFsm(info BaseInfo) *IdleFsm {
	return &IdleFsm{
		baseInfo: info,
	}
}

func (a *IdleFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	fsm, as, err := newPeer(a, p, a.baseInfo.peers)
	if a.baseInfo.peers.ConnectedCount() == a.baseInfo.minPeersMining {
		a.baseInfo.Reschedule()
	}
	sendScore(p, a.baseInfo.storage)
	return fsm, as, err
}

func (a *IdleFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	metrics.FSMScore("idle", score, p.Handshake().NodeName)
	return handleScore(a, a.baseInfo, p, score)
}

func (a *IdleFsm) Block(_ peer.Peer, _ *proto.Block) (FSM, Async, error) {
	return noop(a)
}
