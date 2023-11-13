package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type blockSequence struct {
	cap int                            // Stores capacity of maps and slices, used during reset.
	req []proto.BlockID                // Sequence of block IDs requested from peer, used to preserve order.
	ids map[proto.BlockID]struct{}     // Set of requested IDs used for fast check.
	bks map[proto.BlockID]*proto.Block // Map of received blocks.
}

func newBlockSequence(capacity int) *blockSequence {
	s := &blockSequence{cap: capacity}
	s.reset()
	return s
}

func (s *blockSequence) reset() {
	s.req = make([]proto.BlockID, 0, s.cap)
	s.ids = make(map[proto.BlockID]struct{}, s.cap)
	s.bks = make(map[proto.BlockID]*proto.Block, s.cap)
}

// pushID adds ID to the map and sequence of req IDs. Returns false if there is no space to push the new ID or
// the ID is already stored as req.
func (s *blockSequence) pushID(id proto.BlockID) bool {
	if len(s.req) == s.cap {
		return false // No space to add new IDs.
	}
	if _, ok := s.ids[id]; ok { // Given ID already exist
		return false
	}
	s.req = append(s.req, id)
	s.ids[id] = struct{}{}
	return true
}

func (s *blockSequence) requested(id proto.BlockID) bool {
	_, ok := s.ids[id]
	return ok
}

func (s *blockSequence) putBlock(block *proto.Block) bool {
	id := block.BlockID()
	if _, ok := s.ids[id]; ok { // Put block if it was requested earlier.
		s.bks[id] = block
		return true
	}
	return false
}

// blocks returns ordered sequence of stored blocks.
// Result can be a partial sequence of consecutive blocks up to the first gap.
func (s *blockSequence) blocks() []*proto.Block {
	r := make([]*proto.Block, 0, len(s.bks))
	for i := range s.req {
		b, ok := s.bks[s.req[i]]
		if !ok {
			break
		}
		r = append(r, b)
	}
	return r
}

func (s *blockSequence) full() bool {
	return len(s.bks) == len(s.req)
}

// relativeCompliment returns the elements of second sequence what out of intersection with first sequence.
// The last ID of intersection returned as second result.
// If there is no intersection between the sequences two nils is returned.
// Block IDs sequences should be provided in natural order (from old to new blocks).
func relativeCompliment(first, second []proto.BlockID) ([]proto.BlockID, bool) {
	intersects := false
	p := 0
	for i := range first {
		if second[p] == first[i] {
			intersects = true
			p++
			continue
		}
		if intersects {
			break
		}
	}
	return second[p:], intersects
}
