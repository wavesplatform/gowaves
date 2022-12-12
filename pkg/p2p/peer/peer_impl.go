package peer

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type peerImplID struct {
	addr  string
	nonce uint64
}

func newPeerImplID(addr net.Addr, nonce uint64) peerImplID {
	a := addr.String()
	if idx := strings.LastIndexByte(a, ':'); idx != -1 {
		a = a[:idx]               // "192.0.2.1:25" -> "192.0.2.1"; "[2001:db8::1]:80 -> "[2001:db8::1]"
		a = strings.Trim(a, "[]") // for IPV4 - no changes, for IPV6 "[2001:db8::1]" -> "2001:db8::1"
	}
	return peerImplID{addr: a, nonce: nonce}
}

func (id peerImplID) String() string {
	return fmt.Sprintf("%s-%d", id.addr, id.nonce)
}

type PeerImpl struct {
	handshake proto.Handshake
	conn      conn.Connection
	direction Direction
	remote    Remote
	id        peerImplID
	cancel    context.CancelFunc
}

func NewPeerImpl(handshake proto.Handshake, conn conn.Connection, direction Direction, remote Remote, cancel context.CancelFunc) *PeerImpl {
	return &PeerImpl{
		handshake: handshake,
		conn:      conn,
		direction: direction,
		remote:    remote,
		id:        newPeerImplID(conn.Conn().RemoteAddr(), handshake.NodeNonce),
		cancel:    cancel,
	}
}

func (a *PeerImpl) Direction() Direction {
	return a.direction
}

func (a *PeerImpl) Close() error {
	defer a.cancel()
	return a.conn.Close()
}

func (a *PeerImpl) SendMessage(m proto.Message) {
	b, err := m.MarshalBinary()
	if err != nil {
		zap.S().Errorf("Failed to send message %T: %v", m, err)
		return
	}
	select {
	case a.remote.ToCh <- b:
	default:
		a.remote.ErrCh <- errors.Errorf("remote, chan is full id %s, name %s", a.ID(), a.handshake.NodeName)
	}
}

func (a *PeerImpl) ID() ID {
	return a.id
}

func (a *PeerImpl) Connection() conn.Connection {
	return a.conn
}

func (a *PeerImpl) Handshake() proto.Handshake {
	return a.handshake
}

func (a *PeerImpl) RemoteAddr() proto.TCPAddr {
	addr := a.Connection().Conn().RemoteAddr().(*net.TCPAddr)
	return proto.TCPAddr(*addr)
}
