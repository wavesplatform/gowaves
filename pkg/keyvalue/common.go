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

type KeyInter interface {
	Key() []byte
}

func SafeKey(k KeyInter) []byte {
	iterK := k.Key()
	key := make([]byte, len(iterK))
	copy(key, iterK)
	return key
}

type ValueInter interface {
	Value() []byte
}

func SafeValue(k ValueInter) []byte {
	iterV := k.Value()
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
