package peer

import (
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	defaultChannelSize = 100
	errorChannelSize   = 10
)

type Remote struct {
	ToCh   chan []byte
	FromCh chan *bytebufferpool.ByteBuffer
	ErrCh  chan error
}

func NewRemote() Remote {
	return Remote{
		ToCh:   make(chan []byte, defaultChannelSize),
		FromCh: make(chan *bytebufferpool.ByteBuffer, defaultChannelSize),
		ErrCh:  make(chan error, errorChannelSize),
	}
}

type Parent struct {
	NetworkMessagesCh chan ProtoMessage
	NodeMessagesCh    chan ProtoMessage
	NotificationsCh   chan Notification
	SkipMessageList   *messages.SkipMessageList
}

func NewParent() Parent {
	return Parent{
		NetworkMessagesCh: make(chan ProtoMessage, defaultChannelSize),
		NodeMessagesCh:    make(chan ProtoMessage, defaultChannelSize),
		NotificationsCh:   make(chan Notification, defaultChannelSize),
		SkipMessageList:   &messages.SkipMessageList{},
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
