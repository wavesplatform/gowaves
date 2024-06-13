package peer

import (
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	defaultChannelSize = 100
	lightModeChannelSize = 2 *defaultChannelSize
	errorChannelSize   = 10
)

type Remote struct {
	ToCh   chan []byte
	FromCh chan *bytebufferpool.ByteBuffer
	ErrCh  chan error
}

func NewRemote() Remote {
	return Remote{
		ToCh:   make(chan []byte, lightModeChannelSize),
		FromCh: make(chan *bytebufferpool.ByteBuffer, lightModeChannelSize),
		ErrCh:  make(chan error, errorChannelSize),
	}
}

type Parent struct {
	NetworkMessagesCh chan ProtoMessage
	NodeMessagesCh    chan ProtoMessage
	HistoryMessagesCh chan ProtoMessage
	NotificationsCh   chan Notification
	SkipMessageList   *messages.SkipMessageList
}

func NewParent(enableLightNode bool) Parent {
	channelSize := defaultChannelSize
	if enableLightNode {
		// because in light node we send block and snapshot request messages
		channelSize = lightModeChannelSize
	}
	return Parent{
		NetworkMessagesCh: make(chan ProtoMessage, channelSize),
		NodeMessagesCh:    make(chan ProtoMessage, channelSize),
		HistoryMessagesCh: make(chan ProtoMessage, channelSize),
		NotificationsCh:   make(chan Notification, channelSize),
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
