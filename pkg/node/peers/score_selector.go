package peers

import (
	"container/heap"
	"fmt"
	"math/rand/v2"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type group struct {
	score *proto.Score
	peers []peer.ID
	index int
}

type groupsHeap struct {
	groups []*group
}

func newGroupsHeap() *groupsHeap {
	return &groupsHeap{groups: make([]*group, 0)}
}

func (h *groupsHeap) Len() int {
	return len(h.groups)
}

func (h *groupsHeap) Less(i, j int) bool {
	// Pop selects the group with the largest size
	return len(h.groups[i].peers) > len(h.groups[j].peers)
}

func (h *groupsHeap) Swap(i, j int) {
	h.groups[i], h.groups[j] = h.groups[j], h.groups[i]
	h.groups[i].index = i
	h.groups[j].index = j
}

func (h *groupsHeap) Push(x any) {
	g, ok := x.(*group)
	if !ok {
		panic(fmt.Sprintf("groupsHeap: invalid element type: expected (*group), got (%T)", x))
	}
	g.index = len(h.groups)
	h.groups = append(h.groups, g)
}

func (h *groupsHeap) Pop() any {
	n := len(h.groups)
	g := h.groups[n-1]
	h.groups[n-1] = nil
	g.index = -1 // for safety
	h.groups = h.groups[:n-1]
	return g
}

type (
	peerID   string
	scoreKey string
)

type scoreSelector struct {
	scoreKeyToGroup map[scoreKey]*group // key made of score's string points to the group of peers
	peerToScoreKey  map[peerID]scoreKey
	groups          *groupsHeap
}

func newScoreSelector() *scoreSelector {
	return &scoreSelector{
		scoreKeyToGroup: make(map[scoreKey]*group),
		peerToScoreKey:  make(map[peerID]scoreKey),
		groups:          newGroupsHeap(),
	}
}

func (s *scoreSelector) push(p peer.ID, score *proto.Score) {
	sk := scoreKey(score.String())
	pid := peerID(p.String())
	if prevScoreKey, ok := s.peerToScoreKey[pid]; ok {
		// The peer was added before
		if sk == prevScoreKey { // Do nothing, if the score of the peer hasn't changed.
			return
		}
		// Remove the peer from the previous score group
		s.remove(prevScoreKey, p)
		s.append(sk, score, p)
	} else {
		s.append(sk, score, p)
	}
}

func (s *scoreSelector) delete(p peer.ID) {
	pid := peerID(p.String())
	if sk, ok := s.peerToScoreKey[pid]; ok {
		s.remove(sk, p)
		delete(s.peerToScoreKey, pid)
	}
}

func (s *scoreSelector) append(sk scoreKey, score *proto.Score, p peer.ID) {
	g, ok := s.scoreKeyToGroup[sk]
	if ok {
		// New peer comes to the existent group, just update the peers slice of the group
		g.peers = append(g.peers, p)
	} else {
		// New peer with a new score creates the new group
		g = &group{
			score: score,
			peers: []peer.ID{p},
		}
		heap.Push(s.groups, g)
	}
	// Update heap and all the maps
	heap.Fix(s.groups, g.index)
	pid := peerID(p.String())
	s.scoreKeyToGroup[sk] = g
	s.peerToScoreKey[pid] = sk
}

func (s *scoreSelector) remove(sk scoreKey, p peer.ID) {
	g, ok := s.scoreKeyToGroup[sk]
	if !ok {
		panic(fmt.Sprintf("scoreSelector: inconsistent state of score selector: failed to find group by score %s", sk))
	}
	peerToRemoveID := p.String()
	for i := range g.peers {
		if g.peers[i].String() == peerToRemoveID {
			g.peers[i] = g.peers[len(g.peers)-1] // replace i-th element with the last one
			g.peers = g.peers[:len(g.peers)-1]   // cut the duplicate of the last element
			if len(g.peers) == 0 {               // List is empty after removal of the last element, delete key
				delete(s.scoreKeyToGroup, sk)  // Remove group from map
				heap.Remove(s.groups, g.index) // Remove group from heap
				return
			}
			heap.Fix(s.groups, g.index)
			return
		}
	}
}

func (s *scoreSelector) selectBestPeer(currentBest peer.ID) (peer.ID, *proto.Score) {
	if s.groups.Len() == 0 {
		return nil, nil
	}
	e := heap.Pop(s.groups)
	if g, ok := e.(*group); ok { // Take out group the largest group
		if currentBest != nil {
			currentBestToRemoveID := currentBest.String()
			for i := range g.peers {
				if g.peers[i].String() == currentBestToRemoveID { // The peer is in the group, just return the peer and it's score
					heap.Push(s.groups, g) // Put back the group
					return g.peers[i], g.score
				}
			}
		}
		// The peer was not found in the larges group, time to change the peer.
		// Select the random peer from the group and return it along with a new score value.
		i := rand.IntN(len(g.peers)) // #nosec: it's ok to use math/rand/v2 here
		heap.Push(s.groups, g)
		return g.peers[i], g.score
	}
	panic(fmt.Sprintf("scoreSelector: invalid element type of score selector: expeted (*group), got (%T)", e))
}
