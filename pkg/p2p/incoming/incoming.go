package incoming

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerParams struct {
	WavesNetwork string
	Conn         net.Conn
	Parent       peer.Parent
	DeclAddr     proto.TCPAddr
	Skip         conn.SkipFilter
	NodeName     string
	NodeNonce    uint64
	Version      proto.Version
}

func RunIncomingPeer(ctx context.Context, params PeerParams, logger, dl *slog.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return runIncomingPeer(ctx, cancel, params, logger, dl)
}

func runIncomingPeer(ctx context.Context, cancel context.CancelFunc, params PeerParams, logger, dl *slog.Logger) error {
	c := params.Conn

	readHandshake := proto.Handshake{}
	_, err := readHandshake.ReadFrom(c)
	if err != nil {
		logger.Debug("Failed to read handshake", logging.Error(err))
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
		logger.Debug("Failed to write handshake", logging.Error(err))
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
	connection := conn.WrapConnection(ctx, c, remote.ToCh, remote.FromCh, remote.ErrCh, params.Skip, logger)
	peerImpl, err := peer.NewPeerImpl(readHandshake, connection, peer.Incoming, remote, cancel, dl)
	if err != nil {
		if clErr := connection.Close(); clErr != nil {
			slog.Error("Failed to close incoming connection", logging.Error(clErr))
		}
		slog.Warn("Failed to create new peer impl", logging.Error(err))
		return errors.Wrap(err, "failed to run incoming peer")
	}
	return peer.Handle(ctx, peerImpl, params.Parent, remote, logger, dl)
}
