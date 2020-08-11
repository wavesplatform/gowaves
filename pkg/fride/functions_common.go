package fride

import (
	"github.com/pkg/errors"
)

func checkArgs(args []rideType, count int) error {
	if len(args) != count {
		return errors.Errorf("%d is invalid number of arguments, expected %d", len(args), count)
	}
	for n, arg := range args {
		if arg == nil {
			return errors.Errorf("argument %d is empty", n)
		}
	}
	return nil
}

func eq(args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "eq")
	}
	return rideBoolean(args[0].eq(args[1])), nil
}

func neq(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func instanceOf(args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "instanceOf")
	}
	t, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("instanceOf: second argument is not a String value but '%s'", args[1].instanceOf())
	}
	return rideBoolean(args[0].instanceOf() == string(t)), nil
}

func extract(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func isDefined(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func throw0(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func value(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func valueOrErrorMessage(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func valueOrElse(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}
