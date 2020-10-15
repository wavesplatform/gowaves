package peer_manager

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerStorage interface {
	All() ([]proto.TCPAddr, error)
	Known() ([]proto.TCPAddr, error)
	AddKnown(proto.TCPAddr) error
	Add([]proto.TCPAddr) error
}
