package network

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"

	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
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
}

func RunIncomingPeer(ctx context.Context, params IncomingPeerParams) {
	c := params.Conn
	readHandshake := proto.Handshake{}
	_, err := readHandshake.ReadFrom(c)
	if err != nil {
		zap.S().Error("failed to read handshake: ", err)
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
	zap.S().Infof("read handshake from %s %+v", id, readHandshake)

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
		zap.S().Error("failed to write handshake: ", err)
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
	connection := conn.WrapConnection(ctx, c, remote.ToCh, remote.FromCh, remote.ErrCh, params.Skip)
	ctx, cancel := context.WithCancel(ctx)

	p := &IncomingPeer{
		params:    params,
		conn:      connection,
		remote:    remote,
		uniqueID:  id,
		cancel:    cancel,
		handshake: readHandshake,
	}

	zap.S().Debugf("%s, readhandshake %+v", c.RemoteAddr().String(), readHandshake)
	if err := p.run(ctx); err != nil {
		zap.S().Error("peer.run(): ", err)
	}
}

func (a *IncomingPeer) run(ctx context.Context) error {
	return peer.Handle(ctx, a, a.params.Parent, a.remote, nil)
}

func (a *IncomingPeer) Close() error {
	defer a.cancel()
	return a.conn.Close()
}

func (a *IncomingPeer) SendMessage(m proto.Message) {
	b, err := m.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return
	}
	select {
	case a.remote.ToCh <- b:
	default:
		zap.S().Warnf("can't send bytes to Remote, chan is full ID %s", a.uniqueID)
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
