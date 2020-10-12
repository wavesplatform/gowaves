package ride

import (
	"encoding/binary"
	"strconv"

	"github.com/ericlagergren/decimal"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/math"
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

func twoIntArgs(args []rideType) (rideInt, rideInt, error) {
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

func ge(_ RideEnvironment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "ge")
	}
	return rideBoolean(l1 >= l2), nil
}

func gt(_ RideEnvironment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "gt")
	}
	return rideBoolean(l1 > l2), nil
}

func intToString(_ RideEnvironment, args ...rideType) (rideType, error) {
	l, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "intToString")
	}
	return rideString(strconv.Itoa(int(l))), nil
}

func unaryMinus(_ RideEnvironment, args ...rideType) (rideType, error) {
	l, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryMinus")
	}
	return -l, nil
}

func sum(_ RideEnvironment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sum")
	}
	return l1 + l2, nil
}

func sub(_ RideEnvironment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sub")
	}
	return l1 - l2, nil
}

func mul(_ RideEnvironment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "mul")
	}
	return l1 * l2, nil
}

func div(_ RideEnvironment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "div")
	}
	if l2 == 0 {
		return nil, errors.New("div: division by zero")
	}
	return rideInt(math.FloorDiv(int64(l1), int64(l2))), nil
}

func mod(_ RideEnvironment, args ...rideType) (rideType, error) {
	i1, i2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "mod")
	}
	if i2 == 0 {
		return nil, errors.New("mod: division by zero")
	}
	return rideInt(math.ModDivision(int64(i1), int64(i2))), nil
}

func fraction(_ RideEnvironment, args ...rideType) (rideType, error) {
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

func intToBytes(_ RideEnvironment, args ...rideType) (rideType, error) {
	i, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "intToBytes")
	}
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, uint64(i))
	return rideBytes(out), nil
}

func pow(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 6); err != nil {
		return nil, errors.Wrap(err, "pow")
	}
	base, ok := args[0].(rideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[0].instanceOf())
	}
	bp, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[1].instanceOf())
	}
	exponent, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[2].instanceOf())
	}
	ep, ok := args[3].(rideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[3].instanceOf())
	}
	rp, ok := args[4].(rideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[4].instanceOf())
	}
	round, err := roundingMode(args[5])
	if err != nil {
		return nil, errors.Wrap(err, "pow")
	}
	r, err := math.Pow(int64(base), int64(exponent), int(bp), int(ep), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "pow")
	}
	return rideInt(r), nil
}

func log(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 6); err != nil {
		return nil, errors.Wrap(err, "log")
	}
	base, ok := args[0].(rideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[0].instanceOf())
	}
	bp, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[1].instanceOf())
	}
	exponent, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[2].instanceOf())
	}
	ep, ok := args[3].(rideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[3].instanceOf())
	}
	rp, ok := args[4].(rideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[4].instanceOf())
	}
	round, err := roundingMode(args[5])
	if err != nil {
		return nil, errors.Wrap(err, "log")
	}
	r, err := math.Log(int64(base), int64(exponent), int(bp), int(ep), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "log")
	}
	return rideInt(r), nil
}

func roundingMode(v rideType) (decimal.RoundingMode, error) {
	switch v.instanceOf() {
	case "Ceiling":
		return decimal.ToPositiveInf, nil
	case "Floor":
		return decimal.ToNegativeInf, nil
	case "HalfEven":
		return decimal.ToNearestEven, nil
	case "Down":
		return decimal.ToZero, nil
	case "Up":
		return decimal.AwayFromZero, nil
	case "HalfUp":
		return decimal.ToNearestAway, nil
	case "HalfDown":
		return decimal.ToNearestTowardZero, nil
	default:
		return 0, errors.Errorf("unable to get rounding mode from '%s'", v.instanceOf())
	}
}
