package peer_manager

import "github.com/wavesplatform/gowaves/pkg/proto"

type MemoryPeerStorage struct {
	peers []proto.TCPAddr
}

func (a *MemoryPeerStorage) SavePeers(peers []proto.TCPAddr) error {
	b := make([]proto.TCPAddr, len(peers))
	copy(b, peers)
	a.peers = b
	return nil
}

func (a *MemoryPeerStorage) Peers() ([]proto.TCPAddr, error) {
	return a.peers, nil
}
