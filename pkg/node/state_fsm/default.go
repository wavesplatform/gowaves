package state_fsm

import (
	"time"

	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
)

const (
	askPeersInterval = 5 * time.Minute
)

type Default interface {
	Noop(FSM) (FSM, Async, error)
	PeerError(fsm FSM, p Peer, baseInfo BaseInfo, _ error) (FSM, Async, error)
}

type DefaultImpl struct {
}

func (a DefaultImpl) Noop(f FSM) (FSM, Async, error) {
	return f, nil, nil
}

func (a DefaultImpl) PeerError(fsm FSM, p Peer, baseInfo BaseInfo, _ error) (FSM, Async, error) {
	baseInfo.peers.Disconnect(p)
	if baseInfo.peers.ConnectedCount() == 0 {
		return NewIdleFsm(baseInfo), nil, nil
	}
	return fsm, nil, nil
}
