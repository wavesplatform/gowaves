package keyvalue

import (
	"sync"

	"github.com/coocood/freecache"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"go.uber.org/zap"
)

type pair struct {
	key      []byte
	value    []byte
	deletion bool
}

type batch struct {
	mu    *sync.Mutex
	pairs []pair
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

func (b *batch) addToFilter(filter BloomFilter) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, pair := range b.pairs {
		if !pair.deletion {
			if err := filter.add(pair.key); err != nil {
				return err
			}
		}
	}
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
	filter BloomFilter
	cache  *freecache.Cache
	mu     *sync.RWMutex
}

func initBloomFilter(kv *KeyVal, params BloomFilterParams) error {
	zap.S().Info("Loading stored bloom filter...")
	filter, err := newBloomFilterFromStore(params)
	if err == nil {
		kv.filter = filter
		zap.S().Info("Bloom filter loaded successfully")
		return nil
	}
	zap.S().Info("No stored bloom filter found")
	zap.S().Info("Rebuilding bloom filter from DB can take up a few minutes")
	filter, err = newBloomFilter(params)
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
			zap.S().Fatalf("Iterator error: %v", err)
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
	WriteBuffer            int
	CompactionTableSize    int
	CompactionTotalSize    int
	OpenFilesCacheCapacity int
}

func NewKeyVal(path string, params KeyValParams) (*KeyVal, error) {
	currentFDs, err := fdlimit.CurrentFDs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current file descriptors count")
	}
	openFilesCacheCapacity := params.OpenFilesCacheCapacity

	zap.S().Debugf(
		"leveldb.opt.Options.OpenFilesCacheCapacity has been evaluated to %d. CurrentFDs=%d",
		openFilesCacheCapacity,
		currentFDs,
	)

	dbOptions := &opt.Options{
		WriteBuffer:            params.WriteBuffer,
		CompactionTableSize:    params.CompactionTableSize,
		CompactionTotalSize:    params.CompactionTotalSize,
		OpenFilesCacheCapacity: openFilesCacheCapacity,
	}
	db, err := leveldb.OpenFile(path, dbOptions)
	if err != nil {
		return nil, err
	}
	cache := freecache.NewCache(params.CacheParams.Size)
	kv := &KeyVal{db: db, cache: cache, mu: &sync.RWMutex{}}
	if err := initBloomFilter(kv, params.BloomFilterParams); err != nil {
		return nil, err
	}
	return kv, nil
}

func (k *KeyVal) NewBatch() (Batch, error) {
	return &batch{mu: &sync.Mutex{}}, nil
}

func (k *KeyVal) addToCache(key, val []byte) {
	if err := k.cache.Set(key, val, 0); err != nil {
		// If we can not set the value for some reason, at least make sure the old one is gone.
		k.cache.Del(key)
	}
}

func (k *KeyVal) Get(key []byte) ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if val, err := k.cache.Get(key); err == nil { // If `segment.NotFound` error is returned it ignored here
		return val, nil
	}
	// No entry in cache, looking up in DB
	if k.filter != nil {
		notInTheSet, err := k.filter.notInTheSet(key)
		if err != nil {
			return nil, err // Hashing error here
		}
		if notInTheSet {
			return nil, ErrNotFound
		}
	}
	val, err := k.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, ErrNotFound
	}
	k.addToCache(key, val)
	return val, err
}

func (k *KeyVal) Has(key []byte) (bool, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
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
	k.mu.Lock()
	defer k.mu.Unlock()
	k.cache.Del(key)
	return k.db.Delete(key, nil)
}

func (k *KeyVal) Put(key, val []byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := k.db.Put(key, val, nil); err != nil {
		return err
	}
	if err := k.filter.add(key); err != nil {
		return err
	}
	k.addToCache(key, val)
	return nil
}

func (k *KeyVal) Flush(b1 Batch) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	b, ok := b1.(*batch)
	if !ok {
		return errors.New("can't convert Batch interface to leveldb batch")
	}
	if err := k.db.Write(b.leveldbBatch(), nil); err != nil {
		return err
	}
	b.addToCache(k.cache)
	if err := b.addToFilter(k.filter); err != nil {
		return err
	}
	b.Reset()
	return nil
}

func (k *KeyVal) NewKeyIterator(prefix []byte) (Iterator, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if prefix != nil {
		return k.db.NewIterator(util.BytesPrefix(prefix), nil), nil
	} else {
		return k.db.NewIterator(nil, nil), nil
	}
}

func (k *KeyVal) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	zap.S().Infof("Cache hit rate: %v", k.cache.HitRate())
	err := storeBloomFilter(k.filter)
	if err != nil {
		zap.S().Errorf("Failed to save bloom filter: %v", err)
	} else {
		zap.S().Info("Bloom filter stored successfully")
	}
	return k.db.Close()
}
