package keyvalue

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type KeyVal struct {
	db           *leveldb.DB
	batch        *leveldb.Batch
	maxBatchSize int
}

// 0 maxBatchSize means disable batch and write directly to the database instead.
func NewKeyVal(path string, maxBatchSize int) (*KeyVal, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	var batch *leveldb.Batch
	if maxBatchSize > 0 {
		batch = new(leveldb.Batch)
	}
	return &KeyVal{db: db, batch: batch, maxBatchSize: maxBatchSize}, nil
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
	if k.batch != nil {
		k.batch.Put(key, val)
		if k.batch.Len() >= k.maxBatchSize {
			if err := k.Flush(); err != nil {
				return err
			}
		}
	} else {
		if err := k.db.Put(key, val, nil); err != nil {
			return err
		}
	}
	return nil
}

func (k *KeyVal) Flush() error {
	if k.batch == nil {
		return errors.New("No batch to flush.")
	}
	if err := k.db.Write(k.batch, nil); err != nil {
		return err
	}
	k.batch.Reset()
	return nil
}

func (k *KeyVal) NewKeyIterator(prefix []byte) (Iterator, error) {
	if prefix != nil {
		return k.db.NewIterator(util.BytesPrefix(prefix), nil), nil
	} else {
		return k.db.NewIterator(nil, nil), nil
	}
}
