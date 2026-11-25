package state

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type element struct {
	key        string
	value      ast.Tree
	prev, next *element
	bytes      uint64
}

var defaultValue ast.Tree

// lru is the cache of script ASTs.
//
// Read-path audit (parallel invoke execution): the cache is mutated on reads. A hit in get moves the element to
// the head of the recency list, and the lookups of scriptsStorage (newestScriptByAddr, newestScriptByAsset) store
// a freshly parsed tree with set on a miss. RIDE evaluations running in parallel reach these lookups through
// types.EnrichedSmartState, so get, set, deleteIfExists and clear are serialised by mu.
// The unexported helpers (cut, setNewest, del, makeFreeSpace) must be called with mu held.
//
// Values are stored and returned as ast.Tree values, so every caller gets its own copy of the tree header.
// The nodes are shared between the copies, they are never modified by the evaluation
// (see TestConcurrentTreeEvaluation in pkg/ride).
type lru struct {
	mu sync.Mutex

	maxSize, maxBytes, size, bytesUsed uint64

	m              map[string]*element
	newest, oldest *element
	removed        *element // Created in del(), removed in set().
}

func newLru(maxSize, maxBytes uint64) (*lru, error) {
	if maxSize == 0 || maxBytes == 0 {
		return nil, errors.Errorf("cache size must be > 0")
	}
	return &lru{
		maxSize:  maxSize,
		maxBytes: maxBytes,
		m:        make(map[string]*element),
	}, nil
}

func (l *lru) cut(e *element) {
	prev := e.prev
	next := e.next
	e.prev = nil
	e.next = nil
	if prev != nil && next != nil {
		prev.next = next
		next.prev = prev
	} else if prev != nil {
		// The element is the oldest element.
		prev.next = nil
		l.oldest = prev
	} else if next != nil {
		// The element is the newest element.
		next.prev = nil
		l.newest = next
	} else {
		// The element is the only element.
		l.newest = nil
		l.oldest = nil
	}
}

func (l *lru) setNewest(e *element) {
	if l.newest == nil {
		l.newest = e
		l.oldest = e
	} else {
		e.next = l.newest
		l.newest.prev = e
		l.newest = e
	}
}

func (l *lru) del(e *element) {
	delete(l.m, e.key)
	l.cut(e)
	l.size -= 1
	l.bytesUsed -= e.bytes
	e.value = defaultValue
	l.removed = e
}

func (l *lru) makeFreeSpace(bytes uint64) {
	for l.size+1 > l.maxSize || (l.size > 0 && l.bytesUsed+bytes > l.maxBytes) {
		l.del(l.oldest)
	}
}

func (l *lru) get(key []byte) (value ast.Tree, has bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var e *element
	e, has = l.m[string(key)]
	if !has {
		return
	}
	l.cut(e)
	l.setNewest(e)
	return e.value, true
}

func (l *lru) set(key []byte, value ast.Tree, bytes uint64) (existed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	keyStr := string(key)
	e, has := l.m[keyStr]
	if has {
		l.del(e)
	}
	l.makeFreeSpace(bytes)
	e = l.removed
	if e == nil {
		e = &element{}
	}
	e.key = keyStr
	e.value = value
	e.bytes = bytes
	l.m[keyStr] = e
	l.size += 1
	l.bytesUsed += bytes
	l.setNewest(e)
	l.removed = nil
	return has
}

func (l *lru) deleteIfExists(key []byte) (existed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, has := l.m[string(key)]
	if has {
		l.del(e)
	}
	return has
}

// clear removes all elements from the cache.
func (l *lru) clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.m = make(map[string]*element)
	l.newest, l.oldest, l.removed = nil, nil, nil
	l.size, l.bytesUsed = 0, 0
}
