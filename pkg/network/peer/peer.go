package peer

import (
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Direction int

const Incoming Direction = 1
const Outgoing Direction = 2

func (a Direction) String() string {
	switch a {
	case Incoming:
		return "Incoming"
	case Outgoing:
		return "Outgoing"
	default:
		return "Unknown"
	}
}

type ProtoMessage struct {
	ID      string
	Message proto.Message
}

type InfoMessage struct {
	ID    string
	Value interface{}
}

type Connected struct {
	Peer       Peer
	Version    proto.Version
	DeclAddr   proto.PeerInfo
	LocalAddr  string
	RemoteAddr string
}

type ReceiveFromRemoteCallback func(b []byte, address string, resendTo chan ProtoMessage, pool conn.Pool)

type remote struct {
	toCh   chan []byte
	fromCh chan []byte
	errCh  chan error
}

func newRemote() remote {
	return remote{
		toCh:   make(chan []byte, 10),
		fromCh: make(chan []byte, 10),
		errCh:  make(chan error, 10),
	}
}

type Parent struct {
	MessageCh chan ProtoMessage
	InfoCh    chan InfoMessage
}

func NewParent() Parent {
	return Parent{
		MessageCh: make(chan ProtoMessage, 100),
		InfoCh:    make(chan InfoMessage, 100),
	}
}

type Peer interface {
	Direction() Direction
	Close()
	SendMessage(proto.Message)
	ID() string
}
