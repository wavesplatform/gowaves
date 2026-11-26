package state

import (
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

const (
	treeCacheTestScriptTrue  = "AwZd0cYf" // V3: true
	treeCacheTestScriptFalse = "AQfeYll6" // V1: false
	// dApp with a callable `call(number: Int)` writing an integer entry.
	treeCacheTestDApp = "AAIEAAAAAAAAAAcIAhIDCgEBAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAEAAAAGbnVtYmVyCQAETAAAAAIJAQAAAA" +
		"xJbnRlZ2VyRW50cnkAAAACAgAAAANpbnQFAAAABm51bWJlcgUAAAADbmlsAAAAAE5VO+E="
)

func decodeTreeCacheTestScript(t testing.TB, s string) proto.Script {
	b, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)
	return b
}

func treeCacheTestScripts(t testing.TB) []proto.Script {
	return []proto.Script{
		decodeTreeCacheTestScript(t, treeCacheTestScriptTrue),
		decodeTreeCacheTestScript(t, treeCacheTestScriptFalse),
		decodeTreeCacheTestScript(t, treeCacheTestDApp),
	}
}

func (c *scriptTreeCache) has(script []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := sha256.Sum256(script)
	_, ok := c.l.m[string(key[:])]
	return ok
}

func TestScriptTreeCacheHitsAndMisses(t *testing.T) {
	c := newDefaultScriptTreeCache()
	for _, script := range treeCacheTestScripts(t) {
		expected, err := serialization.Parse(script)
		require.NoError(t, err)

		first, err := c.parse(script) // Miss.
		require.NoError(t, err)
		assert.Equal(t, expected, first)
		assert.True(t, c.has(script))

		second, err := c.parse(script) // Hit.
		require.NoError(t, err)
		assert.Equal(t, expected, second)
		assert.NotSame(t, first, second, "every call must return its own copy of the tree")

		// Changes of the returned tree's fields don't affect the cached tree.
		second.Verifier = nil
		second.Digest = [32]byte{}
		third, err := c.parse(script)
		require.NoError(t, err)
		assert.Equal(t, expected, third)
	}
	assert.Equal(t, 3, c.len())
}

func TestScriptTreeCacheDoesNotCacheErrors(t *testing.T) {
	c := newDefaultScriptTreeCache()
	script := decodeTreeCacheTestScript(t, treeCacheTestScriptTrue)
	script[len(script)-1]++ // Break the checksum.
	for range 2 {
		tree, err := c.parse(script)
		assert.Error(t, err)
		assert.Nil(t, tree)
	}
	assert.Equal(t, 0, c.len())
	_, err := c.parse(proto.Script{})
	assert.Error(t, err)
	assert.Equal(t, 0, c.len())
}

func TestScriptTreeCacheEvictsByEntries(t *testing.T) {
	scripts := treeCacheTestScripts(t)
	c := newScriptTreeCache(2, scriptTreeCacheMaxBytes)
	_, err := c.parse(scripts[0])
	require.NoError(t, err)
	_, err = c.parse(scripts[1])
	require.NoError(t, err)
	_, err = c.parse(scripts[0]) // The first script becomes the most recently used one.
	require.NoError(t, err)
	_, err = c.parse(scripts[2])
	require.NoError(t, err)
	assert.Equal(t, 2, c.len())
	assert.True(t, c.has(scripts[0]))
	assert.False(t, c.has(scripts[1]))
	assert.True(t, c.has(scripts[2]))
}

func TestScriptTreeCacheEvictsByBytes(t *testing.T) {
	scripts := treeCacheTestScripts(t)
	c := newScriptTreeCache(scriptTreeCacheMaxEntries, len(scripts[0])+len(scripts[1]))
	for _, s := range scripts[:2] {
		_, err := c.parse(s)
		require.NoError(t, err)
	}
	assert.Equal(t, 2, c.len())
	// The dApp is bigger than the limit, it is parsed, but not cached and nothing is evicted.
	require.Greater(t, len(scripts[2]), len(scripts[0])+len(scripts[1]))
	tree, err := c.parse(scripts[2])
	require.NoError(t, err)
	assert.True(t, tree.IsDApp())
	assert.Equal(t, 2, c.len())
	assert.False(t, c.has(scripts[2]))

	// A new script that doesn't fit evicts the oldest entry.
	c = newScriptTreeCache(scriptTreeCacheMaxEntries, len(scripts[0])+len(scripts[2]))
	for _, s := range scripts {
		_, err = c.parse(s)
		require.NoError(t, err)
	}
	assert.False(t, c.has(scripts[0]))
	assert.True(t, c.has(scripts[1]))
	assert.True(t, c.has(scripts[2]))
	assert.Equal(t, uint64(len(scripts[1])+len(scripts[2])), c.l.bytesUsed)
}

func TestScriptTreeCacheNil(t *testing.T) {
	var c *scriptTreeCache
	script := decodeTreeCacheTestScript(t, treeCacheTestDApp)
	expected, err := serialization.Parse(script)
	require.NoError(t, err)
	tree, err := c.parse(script)
	require.NoError(t, err)
	assert.Equal(t, expected, tree)
}

