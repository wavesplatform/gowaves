package keyvalue

import (
	"github.com/pkg/errors"
)

var ErrNotFound = errors.New("not found")

type KeyValue interface {
	NewBatch() (Batch, error)
	Has(key []byte) (bool, error)
	Put(key, val []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Flush(batch Batch) error
	Close() error
}

type Batch interface {
	Delete(key []byte)
	Put(key, val []byte)
	Reset()
}

type Iterator interface {
	Key() []byte
	Value() []byte
	Next() bool
	Error() error
	Release()
}

func SafeKey(iter Iterator) []byte {
	key := make([]byte, len(iter.Key()))
	copy(key[:], iter.Key())
	return key
}

func SafeValue(iter Iterator) []byte {
	value := make([]byte, len(iter.Value()))
	copy(value[:], iter.Value())
	return value
}

type IterableKeyVal interface {
	KeyValue
	NewKeyIterator(prefix []byte) (Iterator, error)
}

type CacheParams struct {
	Size int
}
