package fifo_cache

type KeyValue[K comparable, V any] interface {
	Key() K
	Value() V
}

type keyValue[K comparable, V any] struct {
	key   K
	value V
}

func (a keyValue[K, V]) Key() K {
	return a.key
}

func (a keyValue[K, V]) Value() V {
	return a.value
}

type nullableKey[K comparable] struct {
	key   K
	valid bool // valid == true means that key is initialized
}

type FIFOCache[K comparable, V any] struct {
	index     int
	size      int
	insertSeq []nullableKey[K]
	cache     map[K]V
}

func New[K comparable, V any](size int) *FIFOCache[K, V] {
	return &FIFOCache[K, V]{
		size:      size,
		insertSeq: make([]nullableKey[K], size),
		index:     0,
		cache:     make(map[K]V),
	}
}

func (a *FIFOCache[K, V]) Add(keyValue KeyValue[K, V]) {
	if a.exists(keyValue.Key()) {
		return
	}
	key := keyValue.Key()
	a.cache[key] = keyValue.Value()
	a.replace(key)
}

func (a *FIFOCache[K, V]) Add2(key K, value V) {
	a.Add(keyValue[K, V]{
		key:   key,
		value: value,
	})
}

func (a *FIFOCache[K, V]) Get(key K) (V, bool) {
	value, ok := a.cache[key]
	return value, ok
}

func (a *FIFOCache[K, V]) replace(key K) {
	curIdx := a.index % a.size
	curTransaction := a.insertSeq[curIdx]
	if curTransaction.valid {
		delete(a.cache, curTransaction.key)
	}
	a.insertSeq[curIdx] = nullableKey[K]{key: key, valid: true}
	a.index += 1
}

func (a *FIFOCache[K, V]) Exists(key K) bool {
	return a.exists(key)
}

func (a FIFOCache[K, V]) exists(key K) bool {
	_, ok := a.cache[key]
	return ok
}

func (a FIFOCache[K, V]) Len() int {
	return len(a.cache)
}

func (a FIFOCache[K, V]) Cap() int {
	return a.size
}
