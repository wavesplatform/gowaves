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

type PeerImpl struct {
	handshake proto.Handshake
	conn      conn.Connection
	direction Direction
	remote    Remote
	id        string
	cancel    context.CancelFunc
}

func NewPeerImpl(handshake proto.Handshake, conn conn.Connection, direction Direction, remote Remote, cancel context.CancelFunc) *PeerImpl {
	return &PeerImpl{
		handshake: handshake,
		conn:      conn,
		direction: direction,
		remote:    remote,
		id:        id(conn.Conn().RemoteAddr().String(), handshake.NodeNonce),
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
		zap.S().Error(err)
		return
	}
	select {
	case a.remote.ToCh <- b:
	default:
		a.remote.ErrCh <- errors.Errorf("remote, chan is full id %s, name %s", a.ID(), a.handshake.NodeName)
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

func (a *PeerImpl) RemoteAddr() proto.TCPAddr {
	addr := a.Connection().Conn().RemoteAddr().(*net.TCPAddr)
	return proto.TCPAddr(*addr)
}
