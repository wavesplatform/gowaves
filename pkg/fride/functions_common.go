package fride

import (
	"github.com/pkg/errors"
)

const defaultThrowMessage = "Explicit script termination"

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

func eq(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "eq")
	}
	return rideBoolean(args[0].eq(args[1])), nil
}

func neq(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "neq")
	}
	return rideBoolean(!args[0].eq(args[1])), nil
}

func instanceOf(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "instanceOf")
	}
	t, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("instanceOf: second argument is not a String value but '%s'", args[1].instanceOf())
	}
	return rideBoolean(args[0].instanceOf() == string(t)), nil
}

func extract(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "extract")
	}
	if args[0].instanceOf() == "Unit" {
		return nil, Throw{Message: "extract() called on unit value"}
	}
	return args[0], nil
}

func isDefined(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "isDefined")
	}
	if args[0].instanceOf() == "Unit" {
		return rideBoolean(false), nil
	}
	return rideBoolean(true), nil
}

func throw(_ RideEnvironment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "throw")
	}
	return nil, Throw{Message: string(s)}
}

func throw0(_ RideEnvironment, _ ...rideType) (rideType, error) {
	return nil, Throw{Message: defaultThrowMessage}
}

func value(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "value")
	}
	if args[0].instanceOf() == "Unit" {
		return nil, Throw{Message: defaultThrowMessage}
	}
	return args[0], nil
}

func valueOrErrorMessage(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "valueOrErrorMessage")
	}
	msg, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("valueOrErrorMessage: unexpected argument type '%s'", args[1])
	}
	if args[0].instanceOf() == "Unit" {
		return nil, Throw{Message: string(msg)}
	}
	return args[0], nil
}

func valueOrElse(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "valueOrErrorMessage")
	}
	if args[0].instanceOf() == "Unit" {
		return args[1], nil
	}
	return args[0], nil
}

func stringProperty(obj rideObject, key string) (rideString, error) {
	p, ok := obj[rideString(key)]
	if !ok {
		return "", errors.Errorf("property '%s' not found", key)
	}
	r, ok := p.(rideString)
	if !ok {
		return "", errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	return r, nil
}

func intProperty(obj rideObject, key string) (rideInt, error) {
	p, ok := obj[rideString(key)]
	if !ok {
		return 0, errors.Errorf("property '%s' not found", key)
	}
	r, ok := p.(rideInt)
	if !ok {
		return 0, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	return r, nil
}

func booleanProperty(obj rideObject, key string) (rideBoolean, error) {
	p, ok := obj[rideString(key)]
	if !ok {
		return false, errors.Errorf("property '%s' not found", key)
	}
	r, ok := p.(rideBoolean)
	if !ok {
		return false, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	return r, nil
}

func extractValue(v rideType) (rideType, error) {
	if _, ok := v.(rideUnit); ok {
		return nil, Throw{Message: "failed to extract from Unit value"}
	}
	return v, nil
}
