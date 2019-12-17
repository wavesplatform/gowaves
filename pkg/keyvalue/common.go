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
	Prev() bool

	First() bool
	Last() bool

	Error() error
	Release()
}

func SafeKey(iter Iterator) []byte {
	iterK := iter.Key()
	key := make([]byte, len(iterK))
	copy(key, iterK)
	return key
}

func SafeValue(iter Iterator) []byte {
	iterV := iter.Value()
	value := make([]byte, len(iterV))
	copy(value, iterV)
	return value
}

type IterableKeyVal interface {
	KeyValue
	NewKeyIterator(prefix []byte) (Iterator, error)
}

type CacheParams struct {
	Size int
}