func TestScriptTreeCacheConcurrentAccess(t *testing.T) {
	scripts := treeCacheTestScripts(t)
	expected := make([]*ast.Tree, len(scripts))
	for i, s := range scripts {
		tree, err := serialization.Parse(s)
		require.NoError(t, err)
		expected[i] = tree
	}
	c := newScriptTreeCache(2, scriptTreeCacheMaxBytes) // Small cache to exercise the eviction concurrently.
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 200 {
				k := (g + i) % len(scripts)
				tree, err := c.parse(scripts[k])
				if !assert.NoError(t, err) {
					return
				}
				assert.Equal(t, expected[k].Digest, tree.Digest)
				tree.Verifier = nil // Changing the copy must not race with other goroutines.
			}
		})
	}
	wg.Wait()
	assert.LessOrEqual(t, c.len(), 2)
}

func TestLRUClear(t *testing.T) {
	for _, n := range []int{0, 1, 10, lruReuseLimit, lruReuseLimit + 1} {
		l, err := newLru(maxCacheSize, maxCacheBytes)
		require.NoError(t, err)
		for i := range n {
			l.set([]byte{byte(i), byte(i >> 8), byte(i >> 16)}, ast.Tree{LibVersion: ast.LibV3}, scriptSize)
		}
		require.Equal(t, uint64(n), l.size)
		l.clear()
		assert.Equal(t, uint64(0), l.size)
		assert.Equal(t, uint64(0), l.bytesUsed)
		assert.Empty(t, l.m)
		assert.Nil(t, l.newest)
		assert.Nil(t, l.oldest)
		_, ok := l.get([]byte{0, 0, 0})
		assert.False(t, ok)
		// The cache works after clearing.
		l.set([]byte{1}, ast.Tree{LibVersion: ast.LibV4}, scriptSize)
		v, ok := l.get([]byte{1})
		assert.True(t, ok)
		assert.Equal(t, ast.LibV4, v.LibVersion)
		assert.Equal(t, uint64(1), l.size)
	}
}

func TestScriptsStorageTreesAfterUpdateRollbackAndReset(t *testing.T) {
	stor := createStorageObjects(t, true)
	ss, ok := stor.entities.scriptsStorage.(*scriptsStorage)
	require.True(t, ok)
	trees := stor.entities.scriptTrees
	require.NotNil(t, trees)
	require.Same(t, trees, ss.trees)

	scriptA := decodeTreeCacheTestScript(t, treeCacheTestScriptTrue)
	scriptB := decodeTreeCacheTestScript(t, treeCacheTestDApp)
	treeA, err := serialization.Parse(scriptA)
	require.NoError(t, err)
	treeB, err := serialization.Parse(scriptB)
	require.NoError(t, err)
	addr := testGlobal.senderInfo.addr
	pk := testGlobal.senderInfo.pk

	assertScript := func(expected *ast.Tree) {
		newest, nErr := ss.newestScriptByAddr(addr)
		require.NoError(t, nErr)
		assert.Equal(t, expected, newest)
		stable, sErr := ss.scriptByAddr(addr)
		require.NoError(t, sErr)
		assert.Equal(t, expected, stable)
	}

	stor.addBlock(t, blockID0)
	require.NoError(t, ss.setAccountScript(addr, scriptA, pk, blockID0))
	stor.flush(t)
	assertScript(treeA)
	assert.True(t, trees.has(scriptA))

	// Script update.
	stor.addBlock(t, blockID1)
	require.NoError(t, ss.setAccountScript(addr, scriptB, pk, blockID1))
	stor.flush(t)
	assertScript(treeB)
	assert.True(t, trees.has(scriptB))

	// Rollback returns the previous tree, the content-keyed cache is not cleared.
	stor.fullRollbackBlockClearCache(t, blockID0)
	assertScript(treeA)
	assert.Equal(t, 2, trees.len())

	// A script that was set by a validated, but not applied transaction is dropped by the reset.
	stor.addBlock(t, blockID1)
	require.NoError(t, ss.setAccountScript(addr, scriptB, pk, blockID1))
	newest, err := ss.newestScriptByAddr(addr)
	require.NoError(t, err)
	assert.Equal(t, treeB, newest)
	stor.entities.reset()
	require.NoError(t, ss.clearCache())
	assertScript(treeA)
	assert.Equal(t, 2, trees.len())
}

// TestScriptTreeCacheEstimationOfSharedTree checks that the estimators, which set the Block fields of the shared
// nodes, give the same results on cached trees as on freshly parsed ones, and do not break later cache hits.
func TestScriptTreeCacheEstimationOfSharedTree(t *testing.T) {
	c := newDefaultScriptTreeCache()
	for _, script := range treeCacheTestScripts(t) {
		for v := 1; v <= maxEstimatorVersion; v++ {
			fresh, err := serialization.Parse(script)
			require.NoError(t, err)
			expected, err := ride.EstimateTree(fresh, v)
			require.NoError(t, err)
			for range 3 {
				tree, pErr := c.parse(script)
				require.NoError(t, pErr)
				est, eErr := ride.EstimateTree(tree, v)
				require.NoError(t, eErr)
				assert.Equal(t, expected, est, "estimator V%d", v)
			}
		}
	}
}
