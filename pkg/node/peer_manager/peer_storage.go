package peer_manager

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerStorage interface {
	SavePeers([]proto.TCPAddr) error
	Peers() ([]proto.TCPAddr, error)
}
