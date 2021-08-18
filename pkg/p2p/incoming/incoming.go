package incoming

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type DuplicateChecker interface {
	Add([]byte) bool
}

type PeerParams struct {
	WavesNetwork     string
	Conn             net.Conn
	Parent           peer.Parent
	DeclAddr         proto.TCPAddr
	Skip             conn.SkipFilter
	NodeName         string
	NodeNonce        uint64
	Version          proto.Version
	DuplicateChecker DuplicateChecker
}

func RunIncomingPeer(ctx context.Context, params PeerParams) error {
	ctx, cancel := context.WithCancel(ctx)
	return runIncomingPeer(ctx, cancel, params)
}

func runIncomingPeer(ctx context.Context, cancel context.CancelFunc, params PeerParams) error {
	c := params.Conn

	readHandshake := proto.Handshake{}
	_, err := readHandshake.ReadFrom(c)
	if err != nil {
		zap.S().Debug("Failed to read handshake: ", err)
		_ = c.Close()
		return err
	}

	select {
	case <-ctx.Done():
		_ = c.Close()
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
		zap.S().Debug("failed to write handshake: ", err)
		_ = c.Close()
		return err
	}

	select {
	case <-ctx.Done():
		_ = c.Close()
		return errors.Wrap(ctx.Err(), "RunIncomingPeer")
	default:
	}

	remote := peer.NewRemote()
	connection := conn.WrapConnection(c, remote.ToCh, remote.FromCh, remote.ErrCh, params.Skip)
	peerImpl := peer.NewPeerImpl(readHandshake, connection, peer.Incoming, remote, cancel)

	out := peer.InfoMessage{
		Peer: peerImpl,
		Value: &peer.Connected{
			Peer: peerImpl,
		},
	}
	params.Parent.InfoCh <- out

	return peer.Handle(peer.HandlerParams{
		Ctx:              ctx,
		ID:               peerImpl.ID(),
		Connection:       connection,
		Remote:           remote,
		Parent:           params.Parent,
		Peer:             peerImpl,
		DuplicateChecker: params.DuplicateChecker,
	})
}
