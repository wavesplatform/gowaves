package outgoing

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

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

func EstablishConnection(ctx context.Context, params EstablishParams, v proto.Version) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	remote := peer.NewRemote()
	p := connector{
		params: params,
		remote: remote,
	}

	connection, handshake, err := p.connect(ctx, outgoingPeerDialTimeout, v)
	if err != nil {
		zap.S().Named(logging.NetworkNamespace).Debugf("Outgoing connection to address '%s' failed with error: %v",
			params.Address.String(), err)
		return err
	}

	peerImpl, err := peer.NewPeerImpl(handshake, connection, peer.Outgoing, remote, cancel)
	if err != nil {
		if clErr := connection.Close(); clErr != nil {
			zap.S().Named(logging.NetworkNamespace).
				Errorf("Failed to close outgoing connection to '%s': %v", params.Address.String(), clErr)
		}
		zap.S().Named(logging.NetworkNamespace).
			Debugf("Failed to create new peer impl for outgoing conn to %s: %v", params.Address.String(), err)
		return errors.Wrapf(err, "failed to establish connection to %s", params.Address.String())
	}
	zap.S().Named(logging.NetworkNamespace).
		Debugf("Connected outgoing peer with addr '%s', id '%s'", params.Address.String(), peerImpl.ID())
	return peer.Handle(ctx, peerImpl, params.Parent, remote)
}

type connector struct {
	params EstablishParams
	remote peer.Remote
}

func (a *connector) connect(
	ctx context.Context, dialTimeout time.Duration, v proto.Version,
) (conn.Connection, proto.Handshake, error) {
	dialer := net.Dialer{Timeout: dialTimeout}
	c, err := dialer.DialContext(ctx, "tcp", a.params.Address.String())
	if err != nil {
		return nil, proto.Handshake{}, err
	}
	defer func() {
		if err != nil { // close connection on error
			if clErr := c.Close(); clErr != nil {
				zap.S().Named(logging.NetworkNamespace).
					Errorf("Failed to close outgoing connection with '%s': %v", a.params.Address.String(), clErr)
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
		zap.S().Named(logging.NetworkNamespace).
			Debugf("Failed to send handshake to '%s': %v", a.params.Address.String(), wErr)
		return nil, proto.Handshake{}, errors.Wrapf(wErr, "failed to send handshake to '%s'",
			a.params.Address.String())
	}
	select {
	case <-ctx.Done():
		return nil, proto.Handshake{}, errors.Wrap(ctx.Err(), "connector.connect")
	default:
	}

	if _, rErr := handshake.ReadFrom(c); rErr != nil {
		zap.S().Named(logging.NetworkNamespace).
			Debugf("Failed to read handshake from '%s': %v", a.params.Address.String(), rErr)
		return nil, proto.Handshake{}, errors.Wrapf(rErr, "failed to read handshake from addr %q",
			a.params.Address.String())
	}
	select {
	case <-ctx.Done():
		return nil, proto.Handshake{}, errors.Wrap(ctx.Err(), "connector.connect")
	default:
	}

	return conn.WrapConnection(ctx, c, a.remote.ToCh, a.remote.FromCh, a.remote.ErrCh, a.params.Skip), handshake, nil
}
