package peer

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type peerImplID struct {
	addr16 [16]byte
	nonce  uint64
}

func newPeerImplID(addr net.Addr, nonce uint64) (peerImplID, error) {
	var (
		netStr  = addr.Network()
		addrStr = addr.String()
	)
	tcpAddr, err := net.ResolveTCPAddr(netStr, addrStr)
	if err != nil {
		return peerImplID{}, errors.Wrapf(err, "failed to resolve '%s' addr from '%s'", netStr, addrStr)
	}
	var addr16 [16]byte
	copy(addr16[:], tcpAddr.IP.To16())
	return peerImplID{addr16: addr16, nonce: nonce}, nil
}

func (id peerImplID) String() string {
	addr := netip.AddrFrom16(id.addr16).Unmap()
	return fmt.Sprintf("%s-%d", addr.String(), id.nonce)
}

type PeerImpl struct {
	conn      *releaseOnceConn
	handshake proto.Handshake
	direction Direction
	remote    Remote
	id        peerImplID
}

type releaseOnceConn struct {
	conn.Connection
	once         sync.Once
	cancel       context.CancelFunc
	errOnRelease error
}

func newReleaseOnceConn(conn conn.Connection, cancel context.CancelFunc) *releaseOnceConn {
	return &releaseOnceConn{Connection: conn, cancel: cancel}
}

func (c *releaseOnceConn) Close() error {
	c.once.Do(func() {
		defer c.cancel()
		c.errOnRelease = c.Connection.Close()
	})
	return c.errOnRelease
}

func NewPeerImpl(handshake proto.Handshake, conn conn.Connection, direction Direction, remote Remote, cancel context.CancelFunc) (*PeerImpl, error) {
	id, err := newPeerImplID(conn.Conn().RemoteAddr(), handshake.NodeNonce)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new peer")
	}
	return &PeerImpl{
		handshake: handshake,
		conn:      newReleaseOnceConn(conn, cancel),
		direction: direction,
		remote:    remote,
		id:        id,
	}, nil
}

func (a *PeerImpl) Direction() Direction {
	return a.direction
}

func (a *PeerImpl) Close() error {
	return a.conn.Close() // we're using releaseOnceConn, so inner conn.Close() will be called exactly once
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
