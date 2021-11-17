package metamask

import (
	"golang.org/x/crypto/sha3"
	"hash"
)

// KeccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type KeccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

// NewKeccakState creates a new KeccakState
func NewKeccakState() KeccakState {
	return sha3.NewLegacyKeccak256().(KeccakState)
}

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	sha := NewKeccakState()
	for _, b := range data {
		_, _ = sha.Write(b)
	}
	h := make([]byte, HashLength)
	_, _ = sha.Read(h)
	return h
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h Hash) {
	sha := NewKeccakState()
	for _, b := range data {
		_, _ = sha.Write(b)
	}
	_, _ = sha.Read(h[:])
	return h
}
