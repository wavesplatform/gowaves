package state

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

func defaultTestBloomFilterParams() keyvalue.BloomFilterParams {
	return keyvalue.BloomFilterParams{N: 2e6, FalsePositiveProbability: 0.01}
}

type storageObjects struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage
	stateDB *stateDB
	rb      *recentBlocks
}

func (s *storageObjects) flush(t *testing.T) {
	s.rb.flush()
	err := s.hs.flush(true)
	assert.NoError(t, err, "hs.flush() failed")
	err = s.stateDB.flush()
	assert.NoError(t, err, "stateDB.flush() failed")
	s.stateDB.reset()
}

func (s *storageObjects) addBlock(t *testing.T, blockID crypto.Signature) {
	err := s.rb.addNewBlockID(blockID)
	assert.NoError(t, err, "rb.addNewBlockID() failed")
	err = s.stateDB.addBlock(blockID)
	assert.NoError(t, err, "stateDB.addBlock() failed")
}

func createStorageObjects() (*storageObjects, []string, error) {
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, nil, err
	}
	res := []string{dbDir0}
	db, err := keyvalue.NewKeyVal(dbDir0, defaultTestBloomFilterParams())
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	stateDB, err := newStateDB(db, dbBatch)
	if err != nil {
		return nil, res, err
	}
	rb, err := newRecentBlocks(rollbackMaxBlocks, nil)
	if err != nil {
		return nil, res, err
	}
	hs, err := newHistoryStorage(db, dbBatch, stateDB, rb)
	if err != nil {
		return nil, res, err
	}
	return &storageObjects{db, dbBatch, hs, stateDB, rb}, res, nil
}

func genBlockIds(t *testing.T, amount int) []crypto.Signature {
	ids := make([]crypto.Signature, amount)
	for i := 0; i < amount; i++ {
		id := make([]byte, crypto.SignatureSize)
		_, err := rand.Read(id)
		assert.NoError(t, err, "rand.Read() failed")
		blockID, err := crypto.NewSignatureFromBytes(id)
		assert.NoError(t, err, "NewSignatureFromBytes() failed")
		ids[i] = blockID
	}
	return ids
}
