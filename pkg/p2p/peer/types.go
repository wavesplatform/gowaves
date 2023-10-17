package peer

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ProtoMessage struct {
	ID      Peer
	Message proto.Message
}

type Notification interface {
	peerNotificationTypeMaker()
}

type ConnectedNotification struct {
	Peer Peer
}

func (n ConnectedNotification) peerNotificationTypeMaker() {}

type DisconnectedNotification struct {
	Peer Peer
	Err  error
}

func (n DisconnectedNotification) peerNotificationTypeMaker() {}

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
