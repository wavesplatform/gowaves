package state

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

type element struct {
	key        proto.Address
	value      ast.Script
	prev, next *element
	bytes      uint64
}

var defaultValue ast.Script

type lru struct {
	maxSize, maxBytes, size, bytesUsed uint64

	m              map[proto.Address]*element
	newest, oldest *element
	removed        *element // Created in del(), removed in set().
}

func newLru(maxSize, maxBytes uint64) (*lru, error) {
	if maxSize == 0 || maxBytes == 0 {
		return nil, fmt.Errorf("cache size must be > 0")
	}
	return &lru{
		maxSize:  maxSize,
		maxBytes: maxBytes,
		m:        make(map[proto.Address]*element),
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

func (l *lru) get(key proto.Address) (value ast.Script, has bool) {
	var e *element
	e, has = l.m[key]
	if !has {
		return
	}
	l.cut(e)
	l.setNewest(e)
	return e.value, true
}

func (l *lru) set(key proto.Address, value ast.Script, bytes uint64) (existed bool) {
	e, has := l.m[key]
	if has {
		l.del(e)
	}
	l.makeFreeSpace(bytes)
	e = l.removed
	if e == nil {
		e = &element{}
	}
	e.key = key
	e.value = value
	e.bytes = bytes
	l.m[key] = e
	l.size += 1
	l.bytesUsed += bytes
	l.setNewest(e)
	l.removed = nil
	return has
}

func (l *lru) deleteIfExists(key proto.Address) (existed bool) {
	e, has := l.m[key]
	if has {
		l.del(e)
	}
	return has
}
