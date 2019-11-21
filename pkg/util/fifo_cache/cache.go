package fifo_cache

const idSize = 64

type KeyValue interface {
	Key() []byte
	Value() interface{}
}

type keyValue struct {
	key   []byte
	value interface{}
}

func (a keyValue) Key() []byte {
	return a.key
}

func (a keyValue) Value() interface{} {
	return a.value
}

type FIFOCache struct {
	index     int
	size      int
	insertSeq []*[idSize]byte
	cache     map[[idSize]byte]interface{}
}

func New(size int) *FIFOCache {
	return &FIFOCache{
		size:      size,
		insertSeq: make([]*[idSize]byte, size),
		index:     0,
		cache:     make(map[[idSize]byte]interface{}),
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

func (a *FIFOCache) Add2(key []byte, value interface{}) {
	a.Add(keyValue{
		key:   key,
		value: value,
	})
}

func (a *FIFOCache) Get(key []byte) (value interface{}, ok bool) {
	b := [idSize]byte{}
	copy(b[:], key)
	value, ok = a.cache[b]
	return
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
