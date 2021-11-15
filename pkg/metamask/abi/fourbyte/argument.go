package fourbyte

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

type Argument struct {
	Name string
	Type Type
}

type Arguments []Argument

func NewArgumentFromRideTypeMeta(name string, rideT meta.Type) (Argument, error) {
	t, err := AbiTypeFromRideTypeMeta(rideT)
	if err != nil {
		return Argument{}, errors.Wrapf(err,
			"failed to build ABI argument with name %q from ride type metadata", name,
		)
	}
	arg := Argument{
		Name: name,
		Type: t,
	}
	return arg, err
}

// UnpackRideValues can be used to unpack ABI-encoded hexdata according to the ABI-specification,
// without supplying a struct to unpack into. Instead, this method returns a list containing the
// values. An atomic argument will be a list with one element.
func (arguments Arguments) UnpackRideValues(data []byte) ([]ride.RideType, []byte, error) {
	retval := make([]ride.RideType, 0, len(arguments))
	virtualArgs := 0
	readArgsTotal := 0
	for index, arg := range arguments {
		marshalledValue, err := toRideType((index+virtualArgs)*32, arg.Type, data)
		if arg.Type.T == TupleTy && !isDynamicType(arg.Type) {
			// If we have a static tuple, like (uint256, bool, uint256), these are
			// coded as just like uint256,bool,uint256
			tupleSize := getTypeSize(arg.Type)/32 - 1
			virtualArgs += tupleSize
			readArgsTotal += tupleSize
		}
		if err != nil {
			return nil, nil, err
		}
		retval = append(retval, marshalledValue)
		readArgsTotal += 1
	}
	return retval, data[readArgsTotal*32:], nil
}

// TODO(nickeskov): add ABI spec marshaling
