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

type Remote struct {
	ToCh   chan []byte
	FromCh chan []byte
	ErrCh  chan error
}

func NewRemote() Remote {
	return Remote{
		ToCh:   make(chan []byte, 10),
		FromCh: make(chan []byte, 10),
		ErrCh:  make(chan error, 10),
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
	Connection() conn.Connection
	Handshake() proto.Handshake
}
