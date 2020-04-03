package crypto

import (
	"hash"

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
	h     hash.Hash
	stack []subTree
}

func NewMerkleTree() (*MerkleTree, error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return nil, errors.Wrap(err, "merkle tree")
	}
	return &MerkleTree{
		h:     h,
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

func (t *MerkleTree) RebuildRoot(leaf Digest, proofs []Digest, index uint64) Digest {
	digest := leaf
	for i := len(proofs) - 1; i >= 0; i-- {
		if index%2 == 0 { // Left
			digest = t.nodeDigest(digest, proofs[i])
		} else { // Right
			digest = t.nodeDigest(proofs[i], digest)
		}
		index = index / 2
	}
	return digest
}

func (t *MerkleTree) leafDigest(data []byte) Digest {
	t.h.Reset()
	_, err := t.h.Write(data)
	if err != nil {
		panic(err)
	}
	d := Digest{}
	t.h.Sum(d[:0])
	return d
}

func (t *MerkleTree) nodeDigest(a, b Digest) Digest {
	t.h.Reset()
	_, err := t.h.Write(a[:])
	if err != nil {
		panic(err)
	}
	_, err = t.h.Write(b[:])
	if err != nil {
		panic(err)
	}
	d := Digest{}
	t.h.Sum(d[:0])
	return d
}

func (t *MerkleTree) joinSubTrees(a, b subTree) subTree {
	return subTree{
		height: a.height + 1,
		digest: t.nodeDigest(a.digest, b.digest),
	}
}

func (t *MerkleTree) joinAllSubTrees() {
	for len(t.stack) > 1 && t.stack[len(t.stack)-1].height == t.stack[len(t.stack)-2].height {
		i := len(t.stack) - 1
		j := len(t.stack) - 2
		t.stack = append(t.stack[:j], t.joinSubTrees(t.stack[j], t.stack[i]))
	}
}
