package metamask

import (
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"hash"
)

const (
	// HashLength is the expected length of the hash in bytes
	HashLength = 32
)

type hashImpl interface {
	hash.Hash
	Read([]byte) (int, error)
}

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

func (h *Hash) Bytes() []byte {
	if h == nil {
		return nil
	}
	return h[:]
}

func (h *Hash) unmarshalFromFastRLP(val *fastrlp.Value) error {
	if err := val.GetHash(h[:]); err != nil {
		return errors.Wrap(err, "failed to unmarshal Hash from fastRLP value")
	}
	return nil
}

func (h *Hash) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	return arena.NewBytes(h.Bytes())
}

func unmarshalHashesFromFastRLP(value *fastrlp.Value) ([]Hash, error) {
	elems, err := value.GetElems()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get elements array")
	}
	hashes := make([]Hash, 0, len(elems))
	for _, elem := range elems {
		var h Hash
		if err := h.unmarshalFromFastRLP(elem); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal Hash from fastRLP value")
		}
		hashes = append(hashes, h)
	}
	return hashes, nil
}

func marshalHashesToFastRLP(arena *fastrlp.Arena, hashes []Hash) *fastrlp.Value {
	array := arena.NewArray()
	for _, h := range hashes {
		val := h.marshalToFastRLP(arena)
		array.Set(val)
	}
	return array
}
