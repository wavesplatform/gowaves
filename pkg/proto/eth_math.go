package proto

import (
	"fmt"
	"math/big"
)

// Common big integers often used
var (
	big0   = big.NewInt(0)
	big1   = big.NewInt(1)
	big2   = big.NewInt(2)
	big256 = big.NewInt(256)
)

// Various big integer limit values.
var (
	tt256   = new(big.Int).Exp(big2, big256, nil)
	tt256m1 = new(big.Int).Sub(tt256, big1)
)

const (
	// number of bits in a big.Word
	bigWordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// number of bytes in a big.Word
	bigWordBytes = bigWordBits / 8
)

// hexOrDecimal256 marshals big.Int as hex or decimal.
type hexOrDecimal256 big.Int

// newHexOrDecimal256 creates a new hexOrDecimal256
func newHexOrDecimal256(x int64) *hexOrDecimal256 {
	return (*hexOrDecimal256)(big.NewInt(x))
}

// parseEthereumBig256 parses s as a 256 bit integer in decimal or hexadecimal syntax.
// Leading zeros are accepted. The empty string parses as zero.
func parseEthereumBig256(s string) (*big.Int, bool) {
	if s == "" {
		return new(big.Int), true
	}
	var bigint *big.Int
	var ok bool
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		bigint, ok = new(big.Int).SetString(s[2:], 16)
	} else {
		bigint, ok = new(big.Int).SetString(s, 10)
	}
	if ok && bigint.BitLen() > 256 {
		bigint, ok = nil, false
	}
	return bigint, ok
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *hexOrDecimal256) UnmarshalText(input []byte) error {
	bigint, ok := parseEthereumBig256(string(input))
	if !ok {
		return fmt.Errorf("invalid hex or decimal integer %q", input)
	}
	*i = hexOrDecimal256(*bigint)
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (i *hexOrDecimal256) MarshalText() ([]byte, error) {
	if i == nil {
		return []byte("0x0"), nil
	}
	return []byte(fmt.Sprintf("%#x", (*big.Int)(i))), nil
}

// paddedEthereumBigIntToBytes encodes a big integer as a big-endian byte slice. The length
// of the slice is at least n bytes.
func paddedEthereumBigIntToBytes(bigint *big.Int, n int) []byte {
	if bigint.BitLen()/8 >= n {
		return bigint.Bytes()
	}
	buf := make([]byte, n)

	// nickeskov: next we encodes the absolute value of bigint as big-endian bytes. Callers must ensure
	// that buf has enough space. If buf is too short the result will be incomplete.
	i := len(buf)
	for _, d := range bigint.Bits() {
		for j := 0; j < bigWordBytes && i > 0; j++ {
			i--
			buf[i] = byte(d)
			d >>= 8
		}
	}
	return buf
}

// ethereumBigIntToEthereumU256 encodes as a 256 bit two's complement number. This operation is destructive.
func ethereumBigIntToEthereumU256(x *big.Int) *big.Int {
	return x.And(x, tt256m1)
}

// ethereumU256ToBytes converts a big Int into a 256bit EVM number.
// This operation is destructive.
func ethereumU256ToBytes(n *big.Int) []byte {
	return paddedEthereumBigIntToBytes(ethereumBigIntToEthereumU256(n), 32)
}
