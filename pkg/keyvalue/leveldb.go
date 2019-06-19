package keyvalue

import (
	"log"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type pair struct {
	key      []byte
	value    []byte
	deletion bool
}

type batch struct {
	filter *bloomFilter
	pairs  []pair
}

func (b *batch) Delete(key []byte) {
	keyCopy := make([]byte, len(key))
	copy(keyCopy[:], key[:])
	b.pairs = append(b.pairs, pair{key: keyCopy, deletion: true})
}

func (b *batch) Put(key, val []byte) {
	valCopy := make([]byte, len(val))
	copy(valCopy[:], val[:])
	keyCopy := make([]byte, len(key))
	copy(keyCopy[:], key[:])
	b.pairs = append(b.pairs, pair{key: keyCopy, value: valCopy, deletion: false})
	b.filter.add(keyCopy)
}

func (b *batch) leveldbBatch() *leveldb.Batch {
	leveldbBatch := new(leveldb.Batch)
	for _, pair := range b.pairs {
		if pair.deletion {
			leveldbBatch.Delete(pair.key)
		} else {
			leveldbBatch.Put(pair.key, pair.value)
		}
	}
	return leveldbBatch
}

func (b *batch) Reset() {
	b.pairs = nil
}

type KeyVal struct {
	db     *leveldb.DB
	filter *bloomFilter
}

func initBloomFilter(kv *KeyVal, params BloomFilterParams) error {
	filter, err := newBloomFilter(params)
	if err != nil {
		return err
	}
	iter, err := kv.NewKeyIterator([]byte{})
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			log.Fatalf("Iterator error: %v", err)
		}
	}()

	for iter.Next() {
		filter.add(iter.Key())
	}
	kv.filter = filter
	return nil
}

func NewKeyVal(path string, params BloomFilterParams) (*KeyVal, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	kv := &KeyVal{db: db}
	if err := initBloomFilter(kv, params); err != nil {
		return nil, err
	}
	return kv, nil
}

func (k *KeyVal) NewBatch() (Batch, error) {
	return &batch{filter: k.filter}, nil
}

func (k *KeyVal) Get(key []byte) ([]byte, error) {
	if k.filter != nil && k.filter.notInTheSet(key) {
		return nil, ErrNotFound
	}
	val, err := k.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return val, ErrNotFound
	}
	return val, err
}

func (k *KeyVal) Has(key []byte) (bool, error) {
	if k.filter != nil && k.filter.notInTheSet(key) {
		return false, nil
	}
	return k.db.Has(key, nil)
}

func (k *KeyVal) Delete(key []byte) error {
	return k.db.Delete(key, nil)
}

func (k *KeyVal) Put(key, val []byte) error {
	k.filter.add(key)
	if err := k.db.Put(key, val, nil); err != nil {
		return err
	}
	return nil
}

func (k *KeyVal) Flush(b1 Batch) error {
	b, ok := b1.(*batch)
	if !ok {
		return errors.New("can't convert batch interface to leveldb's batch")
	}
	if err := k.db.Write(b.leveldbBatch(), nil); err != nil {
		return err
	}
	b.Reset()
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
