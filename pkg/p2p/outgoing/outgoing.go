package outgoing

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

const outgoingPeerDialTimeout = 5 * time.Second

type DuplicateChecker interface {
	Add([]byte) (isNew bool)
}

type EstablishParams struct {
	Address          proto.TCPAddr
	WavesNetwork     string
	Parent           peer.Parent
	DeclAddr         proto.TCPAddr
	Skip             conn.SkipFilter
	NodeName         string
	NodeNonce        uint64
	DuplicateChecker DuplicateChecker
}

func EstablishConnection(ctx context.Context, params EstablishParams, v proto.Version) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	remote := peer.NewRemote()
	p := connector{
		params: params,
		remote: remote,
	}
	addr := params.Address.String()

	connection, handshake, err := p.connect(ctx, addr, outgoingPeerDialTimeout, v)
	if err != nil {
		zap.S().Debugf("Outgoing connection to address '%s' failed with error: %v", addr, err)
		return errors.Wrapf(err, "%q", addr)
	}

	peerImpl, err := peer.NewPeerImpl(handshake, connection, peer.Outgoing, remote, cancel)
	if err != nil {
		if err := connection.Close(); err != nil {
			zap.S().Errorf("Failed to close outgoing connection to '%s': %v", addr, err)
		}
		zap.S().Debugf("Failed to create new peer impl for outgoing conn to %s: %v", addr, err)
		return errors.Wrapf(err, "failed to establish connection to %s", addr)
	}
	zap.S().Debugf("Connected outgoing peer with addr '%s', id '%s'", addr, peerImpl.ID())
	return peer.Handle(ctx, peerImpl, params.Parent, remote, params.DuplicateChecker)
}

type connector struct {
	params EstablishParams
	remote peer.Remote
}

func (a *connector) connect(ctx context.Context, addr string, dialTimeout time.Duration, v proto.Version) (_ conn.Connection, _ proto.Handshake, err error) {
	dialer := net.Dialer{Timeout: dialTimeout}
	c, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, proto.Handshake{}, errors.Wrapf(err, "failed to dial with addr %q", addr)
	}
	defer func() {
		if err != nil { // close connection on error
			if err := c.Close(); err != nil {
				zap.S().Errorf("Failed to close outgoing connection to '%s': %v", addr, err)
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
		zap.S().Infof("Failed to send handshake with addr %q: %v", addr, err)
		return nil, proto.Handshake{}, errors.Wrapf(err, "failed to send handshake with addr %q", addr)
	}
	select {
	case <-ctx.Done():
		return nil, proto.Handshake{}, errors.Wrap(ctx.Err(), "connector.connect")
	default:
	}

	if _, err := handshake.ReadFrom(c); err != nil {
		addr := a.params.Address.String()
		zap.S().Infof("Failed to read handshake with addr %q: %v", a.params.Address.String(), err)
		return nil, proto.Handshake{}, errors.Wrapf(err, "failed to read handshake with addr %q", addr)
	}
	select {
	case <-ctx.Done():
		return nil, proto.Handshake{}, errors.Wrap(ctx.Err(), "connector.connect")
	default:
	}

	return conn.WrapConnection(ctx, c, a.remote.ToCh, a.remote.FromCh, a.remote.ErrCh, a.params.Skip), handshake, nil
}
