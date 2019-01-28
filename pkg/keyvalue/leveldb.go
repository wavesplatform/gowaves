package keyvalue

import (
	"github.com/syndtr/goleveldb/leveldb"
)

type KeyVal struct {
	db           *leveldb.DB
	batch        *leveldb.Batch
	maxBatchSize int
}

func NewKeyVal(path string, maxBatchSize int) (*KeyVal, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &KeyVal{db: db, batch: new(leveldb.Batch), maxBatchSize: maxBatchSize}, nil
}

func (k *KeyVal) Get(key []byte) ([]byte, error) {
	return k.db.Get(key, nil)
}

func (k *KeyVal) Has(key []byte) (bool, error) {
	return k.db.Has(key, nil)
}

func (k *KeyVal) Delete(key []byte) error {
	return k.db.Delete(key, nil)
}

func (k *KeyVal) Put(key, val []byte) error {
	k.batch.Put(key, val)
	if k.batch.Len() >= k.maxBatchSize {
		if err := k.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func (k *KeyVal) Flush() error {
	if err := k.db.Write(k.batch, nil); err != nil {
		return err
	}
	k.batch.Reset()
	return nil
}
