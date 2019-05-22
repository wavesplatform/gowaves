package state

import (
	"io/ioutil"
	"os"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

func defaultTestBloomFilterParams() keyvalue.BloomFilterParams {
	return keyvalue.BloomFilterParams{N: 2e6, FalsePositiveProbability: 0.01}
}

type storageObjects struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	stateDB *stateDB
	rb      *recentBlocks
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
	return &storageObjects{db, dbBatch, stateDB, rb}, res, nil
}
