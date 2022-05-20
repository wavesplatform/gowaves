package peer

import (
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Remote struct {
	ToCh   chan []byte
	FromCh chan *bytebufferpool.ByteBuffer
	ErrCh  chan error
}

func NewRemote() Remote {
	return Remote{
		ToCh:   make(chan []byte, 100),
		FromCh: make(chan *bytebufferpool.ByteBuffer, 100),
		ErrCh:  make(chan error, 10),
	}
}

type Parent struct {
	MessageCh              chan ProtoMessage
	InfoCh                 chan InfoMessage
	ListOfExcludedCh       chan []uint8
	ListOfExcludedMessages []uint8
}

func NewParent() Parent {
	return Parent{
		MessageCh:              make(chan ProtoMessage, 100),
		InfoCh:                 make(chan InfoMessage, 100),
		ListOfExcludedCh:       make(chan []uint8, 1),
		ListOfExcludedMessages: make([]uint8, 0),
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
