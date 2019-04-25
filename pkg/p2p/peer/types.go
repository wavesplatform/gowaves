package peer

import "github.com/wavesplatform/gowaves/pkg/proto"

type Connected struct {
	Peer Peer
}

type ProtoMessage struct {
	ID      string
	Message proto.Message
}

type InfoMessage struct {
	ID    string
	Value interface{}
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
