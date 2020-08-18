package fride

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const instanceFieldName = "$instance"

type Throw struct {
	Message string
}

func (a Throw) Error() string {
	return a.Message
}

type rideType interface {
	instanceOf() string
	eq(other rideType) bool
}

type rideBoolean bool

func (b rideBoolean) instanceOf() string {
	return "Boolean"
}

func (b rideBoolean) eq(other rideType) bool {
	if o, ok := other.(rideBoolean); ok {
		return b == o
	}
	return false
}

type rideInt int64

func (l rideInt) instanceOf() string {
	return "Int"
}

func (l rideInt) eq(other rideType) bool {
	if o, ok := other.(rideInt); ok {
		return l == o
	}
	return false
}

type rideString string

func (s rideString) instanceOf() string {
	return "String"
}

func (s rideString) eq(other rideType) bool {
	if o, ok := other.(rideString); ok {
		return s == o
	}
	return false
}

type rideBytes []byte

func (b rideBytes) instanceOf() string {
	return "ByteVector"
}

func (b rideBytes) eq(other rideType) bool {
	if o, ok := other.(rideBytes); ok {
		return bytes.Equal(b, o)
	}
	return false
}

type rideObject map[rideString]rideType

func (o rideObject) instanceOf() string {
	if s, ok := o[instanceFieldName].(rideString); ok {
		return string(s)
	}
	return ""
}

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

type rideAddress proto.Address

func (a rideAddress) instanceOf() string {
	return "Address"
}

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

type rideUnit struct{}

func (a rideUnit) instanceOf() string {
	return "Unit"
}

func (a rideUnit) eq(other rideType) bool {
	return a.instanceOf() == other.instanceOf()
}

type rideNamedType struct {
	name string
}

func (a rideNamedType) instanceOf() string {
	return a.name
}

func (a rideNamedType) eq(other rideType) bool {
	return a.instanceOf() == other.instanceOf()
}

type rideFunction func(args ...rideType) (rideType, error)

type rideEnvironment interface {
	height() rideInt
	transaction() rideObject
	this() rideObject
	block() rideObject
}

type rideConstructor func(environment rideEnvironment) rideType

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
