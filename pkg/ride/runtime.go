package ride

import (
	"bytes"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const instanceFieldName = "$instance"

type rideType interface {
	instanceOf() string
	eq(other rideType) bool
	get(prop string) (rideType, error)
}

type rideThrow string

func (a rideThrow) instanceOf() string {
	return "Throw"
}

func (a rideThrow) eq(other rideType) bool {
	if o, ok := other.(rideThrow); ok {
		return a == o
	}
	return false
}

func (a rideThrow) get(prop string) (rideType, error) {
	switch prop {
	case "message":
		return rideString(a), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
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

func (b rideBoolean) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", b.instanceOf(), prop)
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

func (l rideInt) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", l.instanceOf(), prop)
}

type rideBigInt struct {
	v *big.Int
}

func (l rideBigInt) instanceOf() string {
	return "BigInt"
}

func (l rideBigInt) eq(other rideType) bool {
	if o, ok := other.(rideBigInt); ok {
		return l.v.Cmp(o.v) == 0
	}
	return false
}

func (l rideBigInt) get(prop string) (rideType, error) {
	//TODO: there is possibility of few properties like 'bytes', 'int' and so on
	return nil, errors.Errorf("type '%s' has no property '%s'", l.instanceOf(), prop)
}

func (l rideBigInt) String() string {
	return l.v.String()
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

func (s rideString) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", s.instanceOf(), prop)
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

func (b rideBytes) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", b.instanceOf(), prop)
}

type rideObject map[string]rideType

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

func (o rideObject) get(prop string) (rideType, error) {
	v, ok := o[prop]
	if !ok {
		return nil, errors.Errorf("type '%s' has no property '%s'", o.instanceOf(), prop)
	}
	return v, nil
}

func (o rideObject) copy() rideObject {
	r := make(rideObject)
	for k, v := range o {
		r[k] = v
	}
	return r
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

func (a rideAddress) get(prop string) (rideType, error) {
	switch prop {
	case "bytes":
		return rideBytes(a[:]), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
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

func (a rideAddressLike) get(prop string) (rideType, error) {
	switch prop {
	case "bytes":
		return rideBytes(a[:]), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
}

type rideRecipient proto.Recipient

func (a rideRecipient) instanceOf() string {
	switch {
	case a.Address != nil:
		return "Address"
	case a.Alias != nil:
		return "Alias"
	default:
		return "Recipient"
	}
}

func (a rideRecipient) eq(other rideType) bool {
	switch o := other.(type) {
	case rideRecipient:
		return a.Address == o.Address && a.Alias == o.Alias
	case rideAddress:
		return a.Address != nil && bytes.Equal(a.Address[:], o[:])
	case rideAlias:
		return a.Alias != nil && a.Alias.Alias == o.Alias
	case rideBytes:
		return a.Address != nil && bytes.Equal(a.Address[:], o[:])
	default:
		return false
	}
}

func (a rideRecipient) get(prop string) (rideType, error) {
	switch prop {
	case "bytes":
		if a.Address != nil {
			return rideBytes(a.Address[:]), nil
		}
		return rideUnit{}, nil
	case "alias":
		if a.Alias != nil {
			return rideAlias(*a.Alias), nil
		}
		return rideUnit{}, nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
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

func (a rideAlias) get(prop string) (rideType, error) {
	switch prop {
	case "alias":
		return rideString(a.Alias), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
}

type rideUnit struct{}

func (a rideUnit) instanceOf() string {
	return "Unit"
}

func (a rideUnit) eq(other rideType) bool {
	return a.instanceOf() == other.instanceOf()
}

func (a rideUnit) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
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

func (a rideNamedType) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
}

type rideList []rideType

func (a rideList) instanceOf() string {
	return "List[Any]"
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

func (a rideList) get(prop string) (rideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
}

type rideFunction func(env Environment, args ...rideType) (rideType, error)

//go:generate moq -out runtime_moq_test.go . Environment:MockRideEnvironment
type Environment interface {
	scheme() byte
	height() rideInt
	transaction() rideObject
	this() rideType
	block() rideObject
	txID() rideType // Invoke transaction ID
	state() types.SmartState
	timestamp() uint64
	setNewDAppAddress(address proto.Address)
	checkMessageLength(int) bool
	takeString(s string, n int) rideString
	invocation() rideObject // Invocation object made of invoke transaction
	setInvocation(inv rideObject)
	libVersion() int
	validateInternalPayments() bool
	internalPaymentsValidationHeight() uint64
}

type rideConstructor func(Environment) rideType
