package ride

import (
	"encoding/binary"
	"math/big"
	"strconv"

	"github.com/ericlagergren/decimal"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/math"
)

func intArg(args []RideType) (RideInt, error) {
	if len(args) != 1 {
		return 0, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return 0, errors.Errorf("argument 1 is empty")
	}
	l, ok := args[0].(RideInt)
	if !ok {
		return 0, errors.Errorf("argument 1 is not of type 'Int' but '%s'", args[0].instanceOf())
	}
	return l, nil
}

func twoIntArgs(args []RideType) (RideInt, RideInt, error) {
	if len(args) != 2 {
		return 0, 0, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return 0, 0, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return 0, 0, errors.Errorf("argument 2 is empty")
	}
	l1, ok := args[0].(RideInt)
	if !ok {
		return 0, 0, errors.Errorf("argument 1 is not of type 'Int' but '%s'", args[0].instanceOf())
	}
	l2, ok := args[1].(RideInt)
	if !ok {
		return 0, 0, errors.Errorf("argument 2 is not of type 'Int' but '%s'", args[1].instanceOf())
	}
	return l1, l2, nil
}

func intArgs(args []RideType, count int) ([]RideInt, error) {
	if len(args) != count {
		return nil, errors.Errorf("%d is invalid number of arguments, expected %d", len(args), count)
	}
	r := make([]RideInt, len(args))
	for n, arg := range args {
		if arg == nil {
			return nil, errors.Errorf("argument %d is empty", n+1)
		}
		l, ok := arg.(RideInt)
		if !ok {
			return nil, errors.Errorf("argument %d is not of type 'Int' but '%s'", n+1, arg.instanceOf())
		}
		r[n] = l
	}
	return r, nil
}

func ge(_ Environment, args ...RideType) (RideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "ge")
	}
	return RideBoolean(l1 >= l2), nil
}

func gt(_ Environment, args ...RideType) (RideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "gt")
	}
	return RideBoolean(l1 > l2), nil
}

func intToString(_ Environment, args ...RideType) (RideType, error) {
	l, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "intToString")
	}
	return RideString(strconv.Itoa(int(l))), nil
}

func unaryMinus(_ Environment, args ...RideType) (RideType, error) {
	l, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryMinus")
	}
	return -l, nil
}

func sum(_ Environment, args ...RideType) (RideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sum")
	}
	return l1 + l2, nil
}

func sub(_ Environment, args ...RideType) (RideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sub")
	}
	return l1 - l2, nil
}

func mul(_ Environment, args ...RideType) (RideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "mul")
	}
	return l1 * l2, nil
}

func div(_ Environment, args ...RideType) (RideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "div")
	}
	if l2 == 0 {
		return nil, errors.New("div: division by zero")
	}
	return RideInt(math.FloorDiv(int64(l1), int64(l2))), nil
}

func mod(_ Environment, args ...RideType) (RideType, error) {
	i1, i2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "mod")
	}
	if i2 == 0 {
		return nil, errors.New("mod: division by zero")
	}
	return RideInt(math.ModDivision(int64(i1), int64(i2))), nil
}

func fraction(_ Environment, args ...RideType) (RideType, error) {
	values, err := intArgs(args, 3)
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	res, err := math.Fraction(int64(values[0]), int64(values[1]), int64(values[2]))
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	return RideInt(res), nil
}

func fractionIntRounds(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 4); err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	value, ok := args[0].(RideInt)
	if !ok {
		return nil, errors.Errorf("fraction: unexpected argument type '%s'", args[0].instanceOf())
	}
	v := big.NewInt(int64(value))
	numerator, ok := args[1].(RideInt)
	if !ok {
		return nil, errors.Errorf("fraction: unexpected argument type '%s'", args[1].instanceOf())
	}
	n := big.NewInt(int64(numerator))
	denominator, ok := args[2].(RideInt)
	if !ok {
		return nil, errors.Errorf("fraction: unexpected argument type '%s'", args[2].instanceOf())
	}
	d := big.NewInt(int64(denominator))
	round, err := roundingMode(args[3])
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	r, err := fractionBigIntLikeInScala(v, n, d, round)
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	if !r.IsInt64() {
		return nil, errors.New("fraction: result is out of int64 range")
	}
	return RideInt(r.Int64()), nil
}

func intToBytes(_ Environment, args ...RideType) (RideType, error) {
	i, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "intToBytes")
	}

	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, uint64(i))
	return RideBytes(out), nil
}

func pow(env Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 6); err != nil {
		return nil, errors.Wrap(err, "pow")
	}
	base, ok := args[0].(RideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[0].instanceOf())
	}
	bp, ok := args[1].(RideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[1].instanceOf())
	}
	exponent, ok := args[2].(RideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[2].instanceOf())
	}
	ep, ok := args[3].(RideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[3].instanceOf())
	}
	rp, ok := args[4].(RideInt)
	if !ok {
		return nil, errors.Errorf("pow: unexpected argument type '%s'", args[4].instanceOf())
	}
	round, err := roundingMode(args[5])
	if err != nil {
		return nil, errors.Wrap(err, "pow")
	}
	f := math.PowV1
	if env.validateInternalPayments() {
		f = math.PowV2
	}
	r, err := f(int64(base), int64(exponent), int(bp), int(ep), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "pow")
	}
	return RideInt(r), nil
}

func log(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 6); err != nil {
		return nil, errors.Wrap(err, "log")
	}
	base, ok := args[0].(RideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[0].instanceOf())
	}
	bp, ok := args[1].(RideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[1].instanceOf())
	}
	exponent, ok := args[2].(RideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[2].instanceOf())
	}
	ep, ok := args[3].(RideInt)
	if !ok {
		return nil, errors.Errorf("log: unexpected argument type '%s'", args[3].instanceOf())
	}
	rp, ok := args[4].(RideInt)
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
	return RideInt(r), nil
}

func roundingMode(v RideType) (decimal.RoundingMode, error) {
	switch v.instanceOf() {
	case "Ceiling":
		return decimal.ToPositiveInf, nil
	case "Floor":
		return decimal.ToNegativeInf, nil
	case "HalfEven":
		return decimal.ToNearestEven, nil
	case "Down":
		return decimal.ToZero, nil
	case "Up": // round-up v2-v4
		return decimal.AwayFromZero, nil
	case "HalfUp":
		return decimal.ToNearestAway, nil
	case "HalfDown": // round-half-down v2-v4
		return decimal.ToNearestTowardZero, nil
	default:
		return 0, errors.Errorf("unable to get rounding mode from '%s'", v.instanceOf())
	}
}
