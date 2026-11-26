package proto

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// liquidMerkleCacheSize is the number of the most recent liquid block states kept by the cache.
// A liquid block grows by one microblock at a time, so only the last few states are ever reused;
// a few extra entries cover forks of the liquid block and concurrent miner/NG usage.
const liquidMerkleCacheSize = 16

// merkleStateEntry is the Merkle tree state of the transactions of a block with the given ID.
type merkleStateEntry struct {
	id     BlockID
	scheme Scheme
	count  int
	tree   *crypto.MerkleTree
}

// merkleStateCache keeps the transactions Merkle tree states of the recently created protobuf blocks,
// keyed by block ID. It lets AppendMicroBlock push only the microblock transactions instead of
// recomputing the root over all the liquid block transactions.
//
// The protobuf block ID is the hash of the header without signature, which includes the transactions root,
// so a block with the same ID and the same number of transactions has the same transactions. The block
// signature does not change the ID, so the state stored at block creation stays valid after signing.
type merkleStateCache struct {
	mu      sync.Mutex
	entries [liquidMerkleCacheSize]merkleStateEntry
	next    int
}

// liquidMerkleStates is shared by all the block creation paths (NG, micro miner), because the liquid block
// object itself is not kept between microblocks: the state reloads the top block from storage.
var liquidMerkleStates = &merkleStateCache{} //nolint:gochecknoglobals // process-wide cache, see above.

// get returns a copy of the stored tree state for the block, or nil if there is no matching state.
func (c *merkleStateCache) get(id BlockID, scheme Scheme, count int) *crypto.MerkleTree {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.entries {
		e := &c.entries[i]
		if e.tree != nil && e.count == count && e.scheme == scheme && e.id == id {
			return e.tree.Clone()
		}
	}
	return nil
}

// put stores the tree state for the block. The cache takes ownership of the tree,
// the caller must not modify it after the call.
func (c *merkleStateCache) put(id BlockID, scheme Scheme, count int, tree *crypto.MerkleTree) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.entries {
		e := &c.entries[i]
		if e.tree != nil && e.count == count && e.scheme == scheme && e.id == id {
			return // already stored
		}
	}
	c.entries[c.next] = merkleStateEntry{id: id, scheme: scheme, count: count, tree: tree}
	c.next = (c.next + 1) % liquidMerkleCacheSize
}
