package common

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

// AddInt makes safe sum for arbitrary integer type.
func AddInt[T constraints.Integer](a, b T) (T, error) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, nil
	}
	return 0, errors.New("add: integer overflow/underflow")
}

// SubInt makes safe sub for arbitrary integer type.
func SubInt[T constraints.Integer](a, b T) (T, error) {
	c := a - b
	if (c < a) == (b > 0) {
		return c, nil
	}
	return 0, errors.New("sub: integer overflow/underflow")
}

// MulInt makes safe mul for arbitrary integer type.
func MulInt[T constraints.Integer](a, b T) (T, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	c := a * b
	if (c < 0) == ((a < 0) != (b < 0)) {
		if c/b == a {
			return c, nil
		}
	}
	return 0, errors.New("mul: integer overflow/underflow")
}

// Dup duplicate (copy) bytes.
func Dup(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func GetStatePath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".gowaves"), nil
}

func ToBase58JSON(b []byte) []byte {
	s := base58.Encode(b)
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes()
}

func ToBase64JSON(b []byte) []byte {
	s := base64.StdEncoding.EncodeToString(b)
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes()
}

func FromBase58JSONUnchecked(value []byte, name string) ([]byte, error) {
	s := string(value)
	if s == "null" {
		return nil, nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s from JSON", name)
	}
	v, err := base58.Decode(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s from Base58 string", name)
	}
	return v, nil
}

func FromBase58JSON(value []byte, size int, name string) ([]byte, error) {
	v, err := FromBase58JSONUnchecked(value, name)
	if err != nil {
		return nil, err
	}
	if l := len(v); l != size {
		return nil, errors.Errorf("incorrect length %d of %s value, expected %d", l, name, size)
	}
	return v[:size], nil
}

func ToHexJSON(b []byte) []byte {
	return fmt.Appendf(nil, "\"0x%x\"", b)
}

func FromHexJSONUnchecked(value []byte, name string) ([]byte, error) {
	s := string(value)
	if s == "null" {
		return nil, nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s from JSON", name)
	}
	v, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s from hex string", name)
	}
	return v, nil
}

func FromHexJSON(value []byte, size int, name string) ([]byte, error) {
	v, err := FromHexJSONUnchecked(value, name)
	if err != nil {
		return nil, err
	}
	if l := len(v); l != size {
		return nil, errors.Errorf("incorrect length %d of %s value, expected %d", l, name, size)
	}
	return v[:size], nil
}

type tm interface {
	Now() time.Time
}

// EnsureTimeout ensures that no way when expected can be higher than current, but if somehow its happened...
func EnsureTimeout(tm tm, expected uint64) {
	for {
		current := uint64(tm.Now().UnixNano() / 1000000)
		if expected > current {
			<-time.After(5 * time.Millisecond)
			continue
		}
		break
	}
}

func UnixMillisToTime(ts int64) time.Time {
	sec := ts / 1_000
	ns := (ts % 1_000) * 1_000_000
	return time.Unix(sec, ns)
}

func UnixMillisFromTime(t time.Time) int64 {
	return t.UnixMilli()
}

// ReplaceInvalidUtf8Chars replaces invalid utf8 characters with '?' to reproduce JVM behaviour.
func ReplaceInvalidUtf8Chars(s string) string {
	var b strings.Builder

	// Ranging over a string in Go produces runes. When the range keyword
	// encounters an invalid UTF-8 encoding, it returns REPLACEMENT CHARACTER.
	for _, v := range s {
		b.WriteRune(v)
	}
	return b.String()
}

const (
	byte0x00 = 0x00
	byte0x01 = 0x01
	byte0x80 = 0x80
	byte0xff = 0xff
)

// Decode2CBigInt decodes two's complement representation of BigInt from bytes slice.
func Decode2CBigInt(bytes []byte) *big.Int {
	r := new(big.Int)
	if len(bytes) > 0 && bytes[0]&byte0x80 == byte0x80 { // Decode a negative number
		notBytes := make([]byte, len(bytes))
		for i := range notBytes {
			notBytes[i] = ^bytes[i]
		}
		bigOne := big.NewInt(byte0x01)
		r.SetBytes(notBytes)
		r.Add(r, bigOne)
		r.Neg(r)
		return r
	}
	r.SetBytes(bytes)
	return r
}

// Encode2CBigInt encodes BigInt into a two's complement representation.
func Encode2CBigInt(n *big.Int) []byte {
	switch sign := n.Sign(); {
	case sign > 0:
		bts := n.Bytes()
		if len(bts) > 0 && bts[0]&byte0x80 != 0 {
			// We'll have to pad this with 0x00 in order to stop it looking like a negative number
			return padBytes(byte0x00, bts)
		}
		return bts
	case sign == 0: // Zero is written as a single 0 zero rather than no bytes
		return []byte{byte0x00}
	case sign < 0:
		// Convert negative number into two's complement form
		// Subtract 1 and invert
		// If the most-significant-bit isn't set then we'll need to pad the beginning
		// with 0xff in order to keep the number negative
		bigOne := big.NewInt(byte0x01)
		nMinus1 := new(big.Int).Neg(n)
		nMinus1.Sub(nMinus1, bigOne)
		bts := nMinus1.Bytes()
		for i := range bts {
			bts[i] ^= byte0xff
		}
		if l := len(bts); l == 0 || bts[0]&byte0x80 == 0 {
			return padBytes(byte0xff, bts)
		}
		return bts
	default:
		panic("unreachable point reached")
	}
}

func padBytes(p byte, bytes []byte) []byte {
	r := make([]byte, len(bytes)+1)
	r[0] = p
	copy(r[1:], bytes)
	return r
}

func SafeIntToUint32(v int) uint32 {
	r, err := safecast.ToUint32(v)
	if err != nil {
		panic(err)
	}
	return r
}
