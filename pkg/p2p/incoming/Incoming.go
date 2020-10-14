package incoming

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/p2p/common"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type UtxPool interface {
	AddBytes([]byte) error
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
	UtxPool      UtxPool
}

func RunIncomingPeer(ctx context.Context, params IncomingPeerParams) error {
	ctx, cancel := context.WithCancel(ctx)
	return runIncomingPeer(ctx, cancel, params)
}

func runIncomingPeer(ctx context.Context, cancel context.CancelFunc, params IncomingPeerParams) error {
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
		return errors.Wrap(ctx.Err(), "RunIncomingPeer")
	default:
	}

	writeHandshake := proto.Handshake{
		AppName:      params.WavesNetwork,
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
		return errors.Wrap(ctx.Err(), "RunIncomingPeer")
	default:
	}

	remote := peer.NewRemote()
	connection := conn.WrapConnection(c, params.Pool, remote.ToCh, remote.FromCh, remote.ErrCh, params.Skip)
	peerImpl := peer.NewPeerImpl(readHandshake, connection, peer.Incoming, remote, cancel)

	out := peer.InfoMessage{
		Peer: peerImpl,
		Value: &peer.Connected{
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
		Peer:       peerImpl,
		PreHandler: func(message proto.Message) bool {
			return common.PreHandler(message, params.UtxPool)
		},
	})
}
