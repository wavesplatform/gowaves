package incoming

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"time"
)

type IncomingPeer struct {
	params   IncomingPeerParams
	conn     conn.Connection
	remote   peer.Remote
	uniqueID string
	cancel   context.CancelFunc
}

type IncomingPeerParams struct {
	WavesNetwork string
	Conn         net.Conn
	Parent       peer.Parent
	DeclAddr     proto.TCPAddr
	Pool         bytespool.Pool
	Skip         conn.SkipFilter
	NodeName     string
	NodeNonce    uint64
	Version      proto.Version
}

func RunIncomingPeer(ctx context.Context, params IncomingPeerParams) error {
	c := params.Conn

	readHandshake := proto.Handshake{}
	_, err := readHandshake.ReadFrom(c)
	if err != nil {
		zap.S().Error("failed to read handshake: ", err)
		c.Close()
		return err
	}

	select {
	case <-ctx.Done():
		c.Close()
		return ctx.Err()
	default:
	}

	//id := fmt.Sprintf("incoming Connection %s -> %s", c.RemoteAddr().String(), c.LocalAddr().String())
	zap.S().Infof("read handshake from %s %+v", readHandshake)

	writeHandshake := proto.Handshake{
		AppName: params.WavesNetwork,
		// pass the same minor version as received
		Version:      params.Version,
		NodeName:     params.NodeName,
		NodeNonce:    params.NodeNonce,
		DeclaredAddr: proto.HandshakeTCPAddr(params.DeclAddr),
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}

	_, err = writeHandshake.WriteTo(c)
	if err != nil {
		zap.S().Error("failed to write handshake: ", err)
		c.Close()
		return err
	}

	select {
	case <-ctx.Done():
		c.Close()
		return ctx.Err()
	default:
	}

	remote := peer.NewRemote()
	connection := conn.WrapConnection(c, params.Pool, remote.ToCh, remote.FromCh, remote.ErrCh, params.Skip)
	//ctx, cancel := context.WithCancel(ctx)

	//inPeer := &IncomingPeer{
	//	params: params,
	//	conn:   connection,
	//	remote: remote,
	//	//uniqueID: fmt.Sprintf("incoming Connection %s -> %s", c.RemoteAddr().String(), c.LocalAddr().String()),
	//	cancel: cancel,
	//}

	//decl := proto.PeerInfo{}
	//_ = decl.UnmarshalBinary(readHandshake.DeclaredAddr)
	zap.S().Debugf("%s, readhandshake %+v", c.RemoteAddr().String(), readHandshake)

	peerImpl := peers.NewPeerImpl(readHandshake, connection, peer.Incoming, remote)

	out := peer.InfoMessage{
		ID: peerImpl.ID(),
		Value: &peers.Connected{
			Peer: peerImpl,
		},
	}
	params.Parent.InfoCh <- out

	return peer.Handle(peer.HandlerParams{
		Ctx:        ctx,
		ID:         peerImpl.ID(),
		Connection: connection,
		Remote:     remote,
		Parent:     params.Parent,
		Pool:       params.Pool,
	})
}

//
//func (a *IncomingPeer) run(ctx context.Context) error {
//	handleParams := peer.HandlerParams{
//		Connection: a.conn,
//		Ctx:        ctx,
//		Remote:     a.remote,
//		ID:         a.uniqueID,
//		Parent:     a.params.Parent,
//		Pool:       a.params.Pool,
//	}
//	return peer.Handle(handleParams)
//}

//func (a *IncomingPeer) Close() {
//	a.cancel()
//}
//
//func (a *IncomingPeer) SendMessage(m proto.Message) {
//	b, err := m.MarshalBinary()
//	if err != nil {
//		zap.S().Error(err)
//		return
//	}
//	select {
//	case a.remote.ToCh <- b:
//	default:
//		zap.S().Warnf("can't send bytes to Remote, chan is full ID %s", a.uniqueID)
//	}
//}
//
//func (a *IncomingPeer) ID() string {
//	return a.uniqueID
//}
//
//func (a *IncomingPeer) Direction() Direction {
//	return Incoming
//}
//
//func (a *IncomingPeer) Connection() conn.Connection {
//	return a.conn
//}
