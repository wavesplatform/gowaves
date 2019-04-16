package peer

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Remote struct {
	ToCh   chan []byte
	FromCh chan []byte
	ErrCh  chan error
}

func NewRemote() Remote {
	return Remote{
		ToCh:   make(chan []byte, 150),
		FromCh: make(chan []byte, 150),
		ErrCh:  make(chan error, 10),
	}
}

type Parent struct {
	MessageCh chan ProtoMessage
	InfoCh    chan InfoMessage
}

func NewParent() Parent {
	return Parent{
		MessageCh: make(chan ProtoMessage, 1000),
		InfoCh:    make(chan InfoMessage, 100),
	}
}

type Peer interface {
	Direction() Direction
	Close() error
	SendMessage(proto.Message)
	ID() string
	Connection() conn.Connection
	Handshake() proto.Handshake
	RemoteAddr() proto.TCPAddr
}
