package peer

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"time"
)

type OutgoingPeerParams struct {
	Ctx                       context.Context
	Address                   string
	Parent                    Parent
	ReceiveFromRemoteCallback ReceiveFromRemoteCallback
	Pool                      conn.Pool
	DeclAddr                  proto.PeerInfo
}

type OutgoingPeer struct {
	params     OutgoingPeerParams
	ctx        context.Context
	cancel     context.CancelFunc
	remote     remote
	connection conn.Connection
}

func RunOutgoingPeer(params OutgoingPeerParams) {
	ctx, cancel := context.WithCancel(params.Ctx)
	remote := newRemote()
	p := OutgoingPeer{
		params: params,
		ctx:    ctx,
		cancel: cancel,
		remote: remote,
	}

	connection, version, err := p.connect(remote, params.DeclAddr)
	if err != nil {
		zap.S().Error(err)
		return
	}
	p.connection = connection

	connected := InfoMessage{
		ID: params.Address,
		Value: &Connected{
			Peer:    &p,
			Version: version,
		},
	}
	params.Parent.ParentInfoChan <- connected
	zap.S().Debugf("connected %s", params.Address)

	handle(handlerParams{
		ctx:                       ctx,
		address:                   params.Address,
		connection:                p.connection,
		remote:                    remote,
		receiveFromRemoteCallback: params.ReceiveFromRemoteCallback,
		parent:                    params.Parent,
		pool:                      params.Pool,
	})
}

func (a *OutgoingPeer) connect(remote remote, declAddr proto.PeerInfo) (conn.Connection, proto.Version, error) {
	possibleVersions := []uint32{15, 14, 16}
	index := 0

	for {
		c, err := net.Dial("tcp", a.params.Address)
		if err != nil {
			zap.S().Infof("failed to connect, %s id %s", err, a.params.Address)
			select {
			case <-a.ctx.Done():
				return nil, proto.Version{}, a.ctx.Err()
			case <-time.After(5 * time.Minute):
				continue
			}
		}

		bytes, err := declAddr.MarshalBinary()
		if err != nil {
			zap.S().Error(err)
			return nil, proto.Version{}, err
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
			zap.S().Error("failed to send handshake: ", err)
			continue
		}
		_, err = handshake.ReadFrom(c)
		if err != nil {
			zap.S().Error("failed to read handshake: ", err)
			index += 1
			continue
		}

		return conn.WrapConnection(c, a.params.Pool, remote.toCh, remote.fromCh, remote.errCh),
			proto.Version{Major: 0, Minor: possibleVersions[index%len(possibleVersions)]},
			nil

	}
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
