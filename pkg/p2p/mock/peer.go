package mock

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Peer struct {
	Addr                  string
	SendMessageCalledWith []proto.Message
	IncomeCh              chan peer.ProtoMessage
	HandshakeField        proto.Handshake
	RemoteAddress         proto.TCPAddr
}

func NewPeer() *Peer {
	return &Peer{}
}

func (a Peer) RemoteAddr() proto.TCPAddr {
	return a.RemoteAddress
}

func (Peer) Direction() peer.Direction {
	panic("implement me")
}

func (Peer) Reconnect() error {
	panic("implement me")
}

func (Peer) Close() error {
	panic("implement me")
}

func (Peer) Connection() conn.Connection {
	panic("implement me")
}

func (a *Peer) SendMessage(m proto.Message) {
	a.SendMessageCalledWith = append(a.SendMessageCalledWith, m)
}

func (a Peer) ID() string {
	return a.Addr
}

func (a Peer) Handshake() proto.Handshake {
	return a.HandshakeField
}
