package state

import (
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type stateHashes struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch
}

func newStateHashes(db keyvalue.KeyValue, dbBatch keyvalue.Batch) *stateHashes {
	return &stateHashes{db: db, dbBatch: dbBatch}
}

func (s *stateHashes) saveStateHash(stateHash *proto.StateHash, height uint64) error {
	key := stateHashKey{height: height}
	s.dbBatch.Put(key.bytes(), stateHash.MarshalBinary())
	return nil
}

func (s *stateHashes) stateHash(height uint64) (*proto.StateHash, error) {
	key := stateHashKey{height: height}
	stateHashBytes, err := s.db.Get(key.bytes())
	if err != nil {
		return nil, err
	}
	var sh proto.StateHash
	if err := sh.UnmarshalBinary(stateHashBytes); err != nil {
		return nil, err
	}
	return &sh, nil
}

func (s *stateHashes) rollback(newHeight, oldHeight uint64) error {
	for h := oldHeight; h > newHeight; h-- {
		key := stateHashKey{height: h}
		if err := s.db.Delete(key.bytes()); err != nil {
			return err
		}
	}
	return nil
}
