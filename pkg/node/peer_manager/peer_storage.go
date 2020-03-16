package peer_manager

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

type PeerStorage interface {
	SavePeers([]proto.TCPAddr) error
	Peers() ([]proto.TCPAddr, error)
	Mutex() *lock.RwMutex
}
