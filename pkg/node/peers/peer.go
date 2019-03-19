package peers

import (
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"strings"
)

type Connected struct {
	Peer peer.Peer
}

type PeerImpl struct {
	handshake proto.Handshake
	conn      conn.Connection
	direction peer.Direction
	remote    peer.Remote
	id        string
}

func NewPeerImpl(handshake proto.Handshake, conn conn.Connection, direction peer.Direction, remote peer.Remote) *PeerImpl {
	return &PeerImpl{
		handshake: handshake,
		conn:      conn,
		direction: direction,
		remote:    remote,
		id:        id(conn.Conn().RemoteAddr().String(), handshake.NodeNonce),
	}
}

func (a *PeerImpl) Direction() peer.Direction {
	return a.direction
}

func (a *PeerImpl) Close() {
	a.conn.Close()
}

func (a *PeerImpl) SendMessage(m proto.Message) {
	b, err := m.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return
	}
	select {
	case a.remote.ToCh <- b:
	default:
		zap.S().Warnf("can't send bytes to remote, chan is full id %s", a.ID())
	}
}

func (a *PeerImpl) ID() string {
	return a.id
}

func (a *PeerImpl) Connection() conn.Connection {
	return a.conn
}

func (a *PeerImpl) Handshake() proto.Handshake {
	return a.handshake
}

func id(addr string, nonce uint64) string {
	a := strings.Split(addr, ":")[0]
	return fmt.Sprintf("%s-%d", a, nonce)
}

func (a *PeerImpl) RemoteAddr() proto.NodeAddr {
	addr := a.Connection().Conn().RemoteAddr().(*net.TCPAddr)
	return proto.NodeAddrFromTCPAddr(addr)
}
