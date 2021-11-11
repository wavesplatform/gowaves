package ride

import (
	"bytes"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const instanceFieldName = "$instance"

type RideType interface {
	instanceOf() string
	eq(other RideType) bool
	get(prop string) (RideType, error)
}

type rideThrow string

func (a rideThrow) instanceOf() string {
	return "Throw"
}

func (a rideThrow) eq(other RideType) bool {
	if o, ok := other.(rideThrow); ok {
		return a == o
	}
	return false
}

func (a rideThrow) get(prop string) (RideType, error) {
	switch prop {
	case "message":
		return RideString(a), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
}

type RideBoolean bool

func (b RideBoolean) instanceOf() string {
	return "Boolean"
}

func (b RideBoolean) eq(other RideType) bool {
	if o, ok := other.(RideBoolean); ok {
		return b == o
	}
	return false
}

func (b RideBoolean) get(prop string) (RideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", b.instanceOf(), prop)
}

type RideInt int64

func (l RideInt) instanceOf() string {
	return "Int"
}

func (l RideInt) eq(other RideType) bool {
	if o, ok := other.(RideInt); ok {
		return l == o
	}
	return false
}

func (l RideInt) get(prop string) (RideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", l.instanceOf(), prop)
}

type RideBigInt struct {
	V *big.Int
}

func (l RideBigInt) instanceOf() string {
	return "BigInt"
}

func (l RideBigInt) eq(other RideType) bool {
	if o, ok := other.(RideBigInt); ok {
		return l.V.Cmp(o.V) == 0
	}
	return false
}

func (l RideBigInt) get(prop string) (RideType, error) {
	//TODO: there is possibility of few properties like 'bytes', 'int' and so on
	return nil, errors.Errorf("type '%s' has no property '%s'", l.instanceOf(), prop)
}

func (l RideBigInt) String() string {
	return l.V.String()
}

type RideString string

func (s RideString) instanceOf() string {
	return "String"
}

func (s RideString) eq(other RideType) bool {
	if o, ok := other.(RideString); ok {
		return s == o
	}
	return false
}

func (s RideString) get(prop string) (RideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", s.instanceOf(), prop)
}

type RideBytes []byte

func NewRideBytes(b []byte) (RideBytes, error) {
	if len(b) > maxBytesLength {
		return nil, errors.Errorf(
			"NewRideBytes: length of bytes (%d) is greater than allowed (%d)",
			len(b), maxBytesLength,
		)
	}
	return RideBytes(b), nil
}

func (b RideBytes) instanceOf() string {
	return "ByteVector"
}

func (b RideBytes) eq(other RideType) bool {
	if o, ok := other.(RideBytes); ok {
		return bytes.Equal(b, o)
	}
	return false
}

func (b RideBytes) get(prop string) (RideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", b.instanceOf(), prop)
}

type rideObject map[string]RideType

func (o rideObject) instanceOf() string {
	if s, ok := o[instanceFieldName].(RideString); ok {
		return string(s)
	}
	return ""
}

func (o rideObject) eq(other RideType) bool {
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

func (o rideObject) get(prop string) (RideType, error) {
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

type rideAddress proto.WavesAddress

func (a rideAddress) instanceOf() string {
	return "Address"
}

func (a rideAddress) eq(other RideType) bool {
	switch o := other.(type) {
	case rideAddress:
		return bytes.Equal(a[:], o[:])
	case RideBytes:
		return bytes.Equal(a[:], o[:])
	case rideRecipient:
		return o.Address != nil && bytes.Equal(a[:], o.Address[:])
	default:
		return false
	}
}

func (a rideAddress) get(prop string) (RideType, error) {
	switch prop {
	case "bytes":
		return RideBytes(a[:]), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
}

type rideAddressLike []byte

func (a rideAddressLike) instanceOf() string {
	return "Address"
}

func (a rideAddressLike) eq(other RideType) bool {
	switch o := other.(type) {
	case rideAddress:
		return bytes.Equal(a[:], o[:])
	case RideBytes:
		return bytes.Equal(a[:], o[:])
	case rideRecipient:
		return o.Address != nil && bytes.Equal(a[:], o.Address[:])
	default:
		return false
	}
}

func (a rideAddressLike) get(prop string) (RideType, error) {
	switch prop {
	case "bytes":
		return RideBytes(a[:]), nil
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

func (a rideRecipient) eq(other RideType) bool {
	switch o := other.(type) {
	case rideRecipient:
		return a.Address == o.Address && a.Alias == o.Alias
	case rideAddress:
		return a.Address != nil && bytes.Equal(a.Address[:], o[:])
	case rideAlias:
		return a.Alias != nil && a.Alias.Alias == o.Alias
	case RideBytes:
		return a.Address != nil && bytes.Equal(a.Address[:], o[:])
	default:
		return false
	}
}

func (a rideRecipient) get(prop string) (RideType, error) {
	switch prop {
	case "bytes":
		if a.Address != nil {
			return RideBytes(a.Address[:]), nil
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

func (a rideAlias) eq(other RideType) bool {
	switch o := other.(type) {
	case rideRecipient:
		return o.Alias != nil && a.Alias == o.Alias.Alias
	case rideAlias:
		return a.Alias == o.Alias
	default:
		return false
	}
}

func (a rideAlias) get(prop string) (RideType, error) {
	switch prop {
	case "alias":
		return RideString(a.Alias), nil
	default:
		return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
	}
}

type rideUnit struct{}

func (a rideUnit) instanceOf() string {
	return "Unit"
}

func (a rideUnit) eq(other RideType) bool {
	return a.instanceOf() == other.instanceOf()
}

func (a rideUnit) get(prop string) (RideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
}

type rideNamedType struct {
	name string
}

func (a rideNamedType) instanceOf() string {
	return a.name
}

func (a rideNamedType) eq(other RideType) bool {
	return a.instanceOf() == other.instanceOf()
}

func (a rideNamedType) get(prop string) (RideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
}

type RideList []RideType

func (a RideList) instanceOf() string {
	return "List[Any]"
}

func (a RideList) eq(other RideType) bool {
	if a.instanceOf() != other.instanceOf() {
		return false
	}
	o, ok := other.(RideList)
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

func (a RideList) get(prop string) (RideType, error) {
	return nil, errors.Errorf("type '%s' has no property '%s'", a.instanceOf(), prop)
}

type rideFunction func(env Environment, args ...RideType) (RideType, error)

//go:generate moq -out runtime_moq_test.go . Environment:MockRideEnvironment
type Environment interface {
	scheme() byte
	height() RideInt
	transaction() rideObject
	this() RideType
	block() rideObject
	txID() RideType // Invoke transaction ID
	state() types.SmartState
	timestamp() uint64
	setNewDAppAddress(address proto.WavesAddress)
	checkMessageLength(int) bool
	takeString(s string, n int) RideString
	invocation() rideObject // Invocation object made of invoke transaction
	setInvocation(inv rideObject)
	libVersion() int
	validateInternalPayments() bool
	internalPaymentsValidationHeight() uint64
	maxDataEntriesSize() int
}

type rideConstructor func(Environment) RideType
