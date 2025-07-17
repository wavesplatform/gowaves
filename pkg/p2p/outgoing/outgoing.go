package outgoing

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const outgoingPeerDialTimeout = 5 * time.Second

type EstablishParams struct {
	Address      proto.TCPAddr
	WavesNetwork string
	Parent       peer.Parent
	DeclAddr     proto.TCPAddr
	Skip         conn.SkipFilter
	NodeName     string
	NodeNonce    uint64
}

func EstablishConnection(ctx context.Context, params EstablishParams, v proto.Version, logger, dl *slog.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	remote := peer.NewRemote()
	p := connector{
		params: params,
		remote: remote,
	}
	addr := params.Address.String()

	connection, handshake, err := p.connect(ctx, addr, outgoingPeerDialTimeout, v, logger)
	if err != nil {
		logger.Debug("Failed to establish outgoing connection", "address", addr, "error", err)
		return errors.Wrapf(err, "%q", addr)
	}

	peerImpl, err := peer.NewPeerImpl(handshake, connection, peer.Outgoing, remote, cancel, dl)
	if err != nil {
		if err := connection.Close(); err != nil {
			slog.Error("Failed to close outgoing connection to '%s': %v", addr, err)
		}
		logger.Debug("Failed to create peer for outgoing connection", "address", addr, "error", err)
		return errors.Wrapf(err, "failed to establish connection to %s", addr)
	}
	logger.Debug("Connected outgoing peer", "address", addr, "peer", peerImpl.ID())
	return peer.Handle(ctx, peerImpl, params.Parent, remote, logger, dl)
}

type connector struct {
	params EstablishParams
	remote peer.Remote
}

//nolint:nonamedreturns // For deferred close.
func (a *connector) connect(
	ctx context.Context, addr string, dialTimeout time.Duration, v proto.Version, logger *slog.Logger,
) (_ conn.Connection, _ proto.Handshake, err error) {
	dialer := net.Dialer{Timeout: dialTimeout}
	c, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, proto.Handshake{}, errors.Wrapf(err, "failed to dial with addr %q", addr)
	}
	defer func() {
		if err != nil { // close connection on error
			if err := c.Close(); err != nil {
				slog.Error("Failed to close outgoing connection", "address", addr, "error", err)
			}
		}
	}()

	handshake := proto.Handshake{
		AppName:      a.params.WavesNetwork,
		Version:      v,
		NodeName:     a.params.NodeName,
		NodeNonce:    a.params.NodeNonce,
		DeclaredAddr: proto.HandshakeTCPAddr(a.params.DeclAddr),
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}

	if _, err := handshake.WriteTo(c); err != nil {
		addr := a.params.Address.String()
		logger.Debug("Failed to send handshake", "address", addr, "error", err)
		return nil, proto.Handshake{}, errors.Wrapf(err, "failed to send handshake with addr %q", addr)
	}
	select {
	case <-ctx.Done():
		return nil, proto.Handshake{}, errors.Wrap(ctx.Err(), "connector.connect")
	default:
	}

	if _, err := handshake.ReadFrom(c); err != nil {
		addr := a.params.Address.String()
		logger.Debug("Failed to read handshake ", "address", a.params.Address.String(), "error", err)
		return nil, proto.Handshake{}, errors.Wrapf(err, "failed to read handshake with addr %q", addr)
	}
	select {
	case <-ctx.Done():
		return nil, proto.Handshake{}, errors.Wrap(ctx.Err(), "connector.connect")
	default:
	}

	return conn.WrapConnection(ctx, c, a.remote.ToCh, a.remote.FromCh, a.remote.ErrCh, a.params.Skip, logger),
		handshake, nil
}
