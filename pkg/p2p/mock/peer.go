package mock

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MockPeer struct {
	Addr                  string
	SendMessageCalledWith []proto.Message
	IncomeCh              chan peer.ProtoMessage
	HandshakeField        proto.Handshake
	RemoteAddress         proto.TCPAddr
}

func NewPeer() *MockPeer {
	return &MockPeer{}
}

func (a MockPeer) RemoteAddr() proto.TCPAddr {
	return a.RemoteAddress
}

func (MockPeer) Direction() peer.Direction {
	panic("implement me")
}

func (MockPeer) Reconnect() error {
	panic("implement me")
}

func (MockPeer) Close() error {
	panic("implement me")
}

func (MockPeer) Connection() conn.Connection {
	panic("implement me")
}

func (a *MockPeer) SendMessage(m proto.Message) {
	a.SendMessageCalledWith = append(a.SendMessageCalledWith, m)
}

func (a MockPeer) ID() string {
	return a.Addr
}

func (a MockPeer) Handshake() proto.Handshake {
	return a.HandshakeField
}
