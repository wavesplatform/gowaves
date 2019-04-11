package proto

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"time"
)

// PutStringWithUInt16Len writes to the buffer `buf` two bytes of the string `s` length followed with the bytes of the string `s`.
func PutStringWithUInt16Len(buf []byte, s string) {
	sl := uint16(len(s))
	binary.BigEndian.PutUint16(buf, sl)
	copy(buf[2:], s)
}

// StringWithUInt16Len reads a string from the buffer `buf`.
func StringWithUInt16Len(buf []byte) (string, error) {
	if l := len(buf); l < 2 {
		return "", fmt.Errorf("not enought data, expected not less then %d, received %d", 2, l)
	}
	s := binary.BigEndian.Uint16(buf[0:2])
	buf = buf[2:]
	if l := len(buf); l < int(s) {
		return "", fmt.Errorf("not enough data to read sting of length %d, recieved only %d bytes", s, l)
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
		return "", fmt.Errorf("not enought data, expected not less then %d, received %d", 4, l)
	}
	s := binary.BigEndian.Uint32(buf[0:4])
	buf = buf[4:]
	if l := len(buf); l < int(s) {
		return "", fmt.Errorf("not enough data to read sting of length %d, recieved only %d bytes", s, l)
	}
	r := string(buf[:s])
	return r, nil
}

//PutBytesWithUInt16Len prepends given buf with 2 bytes of it's length.
func PutBytesWithUInt16Len(buf []byte, data []byte) {
	sl := uint16(len(data))
	binary.BigEndian.PutUint16(buf, sl)
	copy(buf[2:], data)
}

// BytesWithUInt16Len reads from buf an array of bytes of length encoded in first 2 bytes.
func BytesWithUInt16Len(buf []byte) ([]byte, error) {
	if l := len(buf); l < 2 {
		return nil, fmt.Errorf("not enought data, expected not less then %d, received %d", 2, l)
	}
	s := binary.BigEndian.Uint16(buf[0:2])
	buf = buf[2:]
	if l := len(buf); l < int(s) {
		return nil, fmt.Errorf("not enough data to read array of bytes of lenght %d, recieved only %d bytes", s, l)
	}
	r := make([]byte, s)
	copy(r, buf[:s])
	return r, nil
}

//PutBytesWithUInt32Len prepends given buf with 4 bytes of it's length.
func PutBytesWithUInt32Len(buf []byte, data []byte) {
	sl := uint32(len(data))
	binary.BigEndian.PutUint32(buf, sl)
	copy(buf[4:], data)
}

// BytesWithUInt32Len reads from buf an array of bytes of length encoded in first 4 bytes.
func BytesWithUInt32Len(buf []byte) ([]byte, error) {
	if l := len(buf); l < 4 {
		return nil, fmt.Errorf("not enought data, expected not less then %d, received %d", 4, l)
	}
	s := binary.BigEndian.Uint32(buf[0:4])
	buf = buf[4:]
	if l := len(buf); l < int(s) {
		return nil, fmt.Errorf("not enough data to read array of bytes of lenght %d, recieved only %d bytes", s, l)
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
