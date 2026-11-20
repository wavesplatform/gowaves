package crypto

import (
	"slices"

	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

var (
	ZeroDigest = Digest{
		0x03, 0x17, 0x0a, 0x2e, 0x75, 0x97, 0xb7, 0xb7, 0xe3, 0xd8, 0x4c, 0x05, 0x39, 0x1d, 0x13, 0x9a,
		0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0, 0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14,
	}
)

type subTree struct {
	height int
	digest Digest
}

type MerkleTree struct {
	stack []subTree
}

func NewMerkleTree() (*MerkleTree, error) {
	return &MerkleTree{
		stack: make([]subTree, 0, 16),
	}, nil
}

func (t *MerkleTree) Push(data []byte) {
	leaf := t.leafDigest(data)
	t.stack = append(t.stack, subTree{height: 0, digest: leaf})
	t.joinAllSubTrees()
}

func (t *MerkleTree) Root() Digest {
	switch len(t.stack) {
	case 0:
		return ZeroDigest
	case 1:
		current := t.stack[0]
		if current.height == 0 {
			current = t.joinSubTrees(t.stack[0], subTree{0, ZeroDigest})
		}
		return current.digest
	default:
		current := t.stack[len(t.stack)-1]
		for i := len(t.stack) - 2; i >= 0; i-- {
			h := t.stack[i].height
			for current.height < h {
				current = t.joinSubTrees(current, subTree{height: 0, digest: ZeroDigest})
			}
			current = t.joinSubTrees(t.stack[i], current)
		}
		return current.digest
	}
}

// RebuildRoot is the same as RebuildMerkleRoot.
func (t *MerkleTree) RebuildRoot(leaf Digest, proofsRootToLeafsOrder []Digest, index uint64) (Digest, error) {
	return RebuildMerkleRoot(leaf, proofsRootToLeafsOrder, index)
}

// RebuildMerkleRoot recalculates the Merkle root hash from the leaf hash, the proofs and the leaf index
// without allocations. Bits of the index, starting from the least significant one, select the side of the node
// at each level from leafs to root. The index must address a leaf of the tree defined by the proofs,
// i.e. 0 <= index < 2^len(proofs), otherwise different indices would produce the same root.
func RebuildMerkleRoot(leaf Digest, proofsRootToLeafsOrder []Digest, index uint64) (Digest, error) {
	if index>>len(proofsRootToLeafsOrder) != 0 {
		return Digest{}, errors.Errorf("index %d is out of range for the tree of height %d",
			index, len(proofsRootToLeafsOrder))
	}
	digest := leaf
	for _, p := range slices.Backward(proofsRootToLeafsOrder) { // Iterate from leafs to root, i.e. in reverse order
		if index%2 == 0 { // Left
			digest = nodeDigest(digest, p)
		} else { // Right
			digest = nodeDigest(p, digest)
		}
		index = index / 2
	}
	return digest, nil
}

func (t *MerkleTree) leafDigest(data []byte) Digest {
	return blake2b.Sum256(data)
}

// nodeDigest returns the BLAKE2b-256 hash of the concatenation of the two digests.
func nodeDigest(a, b Digest) Digest {
	var buf [2 * DigestSize]byte
	copy(buf[:DigestSize], a[:])
	copy(buf[DigestSize:], b[:])
	return blake2b.Sum256(buf[:])
}

func (t *MerkleTree) joinSubTrees(a, b subTree) subTree {
	return subTree{
		height: a.height + 1,
		digest: nodeDigest(a.digest, b.digest),
	}
}

func (t *MerkleTree) joinAllSubTrees() {
	for len(t.stack) > 1 && t.stack[len(t.stack)-1].height == t.stack[len(t.stack)-2].height {
		i := len(t.stack) - 1
		j := len(t.stack) - 2
		t.stack = append(t.stack[:j], t.joinSubTrees(t.stack[j], t.stack[i]))
	}
}
