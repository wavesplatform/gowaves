package state_fsm

import (
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

func newPeer(fsm FSM, p peer.Peer, peers peer_manager.PeerManager) (FSM, Async, error) {
	err := peers.NewConnection(p)
	if err != nil {
		return fsm, nil, fsm.Errorf(proto.NewInfoMsg(err))
	}
	return fsm, nil, nil
}

func tryBroadcastTransaction(fsm FSM, baseInfo BaseInfo, p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	t, err := t.Validate(baseInfo.scheme)
	if err != nil {
		err = errors.Wrap(err, "Failed to validate transaction")
		if p != nil {
			baseInfo.peers.AddToBlackList(p, time.Now(), err.Error())
		}
		return fsm, nil, fsm.Errorf(proto.NewInfoMsg(err))
	}

	if err := baseInfo.utx.Add(t); err != nil {
		err = errors.Wrap(err, "Failed to add transaction to utx")
		return fsm, nil, fsm.Errorf(proto.NewInfoMsg(err))
	}
	baseInfo.BroadcastTransaction(t, p)
	return fsm, nil, nil
}

// TODO handle no peers
func peerError(fsm FSM, p peer.Peer, peers peer_manager.PeerManager, _ error) (FSM, Async, error) {
	peers.Disconnect(p)
	return fsm, nil, nil
}

func noop(fsm FSM) (FSM, Async, error) {
	return fsm, nil, nil
}

func sendScore(p peer.Peer, storage state.State) {
	curScore, err := storage.CurrentScore()
	if err != nil {
		zap.S().Errorf("Failed to send current score to peer %q: %v", p.RemoteAddr().String(), err)
		return
	}

	bts := curScore.Bytes()
	p.SendMessage(&proto.ScoreMessage{Score: bts})
}

func syncWithNewPeer(fsm FSM, baseInfo BaseInfo, p peer.Peer) (FSM, Async, error) {
	lastSignatures, err := signatures.LastSignaturesImpl{}.LastBlockIDs(baseInfo.storage)
	if err != nil {
		return fsm, nil, err
	}
	internal := sync_internal.InternalFromLastSignatures(extension.NewPeerExtension(p, baseInfo.scheme), lastSignatures)
	c := conf{
		peerSyncWith: p,
		timeout:      30 * time.Second,
	}
	zap.S().Debugf("[%s] Starting synchronization with peer '%s'", fsm.String(), p.ID())
	return NewSyncFsm(baseInfo, c.Now(baseInfo.tm), internal)
}

func fsmErrorf(fsm FSM, err error) error {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case *proto.InfoMsg:
		return proto.NewInfoMsg(errors.Errorf("[%s] %s", fsm.String(), e.Error()))
	default:
		return errors.Errorf("[%s] %s", fsm.String(), e.Error())
	}
}
