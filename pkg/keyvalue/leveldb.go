package keyvalue

import (
	"log"
	"sync"

	"github.com/coocood/freecache"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type pair struct {
	key      []byte
	value    []byte
	deletion bool
}

type batch struct {
	mu     *sync.Mutex
	filter *bloomFilter
	pairs  []pair
}

func (b *batch) Delete(key []byte) {
	b.mu.Lock()
	keyCopy := make([]byte, len(key))
	copy(keyCopy[:], key[:])
	b.pairs = append(b.pairs, pair{key: keyCopy, deletion: true})
	b.mu.Unlock()
}

func (b *batch) Put(key, val []byte) {
	b.mu.Lock()
	valCopy := make([]byte, len(val))
	copy(valCopy[:], val[:])
	keyCopy := make([]byte, len(key))
	copy(keyCopy[:], key[:])
	b.pairs = append(b.pairs, pair{key: keyCopy, value: valCopy, deletion: false})
	b.mu.Unlock()
}

func (b *batch) addToFilter() error {
	b.mu.Lock()
	for _, pair := range b.pairs {
		if !pair.deletion {
			if err := b.filter.add(pair.key); err != nil {
				return err
			}
		}
	}
	b.mu.Unlock()
	return nil
}

func (b *batch) addToCache(cache *freecache.Cache) {
	b.mu.Lock()
	for _, pair := range b.pairs {
		if pair.deletion {
			cache.Del(pair.key)
		} else {
			if err := cache.Set(pair.key, pair.value, 0); err != nil {
				// If we can not set the value for some reason, at least make sure the old one is gone.
				cache.Del(pair.key)
			}
		}
	}
	b.mu.Unlock()
}

func (b *batch) leveldbBatch() *leveldb.Batch {
	b.mu.Lock()
	leveldbBatch := new(leveldb.Batch)
	for _, pair := range b.pairs {
		if pair.deletion {
			leveldbBatch.Delete(pair.key)
		} else {
			leveldbBatch.Put(pair.key, pair.value)
		}
	}
	b.mu.Unlock()
	return leveldbBatch
}

func (b *batch) Reset() {
	b.mu.Lock()
	b.pairs = nil
	b.mu.Unlock()
}

type KeyVal struct {
	db     *leveldb.DB
	filter *bloomFilter
	cache  *freecache.Cache
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
		if err := filter.add(iter.Key()); err != nil {
			return err
		}
	}
	kv.filter = filter
	return nil
}

type KeyValParams struct {
	CacheParams
	BloomFilterParams
	WriteBuffer         int
	CompactionTableSize int
	CompactionTotalSize int
}

func NewKeyVal(path string, params KeyValParams) (*KeyVal, error) {
	dbOptions := &opt.Options{
		WriteBuffer:         params.WriteBuffer,
		CompactionTableSize: params.CompactionTableSize,
		CompactionTotalSize: params.CompactionTotalSize,
	}
	db, err := leveldb.OpenFile(path, dbOptions)
	if err != nil {
		return nil, err
	}
	cache := freecache.NewCache(params.CacheParams.Size)
	kv := &KeyVal{db: db, cache: cache}
	if err := initBloomFilter(kv, params.BloomFilterParams); err != nil {
		return nil, err
	}
	return kv, nil
}

func (k *KeyVal) NewBatch() (Batch, error) {
	return &batch{filter: k.filter, mu: &sync.Mutex{}}, nil
}

func (k *KeyVal) Get(key []byte) ([]byte, error) {
	if val, err := k.cache.Get(key); err == nil {
		return val, nil
	}
	if k.filter != nil {
		notInTheSet, err := k.filter.notInTheSet(key)
		if err != nil {
			return nil, err
		}
		if notInTheSet {
			return nil, ErrNotFound
		}
	}
	val, err := k.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, ErrNotFound
	}
	return val, err
}

func (k *KeyVal) Has(key []byte) (bool, error) {
	if k.filter != nil {
		notInTheSet, err := k.filter.notInTheSet(key)
		if err != nil {
			return false, err
		}
		if notInTheSet {
			return false, nil
		}
	}
	if _, err := k.cache.Get(key); err == nil {
		return true, nil
	}
	return k.db.Has(key, nil)
}

func (k *KeyVal) Delete(key []byte) error {
	k.cache.Del(key)
	return k.db.Delete(key, nil)
}

func (k *KeyVal) Put(key, val []byte) error {
	if err := k.db.Put(key, val, nil); err != nil {
		return err
	}
	if err := k.filter.add(key); err != nil {
		return err
	}
	if err := k.cache.Set(key, val, 0); err != nil {
		// If we can not set the value for some reason, at least make sure the old one is gone.
		k.cache.Del(key)
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
	b.addToCache(k.cache)
	if err := b.addToFilter(); err != nil {
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
	log.Printf("Cache HitRate: %v\n", k.cache.HitRate())
	return k.db.Close()
}
