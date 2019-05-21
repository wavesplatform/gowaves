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

type IterableKeyVal interface {
	KeyValue
	NewKeyIterator(prefix []byte) (Iterator, error)
}
