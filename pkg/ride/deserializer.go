package ride

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Deserializer struct {
	source []byte
}

func NewDeserializer(source []byte) *Deserializer {
	return &Deserializer{source: source}
}

func (a *Deserializer) readn(n int) ([]byte, error) {
	if len(a.source) >= n {
		out := a.source[:n]
		a.source = a.source[n:]
		return out, nil
	}
	return nil, errors.New("insufficient length")
}

func (a *Deserializer) Uint16() (uint16, error) {
	b, err := a.readn(2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

func (a *Deserializer) Uint32() (uint32, error) {
	b, err := a.readn(4)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

func (a *Deserializer) Byte() (byte, error) {
	b, err := a.readn(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (a *Deserializer) Bool() (bool, error) {
	b, err := a.Byte()
	if err != nil {
		return false, err
	}
	switch b {
	case sTrue:
		return true, nil
	case sFalse:
		return false, nil
	default:
		return false, errors.New("unknown byte")
	}
}

func (a *Deserializer) Bytes() ([]byte, error) {
	ln, err := a.Uint32()
	if err != nil {
		return nil, err
	}
	// This hack is required for correct type assertion.
	if ln == 0 {
		return []byte(nil), nil
	}
	return a.readn(int(ln))
}

func (a *Deserializer) Map() (uint16, error) {
	b, err := a.Byte()
	if err != nil {
		return 0, err
	}
	if b != sMap {
		return 0, errors.Errorf("expected `Map` byte, found %d", b)
	}
	size, err := a.Uint16()
	return size, err
}

func (a *Deserializer) RideString() (string, error) {
	b, err := a.Byte()
	if err != nil {
		return "", err
	}
	if b != sString {
		return "", errors.Errorf("expected `String` byte %d, found %d", sString, b)
	}
	return a.String()
}

func (a *Deserializer) String() (string, error) {
	bts, err := a.Bytes()
	if err != nil {
		return "", err
	}
	return string(bts), nil
}

func (a *Deserializer) Int64() (int64, error) {
	bts, err := a.readn(8)
	if err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint64(bts)
	return int64(v), nil
}

func (a *Deserializer) RideValue() (rideType, error) {
	b, err := a.Byte()
	if err != nil {
		return nil, err
	}
	switch b {
	case sNoValue:
		return nil, nil
	case sTrue:
		return rideBoolean(true), nil
	case sFalse:
		return rideBoolean(false), nil
	case sInt:
		v, err := a.Int64()
		return rideInt(v), err
	case sString:
		v, err := a.String()
		return rideString(v), err
	case sBytes:
		v, err := a.Bytes()
		return rideBytes(v), err
	case sAddress:
		v, err := a.Bytes()
		if err != nil {
			return nil, err
		}
		addr, err := proto.NewAddressFromBytes(v)
		if err != nil {
			return nil, err
		}
		return rideAddress(addr), err
	case sNamedType:
		v, err := a.String()
		return rideNamedType{name: v}, err
	case sUnit:
		return newUnit(nil), nil
	case sList:
		size, err := a.Uint16()
		if err != nil {
			return nil, err
		}
		lst := make(rideList, size)
		for i := uint16(0); i < size; i++ {
			v, err := a.RideValue()
			if err != nil {
				return nil, err
			}
			lst[i] = v
		}
		return lst, nil
	case sObject:
		size, err := a.Map()
		if err != nil {
			return nil, err
		}
		obj := make(rideObject, size)
		for i := uint16(0); i < size; i++ {
			key, err := a.String()
			if err != nil {
				return nil, err
			}
			value, err := a.RideValue()
			if err != nil {
				return nil, err
			}
			obj[key] = value
		}
		return obj, nil
	default:
		if b <= 100 {
			return rideInt(b), nil
		}
		return nil, errors.Errorf("unknown type %d", b)
	}
}
