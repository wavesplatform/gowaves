package state

import (
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type accountsDataStorage struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	stateDB *stateDB
}

func newAccountsDataStorage(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, stateDB *stateDB) (*accountsDataStorage, error) {
	return &accountsDataStorage{db, dbBatch, stateDB}, nil
}

func (s *accountsDataStorage) appendEntry(entry proto.DataEntry) error {
	return nil
}
