package proto

import (
	"encoding/binary"
	"fmt"
)

func PutStringWithUInt16Len(buf [] byte, s string) {
	sl := uint16(len(s))
	_ = buf[sl+1]
	binary.BigEndian.PutUint16(buf, sl)
	copy(buf[2:], s)
}

func StringWithUInt16Len(buf []byte) (string, error) {
	if l := len(buf); l < 2 {
		return "", fmt.Errorf("not enought data, expected not less then %d, received %d", 2, l)
	}
	s := binary.BigEndian.Uint16(buf[0:2])
	buf = buf[2:2+s]
	if l := len(buf); l != int(s) {
		return "", fmt.Errorf("incorrect sting size %d, expected %d", l, s)
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
	switch buf[0] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("invalid bool value %d", buf[0])
	}
}
