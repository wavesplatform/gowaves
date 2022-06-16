package peer_manager

import (
	"math/big"
	"sort"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
)

type ActivePeers struct {
	m             map[peer.PeerID]peerInfo
	sortedByScore []peer.PeerID
}

func NewActivePeers() ActivePeers {
	return ActivePeers{
		m:             make(map[peer.PeerID]peerInfo),
		sortedByScore: make([]peer.PeerID, 0),
	}
}

func (ap *ActivePeers) Add(p peer.Peer) {
	if _, ok := ap.m[p.ID()]; ok {
		return
	}

	ap.m[p.ID()] = newPeerInfo(p)
	ap.sortedByScore = append(ap.sortedByScore, p.ID())
}

func (ap *ActivePeers) UpdateScore(peerID peer.PeerID, score *big.Int) error {
	info, ok := ap.m[peerID]
	if !ok {
		return errors.Errorf("peer '%s' is not active", peerID)
	}

	info.score = score
	ap.m[peerID] = info
	ap.sort()
	return nil
}

func (ap *ActivePeers) Remove(peerID peer.PeerID) {
	if _, ok := ap.Get(peerID); !ok {
		return
	}

	delete(ap.m, peerID)

	i := 0
	for i < len(ap.sortedByScore) {
		if ap.sortedByScore[i] == peerID {
			break
		}
	}

	ap.sortedByScore = append(ap.sortedByScore[:i], ap.sortedByScore[i+1:]...)
}

func (ap *ActivePeers) Get(peerID peer.PeerID) (peerInfo, bool) {
	info, ok := ap.m[peerID]
	return info, ok
}

func (ap *ActivePeers) ForEach(f func(peer.PeerID, peerInfo)) {
	for id, info := range ap.m {
		f(id, info)
	}
}

func (ap *ActivePeers) GetPeerWithMaxScore() (peerInfo, bool) {
	if len(ap.m) == 0 {
		return peerInfo{}, false
	}

	return ap.m[ap.sortedByScore[0]], true
}

func (ap *ActivePeers) Size() int {
	return len(ap.m)
}

func (ap *ActivePeers) sort() {
	sort.SliceStable(
		ap.sortedByScore,
		func(i, j int) bool {
			return ap.m[ap.sortedByScore[i]].score.Cmp(ap.m[ap.sortedByScore[j]].score) == 1
		},
	)
}
