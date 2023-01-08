package peer

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Connected struct {
	Peer Peer
}

func (*Connected) infoMsgValueMark() {}

type InternalErr struct {
	Err error
}

func (*InternalErr) infoMsgValueMark() {}

type ProtoMessage struct {
	ID      Peer
	Message proto.Message
}

type InfoMessage struct {
	Peer  Peer
	Value InfoMessageValue
}

type InfoMessageValue interface {
	infoMsgValueMark()
}

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

type ID interface {
	fmt.Stringer
}
