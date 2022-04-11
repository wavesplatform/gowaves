package state

import (
	"io"
	"sort"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type stateComponent interface {
	less(stateComponent) bool
	writeTo(io.Writer) error
}

type stateComponents []stateComponent

func (s stateComponents) Len() int {
	return len(s)
}

func (s stateComponents) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s stateComponents) Less(i, j int) bool {
	return s[i].less(s[j])
}

type stateForHashes struct {
	pos  map[string]int
	data stateComponents
}

func newStateForHashes() *stateForHashes {
	return &stateForHashes{
		pos:  make(map[string]int),
		data: make(stateComponents, 0),
	}
}

func (s *stateForHashes) set(key string, c stateComponent) {
	pos, ok := s.pos[key]
	if !ok {
		s.pos[key] = len(s.data)
		s.data = append(s.data, c)
	} else {
		s.data[pos] = c
	}
}

func (s *stateForHashes) reset() {
	s.data = make(stateComponents, 0)
	s.pos = make(map[string]int)
}

func (s *stateForHashes) hash() (crypto.Digest, error) {
	sort.Sort(s.data)
	h, err := crypto.NewFastHash()
	if err != nil {
		return crypto.Digest{}, err
	}
	for _, c := range s.data {
		if err := c.writeTo(h); err != nil {
			return crypto.Digest{}, err
		}
	}
	var res crypto.Digest
	h.Sum(res[:0])
	s.reset()
	return res, nil
}

type stateHasher struct {
	curBlockID *proto.BlockID
	storage    *stateForHashes
	hashes     map[proto.BlockID]crypto.Digest
	emptyHash  crypto.Digest
}

func newStateHasher() *stateHasher {
	emptyHash, _ := crypto.FastHash(nil)
	return &stateHasher{
		storage:   newStateForHashes(),
		hashes:    make(map[proto.BlockID]crypto.Digest),
		emptyHash: emptyHash,
	}
}

func (s *stateHasher) stateHashAt(blockID proto.BlockID) crypto.Digest {
	hash, ok := s.hashes[blockID]
	if !ok {
		// If this block does not exist, no changes to state have been made.
		return s.emptyHash
	}
	return hash
}

func (s *stateHasher) calculateHash() error {
	if s.curBlockID == nil {
		return nil
	}
	hash, err := s.storage.hash()
	if err != nil {
		return err
	}
	s.hashes[*s.curBlockID] = hash
	return nil
}

func (s *stateHasher) checkNewBlock(blockID proto.BlockID) error {
	if s.curBlockID == nil {
		// Need to set first block.
		s.curBlockID = &blockID
		return nil
	}
	if *s.curBlockID == blockID {
		// Block has not changed.
		return nil
	}
	// Block has changed, calculate hash.
	if err := s.calculateHash(); err != nil {
		return err
	}
	// Update current block.
	s.curBlockID = &blockID
	return nil
}

func (s *stateHasher) push(key string, c stateComponent, blockID proto.BlockID) error {
	if err := s.checkNewBlock(blockID); err != nil {
		return err
	}
	s.storage.set(key, c)
	return nil
}

func (s *stateHasher) stop() error {
	return s.calculateHash()
}

func (s *stateHasher) reset() {
	s.hashes = make(map[proto.BlockID]crypto.Digest)
	s.storage.reset()
}
