package peer

import (
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
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
	MessageCh       chan ProtoMessage
	InfoCh          chan InfoMessage
	SkipMessageList *messages.SkipMessageList
}

func NewParent() Parent {
	return Parent{
		MessageCh:       make(chan ProtoMessage, 100),
		InfoCh:          make(chan InfoMessage, 100),
		SkipMessageList: &messages.SkipMessageList{},
	}
}

//go:generate moq -out peer_moq.go ./ Peer:mockPeer
type Peer interface {
	Direction() Direction
	Close() error
	SendMessage(proto.Message)
	ID() ID
	Connection() conn.Connection
	Handshake() proto.Handshake
	RemoteAddr() proto.TCPAddr
}
