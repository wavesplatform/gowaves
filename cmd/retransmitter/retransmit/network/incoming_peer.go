package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type IncomingPeer struct {
	params    IncomingPeerParams
	conn      conn.Connection
	remote    peer.Remote
	uniqueID  incomingPeerID
	cancel    context.CancelFunc
	handshake proto.Handshake
}

type incomingPeerID struct {
	remoteAddr string
	localAddr  string
}

func newPeerID(remoteAddr net.Addr, localAddr net.Addr) incomingPeerID {
	return incomingPeerID{remoteAddr: remoteAddr.String(), localAddr: localAddr.String()}
}

func (id incomingPeerID) String() string {
	return fmt.Sprintf("incoming Connection %s -> %s", id.remoteAddr, id.localAddr)
}

type IncomingPeerParams struct {
	WavesNetwork string
	Conn         net.Conn
	Parent       peer.Parent
	DeclAddr     proto.TCPAddr
	Skip         conn.SkipFilter
	Logger       *slog.Logger
	DataLogger   *slog.Logger
}

func RunIncomingPeer(ctx context.Context, params IncomingPeerParams) {
	c := params.Conn
	readHandshake := proto.Handshake{}
	_, err := readHandshake.ReadFrom(c)
	if err != nil {
		slog.Error("Failed to read handshake", logging.Error(err))
		_ = c.Close()
		return
	}

	select {
	case <-ctx.Done():
		_ = c.Close()
		return
	default:
	}

	id := newPeerID(c.RemoteAddr(), c.LocalAddr())
	slog.Info("Read handshake", "from", id, "handshake", readHandshake)

	writeHandshake := proto.Handshake{
		AppName: params.WavesNetwork,
		// pass the same minor version as received
		Version:      proto.NewVersion(readHandshake.Version.Major(), readHandshake.Version.Minor(), 0),
		NodeName:     "retransmitter",
		NodeNonce:    0x0,
		DeclaredAddr: proto.HandshakeTCPAddr(params.DeclAddr),
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}

	_, err = writeHandshake.WriteTo(c)
	if err != nil {
		slog.Error("Failed to write handshake", logging.Error(err))
		_ = c.Close()
		return
	}

	select {
	case <-ctx.Done():
		_ = c.Close()
		return
	default:
	}

	remote := peer.NewRemote()
	connection := conn.WrapConnection(ctx, c, remote.ToCh, remote.FromCh, remote.ErrCh, params.Skip, params.Logger)
	ctx, cancel := context.WithCancel(ctx)

	p := &IncomingPeer{
		params:    params,
		conn:      connection,
		remote:    remote,
		uniqueID:  id,
		cancel:    cancel,
		handshake: readHandshake,
	}

	slog.Debug("Handshake read", "remote", c.RemoteAddr().String(), "handshake", readHandshake)
	if err := p.run(ctx); err != nil {
		slog.Error("Failed peer.run()", logging.Error(err))
	}
}

func (a *IncomingPeer) run(ctx context.Context) error {
	return peer.Handle(ctx, a, a.params.Parent, a.remote, a.params.Logger, a.params.DataLogger)
}

func (a *IncomingPeer) Close() error {
	defer a.cancel()
	return a.conn.Close()
}

func (a *IncomingPeer) SendMessage(m proto.Message) {
	b, err := m.MarshalBinary()
	if err != nil {
		slog.Error("Failed to send message", logging.Error(err))
		return
	}
	select {
	case a.remote.ToCh <- b:
	default:
		slog.Warn("Can't send bytes to Remote, chan is full ID", "ID", a.uniqueID)
	}
}

func (a *IncomingPeer) ID() peer.ID {
	return a.uniqueID
}

func (a *IncomingPeer) Direction() peer.Direction {
	return peer.Incoming
}

func (a *IncomingPeer) Connection() conn.Connection {
	return a.conn
}

func (a *IncomingPeer) Handshake() proto.Handshake {
	return a.handshake
}

func (a *IncomingPeer) RemoteAddr() proto.TCPAddr {
	addr := a.conn.Conn().RemoteAddr().(*net.TCPAddr)
	return proto.TCPAddr(*addr)
}

func (a *IncomingPeer) Equal(other peer.Peer) bool {
	if other == nil {
		return false
	}
	return a.ID() == other.ID()
}
