package state_fsm

import (
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
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
	if err != nil {
		return a, nil, proto.NewInfoMsg(err)
	}
	a.baseInfo.BroadcastTransaction(t, p)
	return a, nil, nil
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

func (a *IdleFsm) Task(task tasks.AsyncTask) (FSM, Async, error) {
	switch task.TaskType {
	case tasks.Ping:
		return noop(a)
	case tasks.AskPeers:
		zap.S().Debug("[Idle] Requesting peers")
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.MineMicro: // Do nothing
		return a, nil, nil
	default:
		return a, nil, errors.Errorf("unexpected internal task '%d' with data '%+v' received by %s FSM", task.TaskType, task.Data, a.String())
	}
}

func (a *IdleFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return a.baseInfo.d.PeerError(a, p, a.baseInfo, e)
}

func (a *IdleFsm) BlockIDs(_ peer.Peer, _ []proto.BlockID) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
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
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, proto.NewInfoMsg(err)
	}
	nodeScore, err := a.baseInfo.storage.CurrentScore()
	if err != nil {
		return a, nil, err
	}
	if score.Cmp(nodeScore) == 1 {
		lastSignatures, err := signatures.LastSignaturesImpl{}.LastBlockIDs(a.baseInfo.storage)
		if err != nil {
			return a, nil, err
		}
		internal := sync_internal.InternalFromLastSignatures(extension.NewPeerExtension(p, a.baseInfo.scheme), lastSignatures)
		c := conf{
			peerSyncWith: p,
			timeout:      30 * time.Second,
		}
		zap.S().Debugf("[Idle] Starting synchronisation with peer '%s'", p.ID())
		return NewSyncFsm(a.baseInfo, c.Now(a.baseInfo.tm), internal)
	}
	return noop(a)
}

func (a *IdleFsm) Block(_ peer.Peer, _ *proto.Block) (FSM, Async, error) {
	return noop(a)
}

func (a *IdleFsm) String() string {
	return "Idle"
}

func NewIdleFsm(info BaseInfo) *IdleFsm {
	return &IdleFsm{
		baseInfo: info,
	}
}
