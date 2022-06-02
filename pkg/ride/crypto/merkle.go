package crypto

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

func MerkleRootHash(leaf, proof []byte) ([]byte, error) {
	leafData := make([]byte, len(leaf)+1)
	copy(leafData[1:], leaf)

	h, err := blake2b.New256(nil)
	if err != nil {
		return nil, errors.Wrap(err, "merkle")
	}

	_, err = h.Write(leafData)
	if err != nil {
		return nil, errors.Wrap(err, "merkle")
	}

	hash := h.Sum(nil)

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
		h.Reset()
		_, err = h.Write([]byte{0x01}) // Internal prefix
		if err != nil {
			return nil, errors.Wrap(err, "merkle")
		}
		switch side {
		case 0: // Left side
			_, err = h.Write(hash)
			if err != nil {
				return nil, errors.Wrap(err, "merkle")
			}
			_, err = h.Write(other)
			if err != nil {
				return nil, errors.Wrap(err, "merkle")
			}
		case 1: // Right side
			_, err = h.Write(other)
			if err != nil {
				return nil, errors.Wrap(err, "merkle")
			}
			_, err = h.Write(hash)
			if err != nil {
				return nil, errors.Wrap(err, "merkle")
			}
		default:
			return nil, errors.Errorf("merkle: invalid side value %d", side)
		}
		hash = h.Sum(nil)
	}
	return hash, nil
}
