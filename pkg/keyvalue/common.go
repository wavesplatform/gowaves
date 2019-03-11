package keyvalue

type KeyValue interface {
	Has(key []byte) (bool, error)
	Put(key, val []byte) error
	PutDirectly(key, val []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	ResetBatch()
	Flush() error
	Close() error
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
