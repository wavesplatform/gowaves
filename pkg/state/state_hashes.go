package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type stateHashes struct {
	hs *historyStorage
}

func newStateHashes(hs *historyStorage) *stateHashes {
	return &stateHashes{hs}
}

func (s *stateHashes) saveLegacyStateHash(sh *proto.StateHash, height proto.Height) error {
	key := legacyStateHashKey{height: height}
	return s.hs.addNewEntry(legacyStateHash, key.bytes(), sh.MarshalBinary(), sh.BlockID)
}

func (s *stateHashes) legacyStateHash(height proto.Height) (*proto.StateHash, error) {
	key := legacyStateHashKey{height: height}
	stateHashBytes, err := s.hs.topEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	var sh proto.StateHash
	if err := sh.UnmarshalBinary(stateHashBytes); err != nil {
		return nil, err
	}
	return &sh, nil
}

func (s *stateHashes) saveSnapshotStateHash(sh crypto.Digest, height proto.Height, blockID proto.BlockID) error {
	key := snapshotStateHashKey{height: height}
	return s.hs.addNewEntry(snapshotStateHash, key.bytes(), sh.Bytes(), blockID)
}

func (s *stateHashes) newestSnapshotStateHash(height proto.Height) (crypto.Digest, error) {
	key := snapshotStateHashKey{height: height}
	stateHashBytes, err := s.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return crypto.Digest{}, err
	}
	return crypto.NewDigestFromBytes(stateHashBytes)
}

func (s *stateHashes) snapshotStateHash(height proto.Height) (crypto.Digest, error) {
	key := snapshotStateHashKey{height: height}
	stateHashBytes, err := s.hs.topEntryData(key.bytes())
	if err != nil {
		return crypto.Digest{}, err
	}
	return crypto.NewDigestFromBytes(stateHashBytes)
}
