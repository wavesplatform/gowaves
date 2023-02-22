package proto

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var ErrNotFound = errors.New("not found")

// PutStringWithUInt8Len converts the string to slice of bytes. The first byte of resulting slice contains the length of the string.
func PutStringWithUInt8Len(buf []byte, s string) {
	sl := uint8(len(s))
	buf[0] = sl
	copy(buf[1:], s)
}

// StringWithUInt8Len reads a string from given slice of bytes. The first byte of slice should contain the length of the following string.
// Function fails then the length of slice is less than 1 byte or the length of remaining slice is less than the length value from first byte.
func StringWithUInt8Len(buf []byte) (string, error) {
	if l := len(buf); l < 1 {
		return "", fmt.Errorf("not enought data, expected not less then %d, received %d", 1, l)
	}
	s := buf[0]
	buf = buf[1:]
	if l := len(buf); l < int(s) {
		return "", fmt.Errorf("not enough data to read sting of length %d, recieved only %d bytes", s, l)
	}
	r := string(buf[:s])
	return r, nil
}

// PutStringWithUInt16Len writes to the buffer `buf` two bytes of the string `s` length followed with the bytes of the string `s`.
func PutStringWithUInt16Len(buf []byte, s string) {
	sl := uint16(len(s))
	binary.BigEndian.PutUint16(buf, sl)
	copy(buf[2:], s)
}

// StringWithUInt16Len reads a string from the buffer `buf`.
func StringWithUInt16Len(buf []byte) (string, error) {
	if l := len(buf); l < 2 {
		return "", fmt.Errorf("not enough data, expected not less than %d, received %d", 2, l)
	}
	s := binary.BigEndian.Uint16(buf[0:2])
	buf = buf[2:]
	if l := len(buf); l < int(s) {
		return "", fmt.Errorf("not enough data to read string of length %d, received only %d bytes", s, l)
	}
	r := string(buf[:s])
	return r, nil
}

// PutStringWithUInt32Len writes to the buffer `buf` four bytes of the string's `s` length followed with the bytes of string itself.
func PutStringWithUInt32Len(buf []byte, s string) {
	sl := uint32(len(s))
	binary.BigEndian.PutUint32(buf, sl)
	copy(buf[4:], s)
}

// StringWithUInt32Len reads a string from the buffer `buf`.
func StringWithUInt32Len(buf []byte) (string, error) {
	if l := len(buf); l < 4 {
		return "", fmt.Errorf("not enough data, expected not less than %d, received %d", 4, l)
	}
	s := binary.BigEndian.Uint32(buf[0:4])
	buf = buf[4:]
	if l := len(buf); l < int(s) {
		return "", fmt.Errorf("not enough data to read string of length %d, received only %d bytes", s, l)
	}
	r := string(buf[:s])
	return r, nil
}

// PutBytesWithUInt16Len prepends given buf with 2 bytes of its length.
func PutBytesWithUInt16Len(buf []byte, data []byte) error {
	l := len(data)
	if l > math.MaxInt16 {
		return errors.Errorf("invalid data length %d", l)
	}
	if bl, rl := len(buf), l+2; bl < rl {
		return errors.Errorf("invalid buffer length %d, required %d", bl, rl)
	}
	binary.BigEndian.PutUint16(buf, uint16(l))
	copy(buf[2:], data)
	return nil
}

// BytesWithUInt16Len reads from buf an array of bytes of length encoded in first 2 bytes.
func BytesWithUInt16Len(buf []byte) ([]byte, error) {
	if l := len(buf); l < 2 {
		return nil, fmt.Errorf("not enough data, expected not less than %d, received %d", 2, l)
	}
	s := binary.BigEndian.Uint16(buf[0:2])
	buf = buf[2:]
	if l := len(buf); l < int(s) {
		return nil, fmt.Errorf("not enough data to read array of bytes of length %d, received only %d bytes", s, l)
	}
	r := make([]byte, s)
	copy(r, buf[:s])
	return r, nil
}

// PutBytesWithUInt32Len prepends given buf with 4 bytes of its length.
func PutBytesWithUInt32Len(buf []byte, data []byte) error {
	l := len(data)
	if l > math.MaxInt32 {
		return errors.Errorf("invalid data length %d", l)
	}
	if bl, rl := len(buf), l+4; bl < rl {
		return errors.Errorf("invalid buffer length %d, required %d", bl, rl)
	}
	binary.BigEndian.PutUint32(buf, uint32(l))
	copy(buf[4:], data)
	return nil
}

// BytesWithUInt32Len reads from buf an array of bytes of length encoded in first 4 bytes.
func BytesWithUInt32Len(buf []byte) ([]byte, error) {
	if l := len(buf); l < 4 {
		return nil, fmt.Errorf("not enough data, expected not less than %d, received %d", 4, l)
	}
	s := binary.BigEndian.Uint32(buf[0:4])
	buf = buf[4:]
	if l := len(buf); l < int(s) {
		return nil, fmt.Errorf("not enough data to read array of bytes of length %d, received only %d bytes", s, l)
	}
	r := make([]byte, s)
	copy(r, buf[:s])
	return r, nil
}

func PutBool(buf []byte, b bool) {
	if b {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
}

func Bool(buf []byte) (bool, error) {
	if l := len(buf); l < 1 {
		return false, errors.New("failed to unmarshal Bool, empty buffer received")
	}
	switch buf[0] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("invalid bool value %d", buf[0])
	}
}

func NewTimestampFromTime(t time.Time) uint64 {
	return NewTimestampFromUnixNano(t.UnixNano())
}

func NewTimestampFromUnixNano(nano int64) uint64 {
	return uint64(nano / 1000000)
}

func NetworkStrFromScheme(scheme Scheme) string {
	prefix := "waves"
	return prefix + string(scheme)
}

// EncodeToHexString encodes b as a hex string with 0x prefix.
func EncodeToHexString(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

// DecodeFromHexString decodes bytes from a hex string which can start with 0x prefix.
func DecodeFromHexString(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}
