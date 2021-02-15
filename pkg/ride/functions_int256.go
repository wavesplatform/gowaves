package ride

import (
	"github.com/pkg/errors"
)

func int256Arg(args []rideType) (rideInt256, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument 1 is empty")
	}
	l, ok := args[0].(rideInt256)
	if !ok {
		return nil, errors.Errorf("argument 1 is not of type 'Int' but '%s'", args[0].instanceOf())
	}
	return l, nil
}


func int256ToBytes(_ RideEnvironment, args ...rideType) (rideType, error) {
	i, err := int256Arg(args)
	if err != nil {
		return nil, errors.Wrap(err, "int256ToBytes")
	}
	return rideBytes(i), nil
}

