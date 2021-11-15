package ride

import "github.com/pkg/errors"

func booleanArg(args []RideType) (RideBoolean, error) {
	if len(args) != 1 {
		return false, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return false, errors.Errorf("argument 1 is empty")
	}
	b, ok := args[0].(RideBoolean)
	if !ok {
		return false, errors.Errorf("argument 1 is not of type 'Boolean' but '%s'", args[0].instanceOf())
	}
	return b, nil
}

func booleanToBytes(_ Environment, args ...RideType) (RideType, error) {
	b, err := booleanArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "booleanToBytes")
	}
	if b {
		return RideBytes([]byte{1}), nil
	} else {
		return RideBytes([]byte{0}), nil
	}
}

func booleanToString(_ Environment, args ...RideType) (RideType, error) {
	b, err := booleanArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "booleanToString")
	}
	if b {
		return RideString("true"), nil
	} else {
		return RideString("false"), nil
	}
}

func unaryNot(_ Environment, args ...RideType) (RideType, error) {
	b, err := booleanArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryNot")
	}
	return !b, nil
}
