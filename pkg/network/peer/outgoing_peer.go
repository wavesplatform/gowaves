package peer

import (
	"context"
	"net"
	"time"

	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type OutgoingPeerParams struct {
	Address      string
	WavesNetwork string
	Parent       Parent
	Pool         bytespool.Pool
	DeclAddr     proto.PeerInfo
	Skip         conn.SkipFilter
}

type OutgoingPeer struct {
	params     OutgoingPeerParams
	cancel     context.CancelFunc
	remote     remote
	connection conn.Connection
}

func RunOutgoingPeer(ctx context.Context, params OutgoingPeerParams) {
	if params.DeclAddr.String() == params.Address {
		zap.S().Errorf("trying to connect to myself")
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	remote := newRemote()
	p := OutgoingPeer{
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

	version := handshake.Version
	declAddr, err := handshake.PeerInfo()
	if err != nil {
		zap.S().Info(err, params.Address)
	}

	connected := InfoMessage{
		ID: params.Address,
		Value: &Connected{
			Peer:       &p,
			Version:    version,
			DeclAddr:   declAddr,
			RemoteAddr: connection.Conn().RemoteAddr().String(),
			LocalAddr:  connection.Conn().LocalAddr().String(),
			AppName:    handshake.AppName,
			NodeName:   handshake.NodeName,
		},
	}
	params.Parent.InfoCh <- connected
	zap.S().Debugf("connected %s", params.Address)

	handle(handlerParams{
		ctx:        ctx,
		id:         params.Address,
		connection: p.connection,
		remote:     remote,
		parent:     params.Parent,
		pool:       params.Pool,
	})
}

func (a *OutgoingPeer) connect(ctx context.Context, wavesNetwork string, remote remote, declAddr proto.PeerInfo) (conn.Connection, *proto.Handshake, error) {
	possibleVersions := []uint32{15, 14, 16}
	index := 0

	for i := 0; i < 20; i++ {

		c, err := net.Dial("tcp", a.params.Address)
		if err != nil {
			zap.S().Infof("failed to connect, %s id %s", err, a.params.Address)
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(5 * time.Minute):
				continue
			}
		}

		bytes, err := declAddr.MarshalBinary()
		if err != nil {
			zap.S().Error(err)
			return nil, nil, err
		}

		handshake := proto.Handshake{
			AppName:           wavesNetwork,
			Version:           proto.Version{Major: 0, Minor: possibleVersions[index%len(possibleVersions)], Patch: 0},
			NodeName:          "retransmitter",
			NodeNonce:         0x0,
			DeclaredAddrBytes: bytes,
			Timestamp:         proto.NewTimestampFromTime(time.Now()),
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
		return conn.WrapConnection(c, a.params.Pool, remote.toCh, remote.fromCh, remote.errCh, a.params.Skip), &handshake, nil
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
	case a.remote.toCh <- b:
	default:
		zap.S().Warnf("can't send bytes to remote, chan is full id %s", a.params.Address)
	}
}

func (a *OutgoingPeer) Direction() Direction {
	return Outgoing
}

func (a *OutgoingPeer) Close() {
	a.cancel()
}

func (a *OutgoingPeer) ID() string {
	return a.params.Address
}

func (a *OutgoingPeer) Connection() conn.Connection {
	return a.connection
}
