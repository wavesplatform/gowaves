package peer

import "github.com/wavesplatform/gowaves/pkg/proto"

type Peer interface {
	Direction() Direction
	Reconnect() error
	Close()
	SendMessage(proto.Message)
	ID() UniqID
}
