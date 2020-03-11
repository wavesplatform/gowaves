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

type TransactionProof struct {
	ID      Digest   `json:"id"`
	Index   int      `json:"transactionIndex"`
	Digests []Digest `json:"merkleProof"`
}

type subTree struct {
	height int
	digest Digest
}

type MerkleTree struct {
	h            hash.Hash
	stack        []subTree
	currentIndex uint64
	proofIndex   uint64
	proofBase    []byte
	proofSet     []Digest
	proofTree    bool
}

func NewMerkleTree() (*MerkleTree, error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return nil, errors.Wrap(err, "merkle tree")
	}
	return &MerkleTree{
		h:     h,
		stack: make([]subTree, 0, 32),
	}, nil
}

func (t *MerkleTree) Push(data []byte) {
	if t.currentIndex == t.proofIndex {
		t.proofBase = data
		t.proofSet = append(t.proofSet, t.leafDigest(data))
	}
	t.stack = append(t.stack, subTree{height: 0, digest: t.leafDigest(data)})
	t.joinAllSubTrees()
	t.currentIndex++
}

func (t *MerkleTree) Root() Digest {
	if len(t.stack) == 0 {
		return ZeroDigest
	}
	current := t.stack[len(t.stack)-1]
	for i := len(t.stack) - 2; i >= 0; i-- {
		current = t.joinSubTrees(t.stack[i], current)
	}
	return current.digest
}

func (t *MerkleTree) leafDigest(data []byte) Digest {
	t.h.Reset()
	t.h.Write(data)
	d := Digest{}
	t.h.Sum(d[:0])
	return d
}

func (t *MerkleTree) nodeDigest(a, b Digest) Digest {
	t.h.Reset()
	t.h.Write(a[:])
	t.h.Write(b[:])
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
		if t.stack[i].height == len(t.proofSet)-1 {
			leaves := uint64(1 << uint(t.stack[i].height))
			mid := (t.currentIndex / leaves) * leaves
			if t.proofIndex < mid {
				t.proofSet = append(t.proofSet, t.stack[i].digest)
			} else {
				t.proofSet = append(t.proofSet, t.stack[j].digest)
			}
		}
		t.stack = append(t.stack[:j], t.joinSubTrees(t.stack[j], t.stack[i]))
	}
}
