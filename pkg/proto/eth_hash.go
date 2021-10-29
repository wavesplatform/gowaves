package proto

import (
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const (
	// EthereumHashSize is the expected length of the hash in bytes
	EthereumHashSize = 32
)

// EthereumHash represents the 32 byte Keccak256 hash of arbitrary data.
type EthereumHash [EthereumHashSize]byte

// Keccak256EthereumHash calculates and returns the Keccak256 hash of the input data,
// converting it to an EthereumHash data structure.
func Keccak256EthereumHash(data []byte) EthereumHash {
	return EthereumHash(crypto.MustKeccak256(data))
}

// BytesToEthereumHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToEthereumHash(b []byte) EthereumHash {
	var h EthereumHash
	h.SetBytes(b)
	return h
}

// Bytes converts the fixed-length byte array of the EthereumHash to a slice of bytes.
func (h EthereumHash) Bytes() []byte {
	return h[:]
}

func (h EthereumHash) MarshalJSON() ([]byte, error) {
	return common.ToHexJSON(h[:]), nil
}

func (h *EthereumHash) UnmarshalJSON(bytes []byte) error {
	hashBytes, err := common.FromHexJSON(bytes, EthereumHashSize, "EthereumHash")
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal EthereumHash from JSON")
	}
	*h = BytesToEthereumHash(hashBytes)
	return err
}

// Bytes converts the fixed-length byte array of the EthereumHash to a slice of bytes.
// If *EthereumAddress == nil copy returns nil.
func (h *EthereumHash) tryToBytes() []byte {
	if h == nil {
		return nil
	}
	return h.Bytes()
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h EthereumHash) String() string {
	return h.Hex()
}

// Hex converts a hash to a hex string.
func (h EthereumHash) Hex() string {
	return EncodeToHexString(h[:])
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *EthereumHash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-EthereumHashSize:]
	}

	copy(h[EthereumHashSize-len(b):], b)
}

func (h *EthereumHash) unmarshalFromFastRLP(val *fastrlp.Value) error {
	if err := val.GetHash(h[:]); err != nil {
		return errors.Wrap(err, "failed to unmarshal EthereumHash from fastRLP value")
	}
	return nil
}

func (h *EthereumHash) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	return arena.NewBytes(h.tryToBytes())
}

func unmarshalHashesFromFastRLP(value *fastrlp.Value) ([]EthereumHash, error) {
	elems, err := value.GetElems()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get elements array")
	}
	hashes := make([]EthereumHash, 0, len(elems))
	for _, elem := range elems {
		var h EthereumHash
		if err := h.unmarshalFromFastRLP(elem); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal EthereumHash from fastRLP value")
		}
		hashes = append(hashes, h)
	}
	return hashes, nil
}

func marshalHashesToFastRLP(arena *fastrlp.Arena, hashes []EthereumHash) *fastrlp.Value {
	array := arena.NewArray()
	for _, h := range hashes {
		val := h.marshalToFastRLP(arena)
		array.Set(val)
	}
	return array
}
