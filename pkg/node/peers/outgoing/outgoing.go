package outgoing

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

type EstablishParams struct {
	Address      string
	WavesNetwork string
	Parent       peer.Parent
	Pool         bytespool.Pool
	DeclAddr     proto.PeerInfo
	Skip         conn.SkipFilter
	NodeName     string
	NodeNonce    uint64
}

func EstablishConnection(ctx context.Context, params EstablishParams, v proto.Version) error {
	ctx, cancel := context.WithCancel(ctx)
	remote := peer.NewRemote()
	p := connector{
		params: params,
		cancel: cancel,
		remote: remote,
	}

	c, err := net.Dial("tcp", params.Address)
	if err != nil {
		return err
	}

	connection, handshake, err := p.connect(ctx, c, v)
	if err != nil {
		zap.S().Error(err, params.Address)
		return err
	}
	p.connection = connection

	peerImpl := peers.NewPeerImpl(*handshake, connection, peer.Outgoing, remote)

	connected := peer.InfoMessage{
		ID: params.Address,
		Value: &peers.Connected{
			Peer: peerImpl,
		},
	}
	params.Parent.InfoCh <- connected
	zap.S().Debugf("connected %s, id: %s", params.Address, peerImpl.ID())

	return peer.Handle(peer.HandlerParams{
		Ctx:        ctx,
		ID:         peerImpl.ID(),
		Connection: p.connection,
		Remote:     remote,
		Parent:     params.Parent,
		Pool:       params.Pool,
	})
}

type connector struct {
	params     EstablishParams
	cancel     context.CancelFunc
	remote     peer.Remote
	connection conn.Connection
}

func (a *connector) connect(ctx context.Context, c net.Conn, v proto.Version) (conn.Connection, *proto.Handshake, error) {
	bytes, err := a.params.DeclAddr.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return nil, nil, err
	}

	handshake := proto.Handshake{
		AppName:           a.params.WavesNetwork,
		Version:           v,
		NodeName:          a.params.NodeName,
		NodeNonce:         a.params.NodeNonce,
		DeclaredAddrBytes: bytes,
		Timestamp:         proto.NewTimestampFromTime(time.Now()),
	}

	_, err = handshake.WriteTo(c)
	if err != nil {
		zap.S().Error("failed to send handshake: ", err, a.params.Address)
		return nil, nil, err
	}

	select {
	case <-ctx.Done():
		c.Close()
		return nil, nil, ctx.Err()
	default:
	}

	_, err = handshake.ReadFrom(c)
	if err != nil {
		zap.S().Debugf("failed to read handshake: %s %s", err, a.params.Address)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(5 * time.Minute):
			return nil, nil, err
		}
	}
	return conn.WrapConnection(c, a.params.Pool, a.remote.ToCh, a.remote.FromCh, a.remote.ErrCh, a.params.Skip), &handshake, nil
	//}

	//return nil, nil, errors.Errorf("can't connect 20 times")
}
