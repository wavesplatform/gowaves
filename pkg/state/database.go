package state

import (
	"encoding/binary"
	"log"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

var void = []byte{}

// stateDB is responsible for all the actions which operate on the whole DB.
// For instance, list of valid blocks and height are DB-wide entities.
type stateDB struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch

	heightChange int
}

func newStateDB(db keyvalue.KeyValue, dbBatch keyvalue.Batch) (*stateDB, error) {
	heightBuf := make([]byte, 8)
	has, err := db.Has([]byte{dbHeightKeyPrefix})
	if err != nil {
		return nil, err
	}
	if !has {
		binary.LittleEndian.PutUint64(heightBuf, 0)
		if err := db.Put([]byte{dbHeightKeyPrefix}, heightBuf); err != nil {
			return nil, err
		}
	}
	has, err = db.Has([]byte{rollbackMinHeightKeyPrefix})
	if err != nil {
		return nil, err
	}
	if !has {
		binary.LittleEndian.PutUint64(heightBuf, 1)
		if err := db.Put([]byte{rollbackMinHeightKeyPrefix}, heightBuf); err != nil {
			return nil, err
		}
	}
	return &stateDB{db: db, dbBatch: dbBatch}, nil
}

// Sync blockReadWriter's storage (files) with the database.
func (s *stateDB) syncRw(rw *blockReadWriter) error {
	dbHeightBytes, err := s.db.Get([]byte{dbHeightKeyPrefix})
	if err != nil {
		return err
	}
	dbHeight := binary.LittleEndian.Uint64(dbHeightBytes)
	rwHeightBytes, err := s.db.Get([]byte{rwHeightKeyPrefix})
	if err != nil {
		return err
	}
	rwHeight := binary.LittleEndian.Uint64(rwHeightBytes)
	log.Printf("Synced to initial height %d.\n", dbHeight)
	if rwHeight < dbHeight {
		// This should never happen, because we update block storage before writing changes into DB.
		panic("Impossible to sync: DB is ahead of block storage; remove data dir and restart the node.")
	}
	if dbHeight == 0 {
		if err := rw.removeEverything(false); err != nil {
			return err
		}
	} else {
		last, err := rw.blockIDByHeight(dbHeight)
		if err != nil {
			return err
		}
		if err := rw.rollback(last, false); err != nil {
			return errors.Errorf("failed to remove blocks from block storage: %v", err)
		}
	}
	return nil
}

func (s *stateDB) addBlock(blockID crypto.Signature) error {
	s.heightChange++
	key := blockIdKey{blockID: blockID}
	s.dbBatch.Put(key.bytes(), void)
	return nil
}

func (s *stateDB) isValidBlock(blockID crypto.Signature) (bool, error) {
	key := blockIdKey{blockID: blockID}
	return s.db.Has(key.bytes())
}

func (s *stateDB) rollbackBlock(blockID crypto.Signature) error {
	// Decrease DB's height (for sync/recovery).
	height, err := s.getHeight()
	if err != nil {
		return err
	}
	if err := s.setHeight(height-1, true); err != nil {
		return err
	}
	key := blockIdKey{blockID: blockID}
	if err := s.db.Delete(key.bytes()); err != nil {
		return err
	}
	return nil
}

func (s *stateDB) setRollbackMinHeight(height uint64) error {
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, height)
	s.dbBatch.Put([]byte{rollbackMinHeightKeyPrefix}, heightBytes)
	return nil
}

func (s *stateDB) getRollbackMinHeight() (uint64, error) {
	heightBytes, err := s.db.Get([]byte{rollbackMinHeightKeyPrefix})
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(heightBytes), nil
}

func (s *stateDB) setHeight(height uint64, directly bool) error {
	dbHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(dbHeightBytes, height)
	if directly {
		if err := s.db.Put([]byte{dbHeightKeyPrefix}, dbHeightBytes); err != nil {
			return err
		}
	} else {
		s.dbBatch.Put([]byte{dbHeightKeyPrefix}, dbHeightBytes)
	}
	return nil
}

func (s *stateDB) getHeight() (uint64, error) {
	dbHeightBytes, err := s.db.Get([]byte{dbHeightKeyPrefix})
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(dbHeightBytes), nil
}

func (s *stateDB) calculateNewRollbackMinHeight(newHeight uint64) (uint64, error) {
	prevRollbackMinHeight, err := s.getRollbackMinHeight()
	if err != nil {
		return 0, err
	}
	if newHeight-prevRollbackMinHeight < rollbackMaxBlocks {
		return prevRollbackMinHeight, nil
	}
	return newHeight - rollbackMaxBlocks, nil
}

func (s *stateDB) flush() error {
	prevHeight, err := s.getHeight()
	if err != nil {
		return err
	}
	newHeight := prevHeight + uint64(s.heightChange)
	if err := s.setHeight(newHeight, false); err != nil {
		return err
	}
	newRollbackMinHeight, err := s.calculateNewRollbackMinHeight(newHeight)
	if err != nil {
		return err
	}
	if err := s.setRollbackMinHeight(newRollbackMinHeight); err != nil {
		return err
	}
	if err := s.db.Flush(s.dbBatch); err != nil {
		return err
	}
	return nil
}

func (s *stateDB) reset() {
	s.heightChange = 0
	s.dbBatch.Reset()
}

func (s *stateDB) close() error {
	return s.db.Close()
}
