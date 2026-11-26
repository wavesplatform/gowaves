package state

import (
	"crypto/sha256"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

const (
	// scriptTreeCacheMaxEntries limits the number of parsed trees in the content-keyed cache.
	scriptTreeCacheMaxEntries = 10000
	// scriptTreeCacheMaxBytes limits the total size of the script bytes whose trees are cached.
	// The tree takes a few times more memory than the script bytes, so the cache holds a few hundred megabytes
	// in the worst case. The biggest dApp script is 32 KiB, so the cache fits at least 2048 of them.
	scriptTreeCacheMaxBytes = 64 << 20
)

// scriptTreeCache is a bounded LRU cache of parsed script trees keyed by the SHA-256 digest of the script bytes.
// The same bytes always produce the same tree, so the entries never go stale: the cache is not cleared on
// rollbacks or UTX validation resets, unlike the address-keyed cache of the scripts storage.
// Only successfully parsed scripts are cached, errors are returned from the parser every time.
//
// The cache returns a shallow copy of the cached tree, so the tree's fields can be changed by the caller.
// The nodes are shared, like it was with the address-keyed cache. The estimators set the Block field of the
// global declarations and callables, the evaluator never reads it, so the trees are safe to use for evaluation
// while the state is locked for writing. The cache is safe for concurrent use.
type scriptTreeCache struct {
	mu       sync.Mutex
	maxBytes int
	l        *lru // Keyed by the SHA-256 digest of the script bytes, sized by the script length.
}

func newScriptTreeCache(maxEntries, maxBytes int) *scriptTreeCache {
	l, err := newLru(uint64(maxEntries), uint64(maxBytes)) // #nosec G115 -- the limits are positive constants.
	if err != nil {
		panic(err)
	}
	return &scriptTreeCache{maxBytes: maxBytes, l: l}
}

func newDefaultScriptTreeCache() *scriptTreeCache {
	return newScriptTreeCache(scriptTreeCacheMaxEntries, scriptTreeCacheMaxBytes)
}

// parse returns the tree of the script, from the cache if possible. A nil cache parses the script every time.
func (c *scriptTreeCache) parse(script []byte) (*ast.Tree, error) {
	if c == nil {
		return serialization.Parse(script)
	}
	key := sha256.Sum256(script)
	if tree, ok := c.get(key); ok {
		return tree, nil
	}
	tree, err := serialization.Parse(script)
	if err != nil {
		return nil, err
	}
	c.put(key, tree, len(script))
	cp := *tree
	return &cp, nil
}

// get returns a shallow copy of the cached tree and marks the entry as the most recently used.
func (c *scriptTreeCache) get(key [sha256.Size]byte) (*ast.Tree, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tree, ok := c.l.get(key[:])
	if !ok {
		return nil, false
	}
	return &tree, true
}

// put caches the tree of a script of the given size in bytes, evicting the least recently used entries to stay
// within the limits. A script larger than the byte limit is not cached. The cached tree value is never returned
// itself, get and parse return shallow copies of it.
func (c *scriptTreeCache) put(key [sha256.Size]byte, tree *ast.Tree, size int) {
	if size > c.maxBytes {
		return // The script is too big to be cached.
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.l.m[string(key[:])]; ok {
		return // Another goroutine has parsed the same script.
	}
	c.l.set(key[:], *tree, uint64(size)) // #nosec G115 -- the size is not negative.
}

func (c *scriptTreeCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.l.m)
}
