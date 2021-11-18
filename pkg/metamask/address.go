package metamask

import (
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
)

const (
	// AddressLength is the expected length of the address in bytes
	AddressLength = 20
)

// Address represents the 20 byte address of an Ethereum account.
type Address [AddressLength]byte

// BytesToAddress returns Address with value b.
// If b is larger than len(h), b will be cropped from the left.
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func (a *Address) Bytes() []byte {
	if a == nil {
		return nil
	}
	return a[:]
}

func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// Hash converts an address to a hash by left-padding it with zeros.
func (a Address) Hash() Hash {
	return BytesToHash(a[:])
}

func (a Address) Hex() string {
	return string(a.checksumHex())
}

// String implements fmt.Stringer.
func (a Address) String() string {
	return a.Hex()
}

func (a *Address) checksumHex() []byte {
	buf := HexEncodeToBytes(a[:])

	// compute checksum
	sha := NewKeccakState()
	_, _ = sha.Write(buf[2:])
	hash := sha.Sum(nil)
	for i := 2; i < len(buf); i++ {
		hashByte := hash[(i-2)/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if buf[i] > '9' && hashByte > 7 {
			buf[i] -= 32
		}
	}
	return buf[:]
}

// copy returns an exact copy of the provided Address.
// If a == nil copy returns nil.
func (a *Address) copy() *Address {
	if a == nil {
		return nil
	}
	cpy := *a
	return &cpy
}

func (a *Address) unmarshalFromFastRLP(val *fastrlp.Value) error {
	if err := val.GetAddr(a[:]); err != nil {
		return errors.Wrap(err, "failed to unmarshal Address from fastRLP value")
	}
	return nil
}

func (a *Address) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	return arena.NewBytes(a.Bytes())
}
