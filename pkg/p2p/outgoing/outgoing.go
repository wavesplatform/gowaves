package outgoing

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
		logger.Debug("Failed to establish outgoing connection", slog.String("address", addr), logging.Error(err))
		return errors.Wrapf(err, "%q", addr)
	}

	peerImpl, err := peer.NewPeerImpl(handshake, connection, peer.Outgoing, remote, cancel, dl)
	if err != nil {
		if err := connection.Close(); err != nil {
			slog.Error("Failed to close outgoing connection", slog.String("address", addr), logging.Error(err))
		}
		logger.Debug("Failed to create peer for outgoing connection", slog.String("address", addr),
			logging.Error(err))
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
			if clErr := c.Close(); clErr != nil {
				slog.Error("Failed to close outgoing connection", slog.String("address", addr), logging.Error(clErr))
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

	if _, wErr := handshake.WriteTo(c); wErr != nil {
		pa := a.params.Address.String()
		logger.Debug("Failed to send handshake", slog.String("address", pa), logging.Error(wErr))
		return nil, proto.Handshake{}, errors.Wrapf(wErr, "failed to send handshake with addr %q", pa)
	}
	select {
	case <-ctx.Done():
		return nil, proto.Handshake{}, errors.Wrap(ctx.Err(), "connector.connect")
	default:
	}

	if _, rErr := handshake.ReadFrom(c); rErr != nil {
		pa := a.params.Address.String()
		logger.Debug("Failed to read handshake", slog.String("address", pa), logging.Error(rErr))
		return nil, proto.Handshake{}, errors.Wrapf(rErr, "failed to read handshake with addr %q", pa)
	}
	select {
	case <-ctx.Done():
		return nil, proto.Handshake{}, errors.Wrap(ctx.Err(), "connector.connect")
	default:
	}

	return conn.WrapConnection(ctx, c, a.remote.ToCh, a.remote.FromCh, a.remote.ErrCh, a.params.Skip, logger),
		handshake, nil
}
