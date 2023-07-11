package peers

import (
	"math/big"
	"sort"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
)

type activePeers struct {
	m             map[peer.ID]peerInfo
	sortedByScore []peer.ID
	selector      *scoreSelector
}

func newActivePeers() activePeers {
	return activePeers{
		m:             make(map[peer.ID]peerInfo),
		sortedByScore: make([]peer.ID, 0),
		selector:      newScoreSelector(),
	}
}

func (ap *activePeers) add(p peer.Peer) {
	if _, ok := ap.m[p.ID()]; ok {
		return
	}

	ap.m[p.ID()] = newPeerInfo(p)
	ap.sortedByScore = append(ap.sortedByScore, p.ID())
}

func (ap *activePeers) updateScore(peerID peer.ID, score *big.Int) error {
	info, ok := ap.m[peerID]
	if !ok {
		return errors.Errorf("peer '%s' is not active", peerID)
	}

	info.score = score
	ap.m[peerID] = info
	ap.sort()
	ap.selector.push(peerID, score)
	return nil
}

func (ap *activePeers) remove(peerID peer.ID) {
	if _, ok := ap.get(peerID); !ok {
		return
	}

	delete(ap.m, peerID)

	i := 0
	for ; i < len(ap.sortedByScore); i++ {
		if ap.sortedByScore[i] == peerID {
			break
		}
	}

	ap.sortedByScore = append(ap.sortedByScore[:i], ap.sortedByScore[i+1:]...)

	ap.selector.delete(peerID)
}

func (ap *activePeers) get(peerID peer.ID) (peerInfo, bool) {
	info, ok := ap.m[peerID]
	return info, ok
}

func (ap *activePeers) forEach(f func(peer.ID, peerInfo)) {
	for id, info := range ap.m {
		f(id, info)
	}
}

func (ap *activePeers) getPeerWithMaxScore() (peerInfo, bool) {
	if len(ap.m) == 0 {
		return peerInfo{}, false
	}

	return ap.m[ap.sortedByScore[0]], true
}

func (ap *activePeers) getPeerFromLargestPeerGroup(p peer.Peer) (peerInfo, bool) {
	var pid peer.ID
	if p != nil {
		pid = p.ID()
	}
	id, _ := ap.selector.selectBestPeer(pid)
	if id == nil {
		return peerInfo{}, false
	}
	info, ok := ap.m[id]
	if !ok {
		// TODO: peerInfo for the selected peer is not found, looks like inconsistent state of activePeers, maybe panic?
		return peerInfo{}, false
	}
	return info, id != pid
}

func (ap *activePeers) size() int {
	return len(ap.m)
}

func (ap *activePeers) sort() {
	sort.SliceStable(
		ap.sortedByScore,
		func(i, j int) bool {
			return ap.m[ap.sortedByScore[i]].score.Cmp(ap.m[ap.sortedByScore[j]].score) == 1
		},
	)
}
