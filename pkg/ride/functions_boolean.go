package ride

import "github.com/pkg/errors"

func booleanArg(args []rideType) (rideBoolean, error) {
	if len(args) != 1 {
		return false, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return false, errors.Errorf("argument 1 is empty")
	}
	b, ok := args[0].(rideBoolean)
	if !ok {
		return false, errors.Errorf("argument 1 is not of type 'Boolean' but '%s'", args[0].instanceOf())
	}
	return b, nil
}

func booleanToBytes(_ environment, args ...rideType) (rideType, error) {
	b, err := booleanArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "booleanToBytes")
	}
	if b {
		return rideBytes([]byte{1}), nil
	} else {
		return rideBytes([]byte{0}), nil
	}
}

func booleanToString(_ environment, args ...rideType) (rideType, error) {
	b, err := booleanArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "booleanToString")
	}
	if b {
		return rideString("true"), nil
	} else {
		return rideString("false"), nil
	}
}

func unaryNot(_ environment, args ...rideType) (rideType, error) {
	b, err := booleanArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryNot")
	}
	return !b, nil
}
