package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

const (
	// Is set to maximum key size used with this storage.
	maxKeySize = assetBalanceKeySize
)

// filterFunc filters database value for given key and returns it.
type filterFunc func(db keyvalue.KeyValue, key []byte) ([]byte, error)

// Workaround for inability to compare slices (and to use them as keys in map).
type storKey struct {
	// Actual key size.
	keySize int
	// Fixed-size array for key.
	key [maxKeySize]byte
}

func newStorKeyFromBytes(key []byte) (storKey, error) {
	var keyArr [maxKeySize]byte
	if len(key) > maxKeySize {
		return storKey{}, errors.Errorf("key size %d is greater than maximum possible %d", len(key), maxKeySize)
	}
	copy(keyArr[:], key)
	return storKey{keySize: len(key), key: keyArr}, nil
}

func (k *storKey) bytes() []byte {
	return k.key[:k.keySize]
}

// localStorage is commonly used (in state) mechanism.
// It caches data from database, applying custom filter, and then allows to modify it.
// Finally, it allows to write all the data to database batch.
// Initial reason for this data structure is inability to read from DB's batches.
type localStorage struct {
	db     keyvalue.KeyValue
	data   map[storKey][]byte
	filter filterFunc
}

func newLocalStorage(db keyvalue.KeyValue, filter filterFunc) (*localStorage, error) {
	return &localStorage{db: db, data: make(map[storKey][]byte), filter: filter}, nil
}

func (ls *localStorage) setRecord(key, record []byte) error {
	internalKey, err := newStorKeyFromBytes(key)
	if err != nil {
		return err
	}
	ls.data[internalKey] = record
	return nil
}

func (ls *localStorage) retrieveRecordFromDb(key []byte) ([]byte, error) {
	has, err := ls.db.Has(key)
	if err != nil {
		return nil, err
	}
	if !has {
		// Special case: the default behaviour is to return new empty value here.
		return nil, nil
	}
	return ls.filter(ls.db, key)
}

func (ls *localStorage) record(key []byte) ([]byte, error) {
	internalKey, err := newStorKeyFromBytes(key)
	if err != nil {
		return nil, err
	}
	if _, ok := ls.data[internalKey]; !ok {
		record, err := ls.retrieveRecordFromDb(key)
		if err != nil {
			return nil, err
		}
		ls.data[internalKey] = record
	}
	return ls.data[internalKey], nil
}

func (ls *localStorage) reset() {
	ls.data = make(map[storKey][]byte)
}

func (ls *localStorage) addToBatch(batch keyvalue.Batch) error {
	for key, record := range ls.data {
		batch.Put(key.bytes(), record)
	}
	return nil
}
