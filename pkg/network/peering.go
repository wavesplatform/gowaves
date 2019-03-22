package network

import (
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"time"
)

type Identity struct {
	IP    net.IP
	Nonce uint64
}

type Direction int

const (
	Inbound Direction = iota
	Outbound
)

func (d Direction) String() string {
	switch d {
	case Inbound:
		return "inbound"
	case Outbound:
		return "outbound"
	default:
		return "unset"
	}
}

type Reason int

const (
	Continuation Reason = iota
	ConnectionFailure
	HandshakeFailure
)

type Suspension struct {
	Until  time.Time
	Reason Reason
}

var NoSuspension = Suspension{Reason: Continuation, Until: time.Unix(1<<63-62135596801, 999999999)}

func (s Suspension) String() string {
	switch s.Reason {
	case Continuation:
		return "not suspended"
	case ConnectionFailure:
		return s.withTime("connection failure")
	case HandshakeFailure:
		return s.withTime("handshake failure")
	}
	return "undefined"
}

func (s Suspension) withTime(reason string) string {
	return fmt.Sprintf("suspended due to %s until %s", reason, s.Until)
}

type Entry struct {
	Peer       proto.PeerAddress
	Version    proto.Version
	Direction  Direction
	Created    time.Time
	Accessed   time.Time
	Suspension Suspension
}

type Contact struct {
	Conn  net.Conn
	Entry Entry
}
