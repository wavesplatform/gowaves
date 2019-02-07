package peer

import (
	"context"
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"time"
)

type spawnedDeleter interface {
	Delete(string)
}

type NoOpDeleter struct{}

func (NoOpDeleter) Delete(string) {}

type OutgoingPeerParams struct {
	Ctx                       context.Context
	Address                   string
	Parent                    Parent
	ReceiveFromRemoteCallback ReceiveFromRemoteCallback
	Pool                      conn.Pool
	DeclAddr                  proto.PeerInfo
	SpawnedPeers              spawnedDeleter
}

type OutgoingPeer struct {
	params     OutgoingPeerParams
	ctx        context.Context
	cancel     context.CancelFunc
	remote     remote
	connection conn.Connection
}

func RunOutgoingPeer(params OutgoingPeerParams) {
	// unsubscribe from spawned peer on exit
	defer params.SpawnedPeers.Delete(params.Address)

	ctx, cancel := context.WithCancel(params.Ctx)
	remote := newRemote()
	p := OutgoingPeer{
		params: params,
		ctx:    ctx,
		cancel: cancel,
		remote: remote,
	}

	connection, handshake, err := p.connect(remote, params.DeclAddr)
	if err != nil {
		zap.S().Error(err, params.Address)
		return
	}
	p.connection = connection

	version := handshake.Version
	declAddr, err := handshake.PeerInfo()
	if err != nil {
		zap.S().Error(err, params.Address)
	}

	connected := InfoMessage{
		ID: params.Address,
		Value: &Connected{
			Peer:       &p,
			Version:    version,
			DeclAddr:   declAddr,
			RemoteAddr: connection.Conn().RemoteAddr().String(),
			LocalAddr:  connection.Conn().LocalAddr().String(),
		},
	}
	params.Parent.InfoCh <- connected
	zap.S().Debugf("connected %s", params.Address)

	handle(handlerParams{
		ctx:                       ctx,
		uniqueID:                  params.Address,
		connection:                p.connection,
		remote:                    remote,
		receiveFromRemoteCallback: params.ReceiveFromRemoteCallback,
		parent:                    params.Parent,
		pool:                      params.Pool,
	})
}

func (a *OutgoingPeer) connect(remote remote, declAddr proto.PeerInfo) (conn.Connection, *proto.Handshake, error) {
	possibleVersions := []uint32{15, 14, 16}
	index := 0

	for i := 0; i < 20; i++ {

		c, err := net.Dial("tcp", a.params.Address)
		if err != nil {
			zap.S().Infof("failed to connect, %s id %s", err, a.params.Address)
			select {
			case <-a.ctx.Done():
				return nil, nil, a.ctx.Err()
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
			Name:              "wavesW",
			Version:           proto.Version{Major: 0, Minor: possibleVersions[index%len(possibleVersions)], Patch: 0},
			NodeName:          "gowaves",
			NodeNonce:         0x0,
			DeclaredAddrBytes: bytes,
			Timestamp:         proto.NewTimestampFromTime(time.Now()),
		}

		_, err = handshake.WriteTo(c)
		if err != nil {
			zap.S().Error("failed to send handshake: ", err, a.params.Address)
			continue
		}
		_, err = handshake.ReadFrom(c)
		if err != nil {
			zap.S().Debugf("failed to read handshake: %s %s", err, a.params.Address)
			index += 1
			select {
			case <-a.ctx.Done():
				return nil, nil, a.ctx.Err()
			case <-time.After(5 * time.Minute):
				continue
			}
		}
		return conn.WrapConnection(c, a.params.Pool, remote.toCh, remote.fromCh, remote.errCh), &handshake, nil
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
