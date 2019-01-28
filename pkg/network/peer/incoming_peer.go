package peer

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"time"
)

type IncomingPeer struct {
	params  IncomingPeerParams
	conn    conn.Connection
	remote  remote
	address string
	cancel  context.CancelFunc
}

type IncomingPeerParams struct {
	Ctx                       context.Context
	Conn                      net.Conn
	ReceiveFromRemoteCallback ReceiveFromRemoteCallback
	Parent                    Parent
	DeclAddr                  proto.PeerInfo
	Pool                      conn.Pool
}

func RunIncomingPeer(params IncomingPeerParams) {
	c := params.Conn
	bytes, err := params.DeclAddr.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		c.Close()
		return
	}

	readHandshake := proto.Handshake{}
	_, err = readHandshake.ReadFrom(c)
	if err != nil {
		zap.S().Error("failed to read handshake: ", err)
		c.Close()
		return
	}

	writeHandshake := proto.Handshake{
		Name:              "wavesW",
		Version:           proto.Version{Major: 0, Minor: 15, Patch: 0},
		NodeName:          "gowaves",
		NodeNonce:         0x0,
		DeclaredAddrBytes: bytes,
		Timestamp:         proto.NewTimestampFromTime(time.Now()),
	}

	_, err = writeHandshake.WriteTo(c)
	if err != nil {
		zap.S().Error("failed to write handshake: ", err)
		c.Close()
		return
	}

	remote := newRemote()

	connection := conn.WrapConnection(c, params.Pool, remote.toCh, remote.fromCh, remote.errCh)

	_, cancel := context.WithCancel(params.Ctx)

	peer := &IncomingPeer{
		params:  params,
		conn:    connection,
		remote:  remote,
		address: c.RemoteAddr().String(),
		cancel:  cancel,
	}

	out := InfoMessage{
		ID: c.RemoteAddr().String(),
		Value: &Connected{
			Peer:    peer,
			Version: readHandshake.Version,
		},
	}
	params.Parent.ParentInfoChan <- out
	peer.run()
}

func (a *IncomingPeer) run() {
	handleParams := handlerParams{
		connection:                a.conn,
		ctx:                       a.params.Ctx,
		remote:                    a.remote,
		receiveFromRemoteCallback: a.params.ReceiveFromRemoteCallback,
		address:                   a.address,
		parent:                    a.params.Parent,
		pool:                      a.params.Pool,
	}
	handle(handleParams)
}

func (a *IncomingPeer) Close() {
	a.cancel()
}

func (a *IncomingPeer) SendMessage(m proto.Message) {
	b, err := m.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return
	}
	select {
	case a.remote.toCh <- b:
	default:
		zap.S().Warnf("can't send bytes to remote, chan is full id %s", a.address)
	}
}

func (a *IncomingPeer) ID() string {
	return a.address
}

func (a *IncomingPeer) Direction() Direction {
	return Incoming
}
