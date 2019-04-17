package network

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"net"
	"time"

	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type IncomingPeer struct {
	params    IncomingPeerParams
	conn      conn.Connection
	remote    peer.Remote
	uniqueID  string
	cancel    context.CancelFunc
	handshake proto.Handshake
}

type IncomingPeerParams struct {
	WavesNetwork string
	Conn         net.Conn
	Parent       peer.Parent
	DeclAddr     proto.TCPAddr
	Pool         bytespool.Pool
	Skip         conn.SkipFilter
}

func RunIncomingPeer(ctx context.Context, params IncomingPeerParams) {
	c := params.Conn
	readHandshake := proto.Handshake{}
	_, err := readHandshake.ReadFrom(c)
	if err != nil {
		zap.S().Error("failed to read handshake: ", err)
		c.Close()
		return
	}

	select {
	case <-ctx.Done():
		c.Close()
		return
	default:
	}

	id := fmt.Sprintf("incoming Connection %s -> %s", c.RemoteAddr().String(), c.LocalAddr().String())
	zap.S().Infof("read handshake from %s %+v", id, readHandshake)

	writeHandshake := proto.Handshake{
		AppName: params.WavesNetwork,
		// pass the same minor version as received
		Version:      proto.Version{Major: 0, Minor: readHandshake.Version.Minor, Patch: 0},
		NodeName:     "retransmitter",
		NodeNonce:    0x0,
		DeclaredAddr: proto.HandshakeTCPAddr(params.DeclAddr),
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}

	_, err = writeHandshake.WriteTo(c)
	if err != nil {
		zap.S().Error("failed to write handshake: ", err)
		c.Close()
		return
	}

	select {
	case <-ctx.Done():
		c.Close()
		return
	default:
	}

	remote := peer.NewRemote()
	connection := conn.WrapConnection(c, params.Pool, remote.ToCh, remote.FromCh, remote.ErrCh, params.Skip)
	ctx, cancel := context.WithCancel(ctx)

	p := &IncomingPeer{
		params:    params,
		conn:      connection,
		remote:    remote,
		uniqueID:  fmt.Sprintf("incoming Connection %s -> %s", c.RemoteAddr().String(), c.LocalAddr().String()),
		cancel:    cancel,
		handshake: readHandshake,
	}

	zap.S().Debugf("%s, readhandshake %+v", c.RemoteAddr().String(), readHandshake)

	out := peer.InfoMessage{
		ID: p.ID(),
		Value: &peer.Connected{
			Peer: p,
		},
	}
	params.Parent.InfoCh <- out
	p.run(ctx)
}

func (a *IncomingPeer) run(ctx context.Context) {
	handleParams := peer.HandlerParams{
		Connection: a.conn,
		Ctx:        ctx,
		Remote:     a.remote,
		ID:         a.uniqueID,
		Parent:     a.params.Parent,
		Pool:       a.params.Pool,
	}
	peer.Handle(handleParams)
}

func (a *IncomingPeer) Close() error {
	a.cancel()
	return nil
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

func (a *IncomingPeer) ID() string {
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
