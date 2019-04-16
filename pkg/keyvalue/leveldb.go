package keyvalue

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type KeyVal struct {
	db *leveldb.DB
}

func NewKeyVal(path string) (*KeyVal, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &KeyVal{db: db}, nil
}

func (k *KeyVal) NewBatch() (Batch, error) {
	return new(leveldb.Batch), nil
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
	if err := k.db.Put(key, val, nil); err != nil {
		return err
	}
	return nil
}

func (k *KeyVal) Flush(batch Batch) error {
	b, ok := batch.(*leveldb.Batch)
	if !ok {
		return errors.New("can't convert batch to leveldb.Batch")
	}
	if err := k.db.Write(b, nil); err != nil {
		return err
	}
	batch.Reset()
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
