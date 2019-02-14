package keyvalue

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type KeyVal struct {
	db    *leveldb.DB
	batch *leveldb.Batch
}

func NewKeyVal(path string, withBatch bool) (*KeyVal, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	var batch *leveldb.Batch
	if withBatch {
		batch = new(leveldb.Batch)
	}
	return &KeyVal{db: db, batch: batch}, nil
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

func (k *KeyVal) PutDirectly(key, val []byte) error {
	if err := k.db.Put(key, val, nil); err != nil {
		return err
	}
	return nil
}

func (k *KeyVal) Put(key, val []byte) error {
	if k.batch != nil {
		k.batch.Put(key, val)
	} else {
		if err := k.db.Put(key, val, nil); err != nil {
			return err
		}
	}
	return nil
}

func (k *KeyVal) Flush() error {
	if k.batch == nil {
		return errors.New("no batch to flush")
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

func (k *KeyVal) Close() error {
	return k.db.Close()
}
