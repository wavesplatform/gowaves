package fourbyte

import "github.com/wavesplatform/gowaves/pkg/ride"

type Argument struct {
	Name string
	Type Type
}

type Arguments []Argument

// UnpackValues can be used to unpack ABI-encoded hexdata according to the ABI-specification,
// without supplying a struct to unpack into. Instead, this method returns a list containing the
// values. An atomic argument will be a list with one element.
func (arguments Arguments) UnpackValues(data []byte) ([]interface{}, []byte, error) {
	retval := make([]interface{}, 0, len(arguments))
	virtualArgs := 0
	readTotal := 0
	for index, arg := range arguments {
		marshalledValue, err := toGoType((index+virtualArgs)*32, arg.Type, data)
		if arg.Type.T == TupleTy && !isDynamicType(arg.Type) {
			// If we have a static tuple, like (uint256, bool, uint256), these are
			// coded as just like uint256,bool,uint256
			virtualArgs += getTypeSize(arg.Type)/32 - 1
		}
		if err != nil {
			return nil, nil, err
		}
		retval = append(retval, marshalledValue)
		readTotal = (index + virtualArgs) * 32
	}
	return retval, data[readTotal:], nil
}

// UnpackRideValues can be used to unpack ABI-encoded hexdata according to the ABI-specification,
// without supplying a struct to unpack into. Instead, this method returns a list containing the
// values. An atomic argument will be a list with one element.
func (arguments Arguments) UnpackRideValues(data []byte) ([]ride.RideType, []byte, error) {
	retval := make([]ride.RideType, 0, len(arguments))
	virtualArgs := 0
	readTotal := 0
	for index, arg := range arguments {
		marshalledValue, err := toRideType((index+virtualArgs)*32, arg.Type, data)
		if arg.Type.T == TupleTy && !isDynamicType(arg.Type) {
			// If we have a static tuple, like (uint256, bool, uint256), these are
			// coded as just like uint256,bool,uint256
			virtualArgs += getTypeSize(arg.Type)/32 - 1
		}
		if err != nil {
			return nil, nil, err
		}
		retval = append(retval, marshalledValue)
		readTotal = (index + virtualArgs) * 32
	}
	return retval, data[readTotal:], nil
}

// TODO(nickeskov): add ABI spec marshaling
