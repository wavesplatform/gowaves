package network

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type OutgoingPeerParams struct {
	Address      string
	WavesNetwork string
	Parent       peer.Parent
	Pool         bytespool.Pool
	DeclAddr     proto.TCPAddr
	Skip         conn.SkipFilter
}

type OutgoingPeer struct {
	params     OutgoingPeerParams
	cancel     context.CancelFunc
	remote     peer.Remote
	connection conn.Connection
	handshake  proto.Handshake
}

func RunOutgoingPeer(ctx context.Context, params OutgoingPeerParams) {
	if params.DeclAddr.String() == params.Address {
		zap.S().Errorf("trying to connect to myself")
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	remote := peer.NewRemote()
	p := &OutgoingPeer{
		params: params,
		cancel: cancel,
		remote: remote,
	}

	connection, handshake, err := p.connect(ctx, params.WavesNetwork, remote, params.DeclAddr)
	if err != nil {
		zap.S().Error(err, params.Address)
		return
	}
	p.connection = connection
	p.handshake = *handshake

	connected := peer.InfoMessage{
		Peer: p,
		Value: &peer.Connected{
			Peer: p,
		},
	}
	params.Parent.InfoCh <- connected
	zap.S().Debugf("connected %s", params.Address)

	if err := peer.Handle(peer.HandlerParams{
		Ctx:        ctx,
		ID:         params.Address,
		Connection: p.connection,
		Remote:     remote,
		Parent:     params.Parent,
		Pool:       params.Pool,
	}); err != nil {
		zap.S().Errorf("peer.Handle(): %v\n", err)
		return
	}
}

func (a *OutgoingPeer) connect(ctx context.Context, wavesNetwork string, remote peer.Remote, declAddr proto.TCPAddr) (conn.Connection, *proto.Handshake, error) {
	possibleVersions := []uint32{15, 14, 16}
	index := 0

	for i := 0; i < 20; i++ {

		c, err := net.Dial("tcp", a.params.Address)
		if err != nil {
			zap.S().Infof("failed to connect, %s ID %s", err, a.params.Address)
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(5 * time.Minute):
				continue
			}
		}

		handshake := proto.Handshake{
			AppName:      wavesNetwork,
			Version:      proto.Version{Major: 0, Minor: possibleVersions[index%len(possibleVersions)], Patch: 0},
			NodeName:     "retransmitter",
			NodeNonce:    0x0,
			DeclaredAddr: proto.HandshakeTCPAddr(declAddr),
			Timestamp:    proto.NewTimestampFromTime(time.Now()),
		}

		_, err = handshake.WriteTo(c)
		if err != nil {
			zap.S().Error("failed to send handshake: ", err, a.params.Address)
			continue
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
			index += 1
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(5 * time.Minute):
				continue
			}
		}
		return conn.WrapConnection(c, a.params.Pool, remote.ToCh, remote.FromCh, remote.ErrCh, a.params.Skip), &handshake, nil
	}

	return nil, nil, errors.Errorf("can't connect 20 times")
}

func (a *OutgoingPeer) SendMessage(m proto.Message) {
	b, err := m.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return
	}
	select {
	case a.remote.ToCh <- b:
	default:
		zap.S().Warnf("can't send bytes to Remote, chan is full ID %s", a.params.Address)
	}
}

func (a *OutgoingPeer) Direction() peer.Direction {
	return peer.Outgoing
}

func (a *OutgoingPeer) Close() error {
	a.cancel()
	return nil
}

func (a *OutgoingPeer) ID() string {
	return a.params.Address
}

func (a *OutgoingPeer) Connection() conn.Connection {
	return a.connection
}

func (a *OutgoingPeer) Handshake() proto.Handshake {
	return a.handshake
}

func (a *OutgoingPeer) RemoteAddr() proto.TCPAddr {
	return proto.TCPAddr(*a.connection.Conn().RemoteAddr().(*net.TCPAddr))
}
