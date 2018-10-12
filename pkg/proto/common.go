package proto

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
)

func PutStringWithUInt16Len(buf [] byte, s string) {
	sl := uint16(len(s))
	binary.BigEndian.PutUint16(buf, sl)
	copy(buf[2:], s)
}

func StringWithUInt16Len(buf []byte) (string, error) {
	if l := len(buf); l < 2 {
		return "", fmt.Errorf("not enought data, expected not less then %d, received %d", 2, l)
	}
	s := binary.BigEndian.Uint16(buf[0:2])
	buf = buf[2:]
	if l := len(buf); l < int(s) {
		return "", fmt.Errorf("not enough data to read sting of lenght %d, recieved only %d bytes", s, l)
	}
	r := string(buf[:s])
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
