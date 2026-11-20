package crypto

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

const (
	leafPrefix     = 0x00
	internalPrefix = 0x01
	// maxProofElementSize is the maximum size of the sibling hash in the proof, the size is encoded with one byte.
	maxProofElementSize = 255
)

func MerkleRootHash(leaf, proof []byte) ([]byte, error) {
	// The buffer is big enough for a node (prefix, hash and the longest possible sibling) and for small leafs,
	// so the hashing doesn't allocate in the common case.
	var buf [1 + blake2b.Size256 + maxProofElementSize]byte
	leafData := append(append(buf[:0], leafPrefix), leaf...)
	hash := blake2b.Sum256(leafData)

	for pos := 0; len(proof[pos:]) > 0; {
		if len(proof[pos:]) < 2 {
			return nil, errors.New("merkle: invalid proof length")
		}
		side := proof[pos]
		pos++
		l := int(proof[pos])
		pos++
		if len(proof[pos:]) < l {
			return nil, errors.New("merkle: invalid proof length")
		}
		other := proof[pos : pos+l]
		pos += l
		node := append(buf[:0], internalPrefix)
		switch side {
		case 0: // Left side
			node = append(append(node, hash[:]...), other...)
		case 1: // Right side
			node = append(append(node, other...), hash[:]...)
		default:
			return nil, errors.Errorf("merkle: invalid side value %d", side)
		}
		hash = blake2b.Sum256(node)
	}
	return hash[:], nil
}
