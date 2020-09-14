package fride

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
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
	case rideRecipient:
		return o.Address != nil && bytes.Equal(a[:], o.Address[:])
	default:
		return false
	}
}

type rideAddressLike []byte

func (a rideAddressLike) instanceOf() string {
	return "Address"
}

func (a rideAddressLike) eq(other rideType) bool {
	switch o := other.(type) {
	case rideAddress:
		return bytes.Equal(a[:], o[:])
	case rideBytes:
		return bytes.Equal(a[:], o[:])
	case rideRecipient:
		return o.Address != nil && bytes.Equal(a[:], o.Address[:])
	default:
		return false
	}
}

type rideRecipient proto.Recipient

func (a rideRecipient) instanceOf() string {
	return "Recipient"
}

func (a rideRecipient) eq(other rideType) bool {
	switch o := other.(type) {
	case rideRecipient:
		return a.Address == o.Address && a.Alias == o.Alias
	case rideAddress:
		return a.Address != nil && bytes.Equal(a.Address[:], o[:])
	case rideBytes:
		return a.Address != nil && bytes.Equal(a.Address[:], o[:])
	default:
		return false
	}
}

func (a rideRecipient) String() string {
	r := proto.Recipient(a)
	return r.String()
}

type rideAlias proto.Alias

func (a rideAlias) instanceOf() string {
	return "Alias"
}

func (a rideAlias) eq(other rideType) bool {
	switch o := other.(type) {
	case rideRecipient:
		return o.Alias != nil && a.Alias == o.Alias.Alias
	case rideAlias:
		return a.Alias == o.Alias
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

type rideList []rideType

func (a rideList) instanceOf() string {
	return "List"
}

func (a rideList) eq(other rideType) bool {
	if a.instanceOf() != other.instanceOf() {
		return false
	}
	o, ok := other.(rideList)
	if !ok {
		return false
	}
	if len(a) != len(o) {
		return false
	}
	for i, item := range a {
		if !item.eq(o[i]) {
			return false
		}
	}
	return true
}

type rideFunction func(env RideEnvironment, args ...rideType) (rideType, error)

//go:generate moq -out runtime_test.go . RideEnvironment:MockRideEnvironment
type RideEnvironment interface {
	scheme() byte
	height() rideInt
	transaction() rideObject
	this() rideObject
	block() rideObject
	txID() rideType // Invoke transaction ID
	state() types.SmartState
	checkMessageLength(int) bool
}

type rideConstructor func(RideEnvironment) rideType

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
