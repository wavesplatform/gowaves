package metamask

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"golang.org/x/crypto/sha3"
)

const (
	// AddressLength is the expected length of the address in bytes
	AddressLength = 20
)

// Address represents the 20 byte address of an Ethereum account.
type Address [AddressLength]byte

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

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

func FromHex(s string) []byte {
	if has0xPrefix(s) {
		s = s[2:]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

func (a *Address) Decode() string {
	var b []byte
	b = a[:]
	return hex.EncodeToString(b)
}

func (a Address) hex() []byte {
	var buf [len(a)*2 + 2]byte
	copy(buf[:2], "0x")
	hex.Encode(buf[2:], a[:])
	return buf[:]
}

func (a *Address) checksumHex() []byte {
	buf := a.hex()

	// compute checksum
	sha := sha3.NewLegacyKeccak256()
	sha.Write(buf[2:])
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

func (a Address) Hex() string {
	return string(a.checksumHex())
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
