package keyvalue

import (
	"log/slog"
	"sync"

	"github.com/coocood/freecache"
	"github.com/pkg/errors"
	"github.com/wavesplatform/goleveldb/leveldb"
	"github.com/wavesplatform/goleveldb/leveldb/opt"
	"github.com/wavesplatform/goleveldb/leveldb/util"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
)

// batch accumulates writes directly in a leveldb.Batch, which copies keys and values into its own buffer.
type batch struct {
	mu sync.Mutex
	lb leveldb.Batch
}

func (b *batch) Delete(key []byte) {
	b.mu.Lock()
	b.lb.Delete(key)
	b.mu.Unlock()
}

func (b *batch) Put(key, val []byte) {
	b.mu.Lock()
	b.lb.Put(key, val)
	b.mu.Unlock()
}

func (b *batch) Reset() {
	b.mu.Lock()
	b.lb.Reset()
	b.mu.Unlock()
}

// cacheFilterUpdater replays the batch records into the cache and the bloom filter.
// Keys and values passed to Put and Delete point into the batch buffer, so they are only valid during the call;
// both the cache and the filter copy or hash them without retaining the slices.
type cacheFilterUpdater struct {
	cache  *freecache.Cache
	filter BloomFilter
	err    error
}

func (u *cacheFilterUpdater) Put(key, value []byte) {
	if err := u.cache.Set(key, value, 0); err != nil {
		// If we can not set the value for some reason, at least make sure the old one is gone.
		u.cache.Del(key)
	}
	if u.err == nil && u.filter != nil {
		u.err = u.filter.add(key)
	}
}

func (u *cacheFilterUpdater) Delete(key []byte) {
	u.cache.Del(key)
}

type KeyVal struct {
	db     *leveldb.DB
	filter BloomFilter
	cache  *freecache.Cache
	mu     sync.RWMutex
}

func initBloomFilter(kv *KeyVal, params BloomFilterParams) error {
	slog.Info("Loading stored bloom filter...")
	filter, err := newBloomFilterFromStore(params)
	if err == nil {
		kv.filter = filter
		slog.Info("Bloom filter loaded successfully")
		return nil
	}
	slog.Info("No stored bloom filter found")
	slog.Info("Rebuilding bloom filter from DB can take up a few minutes")
	filter, err = newBloomFilter(params)
	if err != nil {
		return errors.Wrap(err, "failed to create new bloom filter")
	}
	iter, err := kv.NewKeyIterator([]byte{})
	if err != nil {
		return errors.Wrap(err, "failed to create new bloom filter iterator")
	}
	defer func() {
		iter.Release()
		if itErr := iter.Error(); itErr != nil {
			slog.Error("Iterator error", logging.Error(itErr))
			panic(itErr)
		}
	}()

	for iter.Next() {
		if kErr := filter.add(iter.Key()); kErr != nil {
			return errors.Wrap(kErr, "failed to add key to bloom filter")
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
	CompressionAlgo        CompressionAlgo
}

func NewKeyVal(path string, params KeyValParams) (*KeyVal, error) {
	currentFDs, err := fdlimit.CurrentFDs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current file descriptors count")
	}
	openFilesCacheCapacity := params.OpenFilesCacheCapacity

	slog.Debug("Parameter leveldb.opt.Options.OpenFilesCacheCapacity has been evaluated",
		"value", openFilesCacheCapacity, "CurrentFDs", currentFDs)

	dbOptions := &opt.Options{
		WriteBuffer:            params.WriteBuffer,
		CompactionTableSize:    params.CompactionTableSize,
		CompactionTotalSize:    params.CompactionTotalSize,
		OpenFilesCacheCapacity: openFilesCacheCapacity,
		Strict:                 opt.DefaultStrict | opt.StrictManifest,
		Compression:            opt.Compression(params.CompressionAlgo),
	}
	db, err := openLevelDB(path, dbOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open key-value db by path '%s'", path)
	}
	cache := freecache.NewCache(params.CacheSize)
	kv := &KeyVal{db: db, cache: cache}
	if bfErr := initBloomFilter(kv, params.BloomFilterParams); bfErr != nil {
		return nil, errors.Wrap(bfErr, "failed to initialize bloom filter")
	}
	return kv, nil
}

func openLevelDB(path string, dbOptions *opt.Options) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(path, dbOptions)
	if err == nil { // If no error, then the database is opened successfully.
		return db, nil
	}
	slog.Warn("Failed to open leveldb DB", slog.String("path", path), logging.Error(err))
	slog.Info("Attempting to recover corrupted leveldb, may take a while...", "path", path)
	db, rErr := leveldb.RecoverFile(path, dbOptions)
	if rErr != nil {
		return nil, errors.Wrap(rErr, "failed to recover leveldb")
	}
	slog.Info("Recovered leveldb db", "path", path)
	return db, nil
}

func (k *KeyVal) NewBatch() (Batch, error) {
	return &batch{}, nil
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
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	k.addToCache(key, val)
	return val, nil
}

// noopValueFn is used to check the presence of a key in the cache without copying its value.
func noopValueFn([]byte) error { return nil }

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
	if err := k.cache.GetFn(key, noopValueFn); err == nil {
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
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := k.db.Write(&b.lb, nil); err != nil {
		return err
	}
	u := cacheFilterUpdater{cache: k.cache, filter: k.filter}
	if err := b.lb.Replay(&u); err != nil {
		return err
	}
	if u.err != nil {
		return u.err
	}
	b.lb.Reset()
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
	slog.Info("Cache hit rate", "rate", k.cache.HitRate())
	err := storeBloomFilter(k.filter)
	if err != nil {
		slog.Error("Failed to save bloom filter", logging.Error(err))
	} else {
		slog.Info("Bloom filter stored successfully")
	}
	return k.db.Close()
}
