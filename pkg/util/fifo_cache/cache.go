package fifo_cache

const idSize = 64

type KeyValue interface {
	Key() []byte
	Value() any
}

type keyValue struct {
	key   []byte
	value any
}

func (a keyValue) Key() []byte {
	return a.key
}

func (a keyValue) Value() any {
	return a.value
}

type FIFOCache struct {
	index     int
	size      int
	insertSeq []*[idSize]byte
	cache     map[[idSize]byte]any
}

func New(size int) *FIFOCache {
	return &FIFOCache{
		size:      size,
		insertSeq: make([]*[idSize]byte, size),
		index:     0,
		cache:     make(map[[idSize]byte]any),
	}
}

func (a *FIFOCache) Add(keyValue KeyValue) {
	if a.exists(keyValue.Key()) {
		return
	}
	b := [idSize]byte{}
	copy(b[:], keyValue.Key())
	a.cache[b] = keyValue.Value()
	a.replace(keyValue)
}

func (a *FIFOCache) Add2(key []byte, value any) {
	a.Add(keyValue{
		key:   key,
		value: value,
	})
}

func (a *FIFOCache) Get(key []byte) (any, bool) {
	b := [idSize]byte{}
	copy(b[:], key)
	value, ok := a.cache[b]
	return value, ok
}

func (a *FIFOCache) replace(keyValue KeyValue) {
	curIdx := a.index % a.size
	curTransaction := a.insertSeq[curIdx]
	if curTransaction != nil {
		delete(a.cache, *curTransaction)
	} else {
		a.insertSeq[curIdx] = &[idSize]byte{}
	}
	copy(a.insertSeq[curIdx][:], keyValue.Key())
	a.index += 1
}

func (a *FIFOCache) Exists(key []byte) bool {
	return a.exists(key)
}

func (a FIFOCache) exists(key []byte) bool {
	b := [idSize]byte{}
	copy(b[:], key)
	_, ok := a.cache[b]
	return ok
}

func (a FIFOCache) Len() int {
	return len(a.cache)
}

func (a FIFOCache) Cap() int {
	return a.size
}
