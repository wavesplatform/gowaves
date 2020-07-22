package fride

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type rideType interface {
	_rideType()
	eq(other rideType) bool
	ge(other rideType) bool
}

type rideBoolean bool

func (b rideBoolean) _rideType() {}

func (b rideBoolean) eq(other rideType) bool {
	if o, ok := other.(rideBoolean); ok {
		return b == o
	}
	return false
}

func (b rideBoolean) ge(other rideType) bool {
	return false
}

type rideLong int64

func (l rideLong) _rideType() {}

func (l rideLong) eq(other rideType) bool {
	if o, ok := other.(rideLong); ok {
		return l == o
	}
	return false
}

func (l rideLong) ge(other rideType) bool {
	if o, ok := other.(rideLong); ok {
		return l >= o
	}
	return false
}

type rideString string

func (s rideString) _rideType() {}

func (s rideString) eq(other rideType) bool {
	if o, ok := other.(rideString); ok {
		return s == o
	}
	return false
}

func (s rideString) ge(other rideType) bool {
	if o, ok := other.(rideString); ok {
		return s >= o
	}
	return false
}

type rideBytes []byte

func (b rideBytes) _rideType() {}

func (b rideBytes) eq(other rideType) bool {
	if o, ok := other.(rideBytes); ok {
		return bytes.Equal(b, o)
	}
	return false
}

func (b rideBytes) ge(other rideType) bool {
	if o, ok := other.(rideBytes); ok {
		return bytes.Compare(b, o) >= 0
	}
	return false
}

type rideObject map[rideString]rideType

func (o rideObject) _rideType() {}

func (o rideObject) eq(other rideType) bool {
	if oo, ok := other.(rideObject); ok {
		for k, v := range o {
			if ov, ok := oo[k]; ok {
				if !v.eq(ov) {
					return false
				}
			} else {
				return false
			}
		}
		return true
	}
	return false
}

func (o rideObject) ge(other rideType) bool {
	return false
}

type rideCall struct {
	name  string
	count int
}

func (c rideCall) _rideType() {}

func (c rideCall) eq(other rideType) bool {
	return false //Call is incomparable
}

func (c rideCall) ge(other rideType) bool {
	return false //Call is incomparable
}

type rideAddress proto.Address

func (a rideAddress) _rideType() {}

func (a rideAddress) eq(other rideType) bool {
	switch o := other.(type) {
	case rideAddress:
		return bytes.Equal(a[:], o[:])
	case rideBytes:
		return bytes.Equal(a[:], o[:])
	//TODO: add case to compare with rideRecipient
	default:
		return false
	}
}

func (a rideAddress) ge(other rideType) bool {
	return false
}

type rideFunction func(args ...rideType) (rideType, error)

func fetch(from rideType, prop rideType) (rideType, error) {
	obj, ok := from.(rideObject)
	if ok {
		name, ok := prop.(rideString)
		if !ok {
			return nil, errors.Errorf("unable to fetch by property of invalid type '%T'", prop)
		}
		prop, ok := obj[name]
		if ok {
			return prop, nil
		}
	}
	return nil, errors.Errorf("unable to fetch from non object type '%T'", from)
}
