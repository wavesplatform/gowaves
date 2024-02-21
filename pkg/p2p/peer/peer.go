package peer

import (
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	defaultChSize     = 100
	chSizeInLightMode = defaultChSize * 2
)

type Remote struct {
	ToCh   chan []byte
	FromCh chan *bytebufferpool.ByteBuffer
	ErrCh  chan error
}

func NewRemote() Remote {
	return Remote{
		ToCh:   make(chan []byte, chSizeInLightMode),
		FromCh: make(chan *bytebufferpool.ByteBuffer, chSizeInLightMode),
		ErrCh:  make(chan error, 10),
	}
}

type Parent struct {
	MessageCh       chan ProtoMessage
	InfoCh          chan InfoMessage
	SkipMessageList *messages.SkipMessageList
}

func NewParent(enableLightNode bool) Parent {
	messageChSize := defaultChSize
	if enableLightNode {
		// because in light node we send block and snapshot request messages
		messageChSize = chSizeInLightMode
	}
	return Parent{
		MessageCh:       make(chan ProtoMessage, messageChSize),
		InfoCh:          make(chan InfoMessage, messageChSize),
		SkipMessageList: &messages.SkipMessageList{},
	}
}

//go:generate moq -out peer_moq.go . Peer:mockPeer
type Peer interface {
	Direction() Direction
	Close() error
	SendMessage(proto.Message)
	ID() ID
	Connection() conn.Connection
	Handshake() proto.Handshake
	RemoteAddr() proto.TCPAddr
	Equal(Peer) bool
}
