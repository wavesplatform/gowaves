package ride

import (
	"encoding/binary"
	stdmath "math"
	"math/big"
	"strconv"

	"github.com/ericlagergren/decimal"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/math"
	"github.com/wavesplatform/gowaves/pkg/util/common"
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

func threeIntArgs(args []rideType) (rideInt, rideInt, rideInt, error) {
	const count = 3
	if len(args) != count {
		return 0, 0, 0, errors.Errorf("%d is invalid number of arguments, expected %d", len(args), count)
	}
	var r [count]rideInt
	for n, arg := range args {
		if arg == nil {
			return 0, 0, 0, errors.Errorf("argument %d is empty", n+1)
		}
		l, ok := arg.(rideInt)
		if !ok {
			return 0, 0, 0, errors.Errorf("argument %d is not of type 'Int' but '%s'", n+1, arg.instanceOf())
		}
		r[n] = l
	}
	return r[0], r[1], r[2], nil
}

func ge(_ environment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "ge")
	}
	return rideBoolean(l1 >= l2), nil
}

func gt(_ environment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "gt")
	}
	return rideBoolean(l1 > l2), nil
}

func intToString(_ environment, args ...rideType) (rideType, error) {
	l, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "intToString")
	}
	return rideString(strconv.Itoa(int(l))), nil
}

func unaryMinus(_ environment, args ...rideType) (rideType, error) {
	l, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryMinus")
	}
	return -l, nil
}

func sum(_ environment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sum")
	}
	return common.AddInt(l1, l2)
}

func sub(_ environment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sub")
	}
	return common.SubInt(l1, l2)
}

func mul(_ environment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "mul")
	}
	return common.MulInt(l1, l2)
}

func div(_ environment, args ...rideType) (rideType, error) {
	l1, l2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "div")
	}
	if l2 == 0 {
		return nil, errors.New("div: division by zero")
	}
	return rideInt(math.FloorDiv(int64(l1), int64(l2))), nil
}

func mod(_ environment, args ...rideType) (rideType, error) {
	i1, i2, err := twoIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "mod")
	}
	if i2 == 0 {
		return nil, errors.New("mod: division by zero")
	}
	return rideInt(math.ModDivision(int64(i1), int64(i2))), nil
}

func fraction(_ environment, args ...rideType) (rideType, error) {
	value, numerator, denominator, err := threeIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	res, err := math.Fraction(int64(value), int64(numerator), int64(denominator))
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	return rideInt(res), nil
}

func fractionIntRounds(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 4); err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	value, ok := args[0].(rideInt)
	if !ok {
		return nil, errors.Errorf("fraction: unexpected argument type '%s'", args[0].instanceOf())
	}
	numerator, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("fraction: unexpected argument type '%s'", args[1].instanceOf())
	}
	denominator, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("fraction: unexpected argument type '%s'", args[2].instanceOf())
	}
	round, err := roundingMode(args[3])
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	if r, fits := fractionInt64LikeInScala(int64(value), int64(numerator), int64(denominator), round); fits {
		return rideInt(r), nil
	}
	v := big.NewInt(int64(value))
	n := big.NewInt(int64(numerator))
	d := big.NewInt(int64(denominator))
	r, err := fractionBigIntLikeInScala(v, n, d, round)
	if err != nil {
		return nil, errors.Wrap(err, "fraction")
	}
	if !r.IsInt64() {
		return nil, errors.New("fraction: result is out of int64 range")
	}
	return rideInt(r.Int64()), nil
}

// fractionInt64LikeInScala is the int64 version of fractionBigIntLikeInScala.
// It returns false if the product of v and n or the result doesn't fit in int64,
// or the rounding mode is not supported, in this case fractionBigIntLikeInScala should be used.
func fractionInt64LikeInScala(v, n, d int64, roundingMode decimal.RoundingMode) (int64, bool) {
	p, ok := math.MulInt64(v, n)
	if !ok || d == 0 {
		return 0, false
	}
	s := sign(p) * sign(d) // Sign of the result.
	pa, da := absUint64(p), absUint64(d)
	r, m := pa/da, pa%da
	var inc bool
	switch roundingMode {
	case decimal.ToZero: // Down
	case decimal.AwayFromZero: // Up
		inc = m > 0
	case decimal.ToNearestAway: // HalfUp
		inc = 2*m >= da // No overflow, because m < da <= 2^63.
	case decimal.ToNearestTowardZero: // RoundHalfDown
		inc = 2*m > da
	case decimal.ToPositiveInf: // Ceiling
		inc = s > 0 && m > 0
	case decimal.ToNegativeInf: // Floor
		inc = s < 0 && m > 0
	case decimal.ToNearestEven: // HalfEven
		inc = 2*m > da || (2*m == da && r%2 == 1)
	default:
		return 0, false
	}
	if inc {
		r++
	}
	switch {
	case s > 0 && r <= stdmath.MaxInt64:
		return int64(r), true
	case s < 0 && r <= 1<<63:
		// For r == 2^63 the conversion gives MinInt64 and negation keeps it.
		return -int64(r), true //nolint:gosec // r <= 2^63 is checked above.
	case s == 0:
		return 0, true
	default:
		return 0, false
	}
}

func sign(x int64) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}

func absUint64(x int64) uint64 {
	if x < 0 {
		return uint64(-x) // For MinInt64 the result is 2^63 as expected.
	}
	return uint64(x)
}

func intToBytes(_ environment, args ...rideType) (rideType, error) {
	i, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "intToBytes")
	}

	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, uint64(i))
	return rideByteVector(out), nil
}

func pow(env environment, args ...rideType) (rideType, error) {
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
	f := math.PowV1
	if env.validateInternalPayments() {
		f = math.PowV2
	}
	r, err := f(int64(base), int64(exponent), int(bp), int(ep), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "pow")
	}
	return rideInt(r), nil
}

func log(_ environment, args ...rideType) (rideType, error) {
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

func sqrt(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 4); err != nil {
		return nil, errors.Wrap(err, "sqrt")
	}
	n, ok := args[0].(rideInt)
	if !ok {
		return nil, errors.Errorf("sqrt: unexpected argument type '%s'", args[0].instanceOf())
	}
	np, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("sqrt: unexpected argument type '%s'", args[1].instanceOf())
	}
	rp, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("sqrt: unexpected argument type '%s'", args[2].instanceOf())
	}
	round, err := roundingMode(args[3])
	if err != nil {
		return nil, errors.Wrap(err, "sqrt")
	}
	r, err := math.Sqrt(int64(n), int(np), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "sqrt")
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
