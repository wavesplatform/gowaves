package peer

import (
	"context"
	"fmt"
	"net"
	"net/netip"

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
	handshake proto.Handshake
	conn      conn.Connection
	direction Direction
	remote    Remote
	id        peerImplID
	cancel    context.CancelFunc
}

func NewPeerImpl(handshake proto.Handshake, conn conn.Connection, direction Direction, remote Remote, cancel context.CancelFunc) (*PeerImpl, error) {
	id, err := newPeerImplID(conn.Conn().RemoteAddr(), handshake.NodeNonce)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new peer")
	}
	return &PeerImpl{
		handshake: handshake,
		conn:      conn,
		direction: direction,
		remote:    remote,
		id:        id,
		cancel:    cancel,
	}, nil
}

func (a *PeerImpl) Direction() Direction {
	return a.direction
}

func (a *PeerImpl) Close() error {
	defer a.cancel()
	return a.conn.Close()
}

// SendMessage marshals provided message and sends it to its internal Remote.ToCh channel.
// It sends the error to internal Remote.ErrCh if Remote.ToCh is full.
// That notifies Handle to propagate this error to FMS through Parent.InfoCh.
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
