package fride

import (
	"encoding/binary"
	"strconv"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/fride/math"
)

func intArg(args []rideType) (rideInt, error) {
	if len(args) != 1 {
		return 0, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return 0, errors.Errorf("argument 1 is empty")
	}
	l, ok := args[0].(rideInt)
	if !ok {
		return 0, errors.Errorf("argument 1 is not of type 'Int' but '%s'", args[0].instanceOf())
	}
	return l, nil
}

func intArgs2(args []rideType) (rideInt, rideInt, error) {
	if len(args) != 2 {
		return 0, 0, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return 0, 0, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return 0, 0, errors.Errorf("argument 2 is empty")
	}
	l1, ok := args[0].(rideInt)
	if !ok {
		return 0, 0, errors.Errorf("argument 1 is not of type 'Int' but '%s'", args[0].instanceOf())
	}
	l2, ok := args[1].(rideInt)
	if !ok {
		return 0, 0, errors.Errorf("argument 2 is not of type 'Int' but '%s'", args[1].instanceOf())
	}
	return l1, l2, nil
}

func intArgs(args []rideType, count int) ([]rideInt, error) {
	if len(args) != count {
		return nil, errors.Errorf("%d is invalid number of arguments, expected %d", len(args), count)
	}
	r := make([]rideInt, len(args))
	for n, arg := range args {
		if arg == nil {
			return nil, errors.Errorf("argument %d is empty", n+1)
		}
		l, ok := arg.(rideInt)
		if !ok {
			return nil, errors.Errorf("argument %d is not of type 'Int' but '%s'", n+1, arg.instanceOf())
		}
		r[n] = l
	}
	return r, nil
}

func ge(args ...rideType) (rideType, error) {
	l1, l2, err := intArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "ge")
	}
	return rideBoolean(l1 >= l2), nil
}

func gt(args ...rideType) (rideType, error) {
	l1, l2, err := intArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "gt")
	}
	return rideBoolean(l1 > l2), nil
}

func intToString(args ...rideType) (rideType, error) {
	l, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "intToString")
	}
	return rideString(strconv.Itoa(int(l))), nil
}

func unaryMinus(args ...rideType) (rideType, error) {
	l, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryMinus")
	}
	return -l, nil
}

func sum(args ...rideType) (rideType, error) {
	l1, l2, err := intArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "sum")
	}
	return l1 + l2, nil
}

func sub(args ...rideType) (rideType, error) {
	l1, l2, err := intArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "sub")
	}
	return l1 - l2, nil
}

func mul(args ...rideType) (rideType, error) {
	l1, l2, err := intArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "mul")
	}
	return l1 * l2, nil
}

func div(args ...rideType) (rideType, error) {
	l1, l2, err := intArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "div")
	}
	if l2 == 0 {
		return nil, errors.New("div: division by zero")
	}
	return rideInt(math.FloorDiv(int64(l1), int64(l2))), nil
}

func mod(args ...rideType) (rideType, error) {
	i1, i2, err := intArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "mod")
	}
	if i2 == 0 {
		return nil, errors.New("mod: division by zero")
	}
	return rideInt(math.ModDivision(int64(i1), int64(i2))), nil
}

func fraction(args ...rideType) (rideType, error) {
	values, err := intArgs(args, 3)
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	res, err := math.Fraction(int64(values[0]), int64(values[1]), int64(values[2]))
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	return rideInt(res), nil
}

func intToBytes(args ...rideType) (rideType, error) {
	i, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "intToBytes")
	}
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, uint64(i))
	return rideBytes(out), nil
}

func pow(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func log(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}
