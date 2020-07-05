package state

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type stateHashes struct {
	hs *historyStorage
}

func newStateHashes(hs *historyStorage) *stateHashes {
	return &stateHashes{hs}
}

func (s *stateHashes) saveStateHash(sh *proto.StateHash, height uint64) error {
	key := stateHashKey{height: height}
	return s.hs.addNewEntry(stateHash, key.bytes(), sh.MarshalBinary(), sh.BlockID)
}

func (s *stateHashes) stateHash(height uint64, filter bool) (*proto.StateHash, error) {
	key := stateHashKey{height: height}
	stateHashBytes, err := s.hs.topEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var sh proto.StateHash
	if err := sh.UnmarshalBinary(stateHashBytes); err != nil {
		return nil, err
	}
	return &sh, nil
}
