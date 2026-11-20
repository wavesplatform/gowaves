package ride

import (
	"math/big"
	"slices"
	"unicode/utf8"

	"github.com/ericlagergren/decimal"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/math"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const base10 = 10

func bigIntArg(args []rideType) (rideBigInt, error) {
	if len(args) != 1 {
		return rideBigInt{}, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return rideBigInt{}, errors.Errorf("argument 1 is empty")
	}
	l, ok := args[0].(rideBigInt)
	if !ok {
		return rideBigInt{}, errors.Errorf("argument 1 is not of type 'BigInt' but '%s'", args[0].instanceOf())
	}
	return l, nil
}

func twoBigIntArgs(args []rideType) (rideBigInt, rideBigInt, error) {
	if len(args) != 2 {
		return rideBigInt{}, rideBigInt{}, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return rideBigInt{}, rideBigInt{}, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return rideBigInt{}, rideBigInt{}, errors.Errorf("argument 2 is empty")
	}
	v1, ok := args[0].(rideBigInt)
	if !ok {
		return rideBigInt{}, rideBigInt{}, errors.Errorf("argument 1 is not of type 'BigInt' but '%s'", args[0].instanceOf())
	}
	v2, ok := args[1].(rideBigInt)
	if !ok {
		return rideBigInt{}, rideBigInt{}, errors.Errorf("argument 2 is not of type 'BigInt' but '%s'", args[1].instanceOf())
	}
	return v1, v2, nil
}

func threeBigIntArgs(args []rideType) (rideBigInt, rideBigInt, rideBigInt, error) {
	if len(args) != 3 {
		return rideBigInt{}, rideBigInt{}, rideBigInt{}, errors.Errorf("%d is invalid number of arguments, expected 3", len(args))
	}
	if args[0] == nil {
		return rideBigInt{}, rideBigInt{}, rideBigInt{}, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return rideBigInt{}, rideBigInt{}, rideBigInt{}, errors.Errorf("argument 2 is empty")
	}
	if args[2] == nil {
		return rideBigInt{}, rideBigInt{}, rideBigInt{}, errors.Errorf("argument 3 is empty")
	}
	v1, ok := args[0].(rideBigInt)
	if !ok {
		return rideBigInt{}, rideBigInt{}, rideBigInt{}, errors.Errorf("argument 1 is not of type 'BigInt' but '%s'", args[0].instanceOf())
	}
	v2, ok := args[1].(rideBigInt)
	if !ok {
		return rideBigInt{}, rideBigInt{}, rideBigInt{}, errors.Errorf("argument 2 is not of type 'BigInt' but '%s'", args[1].instanceOf())
	}
	v3, ok := args[2].(rideBigInt)
	if !ok {
		return rideBigInt{}, rideBigInt{}, rideBigInt{}, errors.Errorf("argument 3 is not of type 'BigInt' but '%s'", args[2].instanceOf())
	}
	return v1, v2, v3, nil
}

func powBigInt(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 6); err != nil {
		return nil, errors.Wrap(err, "powBigInt")
	}
	base, ok := args[0].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("powBigInt: unexpected argument type '%s'", args[0].instanceOf())
	}
	bp, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("powBigInt: unexpected argument type '%s'", args[1].instanceOf())
	}
	exponent, ok := args[2].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("powBigInt: unexpected argument type '%s'", args[2].instanceOf())
	}
	ep, ok := args[3].(rideInt)
	if !ok {
		return nil, errors.Errorf("powBigInt: unexpected argument type '%s'", args[3].instanceOf())
	}
	rp, ok := args[4].(rideInt)
	if !ok {
		return nil, errors.Errorf("powBigInt: unexpected argument type '%s'", args[4].instanceOf())
	}
	round, err := roundingMode(args[5])
	if err != nil {
		return nil, errors.Wrap(err, "powBigInt")
	}
	r, err := math.PowBigInt(base.v, exponent.v, int(bp), int(ep), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "powBigInt")
	}
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.New("powBigInt: result is out of range")
	}
	return rideBigInt{v: r}, nil
}

func logBigInt(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 6); err != nil {
		return nil, errors.Wrap(err, "logBigInt")
	}
	base, ok := args[0].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("logBigInt: unexpected argument type '%s'", args[0].instanceOf())
	}
	bp, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("logBigInt: unexpected argument type '%s'", args[1].instanceOf())
	}
	exponent, ok := args[2].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("logBigInt: unexpected argument type '%s'", args[2].instanceOf())
	}
	ep, ok := args[3].(rideInt)
	if !ok {
		return nil, errors.Errorf("logBigInt: unexpected argument type '%s'", args[3].instanceOf())
	}
	rp, ok := args[4].(rideInt)
	if !ok {
		return nil, errors.Errorf("logBigInt: unexpected argument type '%s'", args[4].instanceOf())
	}
	round, err := roundingMode(args[5])
	if err != nil {
		return nil, errors.Wrap(err, "logBigInt")
	}
	r, err := math.LogBigInt(base.v, exponent.v, int(bp), int(ep), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "logBigInt")
	}
	return rideBigInt{v: r}, nil
}

func toBigInt(_ environment, args ...rideType) (rideType, error) {
	i, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "toBigInt")
	}
	v := big.NewInt(int64(i))
	return rideBigInt{v: v}, nil
}

func sumBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sumBigInt")
	}
	r := new(big.Int).Add(a.v, b.v)
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("sumBigInt: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func subtractBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "subtractBigInt")
	}
	r := new(big.Int).Sub(a.v, b.v)
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("subtractBigInt: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func multiplyBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "multiplyBigInt")
	}
	r := new(big.Int).Mul(a.v, b.v)
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("multiplyBigInt: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func divideBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "divideBigInt")
	}
	if b.v.Sign() == 0 {
		return nil, errors.New("divideBigInt: division by zero")
	}
	r := new(big.Int).Quo(a.v, b.v)
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("divideBigInt: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func moduloBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "moduloBigInt")
	}
	if b.v.Sign() == 0 {
		return nil, errors.New("moduloBigInt: division by zero")
	}
	r := new(big.Int).Rem(a.v, b.v)
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("moduloBigInt: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func fractionBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, c, err := threeBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "fractionBigInt")
	}
	if c.v.Sign() == 0 {
		return nil, errors.New("fractionBigInt: division by zero")
	}
	r := new(big.Int).Mul(a.v, b.v)
	r = r.Quo(r, c.v)
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("fractionBigInt: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func fractionBigIntRounds(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 4); err != nil {
		return nil, errors.Wrap(err, "fractionBigIntRounds")
	}
	v1, ok := args[0].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("fractionBigIntRounds: unexpected argument type '%s'", args[0].instanceOf())
	}
	v2, ok := args[1].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("fractionBigIntRounds: unexpected argument type '%s'", args[1].instanceOf())
	}
	v3, ok := args[2].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("fractionBigIntRounds: unexpected argument type '%s'", args[2].instanceOf())
	}
	round, err := roundingMode(args[3])
	if err != nil {
		return nil, errors.Wrap(err, "fractionBigIntRounds")
	}
	r, err := fractionBigIntLikeInScala(v1.v, v2.v, v3.v, round)
	if err != nil {
		return nil, errors.Wrap(err, "fractionBigIntRounds")
	}
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("fractionBigIntRounds: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

// fractionBigIntLikeInScala the algo is fully taken from Scala implementation.
// The arguments are not modified, the result is always a new value.
func fractionBigIntLikeInScala(v, n, d *big.Int, roundingMode decimal.RoundingMode) (*big.Int, error) {
	if d.Sign() == 0 {
		return nil, errors.New("division by zero")
	}
	s := v.Sign() * n.Sign() * d.Sign() // Sign of the result.
	r := new(big.Int).Mul(v, n)
	m := new(big.Int)
	r.QuoRem(r, d, m) // Truncated division, so abs(r) and abs(m) are the quotient and remainder of abs values.
	r.Abs(r)
	m.Abs(m)
	// halfCmp compares the doubled remainder with the abs value of the denominator.
	halfCmp := func() int { return m.Lsh(m, 1).CmpAbs(d) }
	var inc bool
	switch roundingMode {
	case decimal.ToZero: // Down
	case decimal.AwayFromZero: // Up
		inc = m.Sign() > 0
	case decimal.ToNearestAway: // HalfUp
		inc = halfCmp() >= 0
	case decimal.ToNearestTowardZero: // RoundHalfDown
		inc = halfCmp() > 0
	case decimal.ToPositiveInf: // Ceiling
		inc = s > 0 && m.Sign() > 0
	case decimal.ToNegativeInf: // Floor
		inc = s < 0 && m.Sign() > 0
	case decimal.ToNearestEven: // HalfEven
		c := halfCmp()
		inc = c > 0 || (c == 0 && r.Bit(0) == 1)
	default:
		return nil, errors.New("unsupported rounding mode")
	}
	if inc {
		r.Add(r, math.OneBigInt)
	}
	switch {
	case s < 0:
		r.Neg(r)
	case s == 0:
		r.SetInt64(0)
	}
	return r, nil
}

func unaryMinusBigInt(_ environment, args ...rideType) (rideType, error) {
	v, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryMinusBigInt")
	}
	if v.v.Cmp(math.MinBigInt) == 0 {
		return nil, errors.New("unaryMinusBigInt: positive BigInt overflow")
	}
	r := new(big.Int).Neg(v.v)
	return rideBigInt{v: r}, nil
}

func gtBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "gtBigInt")
	}
	return rideBoolean(a.v.Cmp(b.v) > 0), nil
}

func geBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "geBigInt")
	}
	return rideBoolean(a.v.Cmp(b.v) >= 0), nil
}

func maxListBigInt(_ environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "maxListBigInt")
	}
	size := len(list)
	if size > maxListSize || size == 0 {
		return nil, errors.Errorf("maxListBigInt: invalid list size %d", size)
	}
	_, maxV, err := minMaxListBigInt(list)
	if err != nil {
		return nil, errors.Wrap(err, "maxListBigInt")
	}
	return rideBigInt{v: maxV}, nil
}

func minListBigInt(_ environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "minListBigInt")
	}
	size := len(list)
	if size > maxListSize || size == 0 {
		return nil, errors.Errorf("minListBigInt: invalid list size %d", size)
	}
	minV, _, err := minMaxListBigInt(list)
	if err != nil {
		return nil, errors.Wrap(err, "minListBigInt")
	}
	return rideBigInt{v: minV}, nil
}

func bigIntToBytes(_ environment, args ...rideType) (rideType, error) {
	v, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bigIntToBytes")
	}
	r := common.Encode2CBigInt(v.v)
	return rideByteVector(r), nil
}

func bytesToBigInt(_ environment, args ...rideType) (rideType, error) {
	bts, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToBigInt")
	}
	if l := len(bts); l > 64 { // No more than 64 bytes can be converted to BigInt, max size of BigInt value is 512 bit.
		return nil, errors.Errorf("bytesToBigInt: bytes array is too long (%d) for a BigInt", l)
	}
	r := common.Decode2CBigInt(bts)
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("bytesToBigInt: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func bytesToBigIntLim(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "bytesToBigIntLim")
	}
	bts, ok := args[0].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("bytesToBigIntLim: argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	l := len(bts)
	offset, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("bytesToBigIntLim: argument 2 is not of type 'Int' but '%s'", args[1].instanceOf())
	}
	if offset < 0 || int(offset) >= l {
		return nil, errors.Errorf("bytesToBigIntLim: offset %d is out of range [0; %d]", offset, len(bts)-1)
	}
	size, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("bytesToBigIntLim: argument 3 is not of type 'Int' but '%s'", args[2].instanceOf())
	}
	if size <= 0 || size > 64 { // No more than 64 bytes can be converted to BigInt, max size of BigInt value is 512 bit.
		return nil, errors.Errorf("bytesToBigIntLim: size %d is out of ranger [1; 64]", size)
	}
	end := min(int(offset+size), l)
	r := common.Decode2CBigInt(bts[offset:end])
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("bytesToBigIntLim: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func bigIntToInt(_ environment, args ...rideType) (rideType, error) {
	v, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bigIntToInt")
	}
	if !v.v.IsInt64() {
		return nil, errors.Errorf("bigIntToInt: value (%s) is too big for an Int", v.v.String())
	}
	return rideInt(v.v.Int64()), nil
}

func bigIntToString(_ environment, args ...rideType) (rideType, error) {
	v, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bigIntToString")
	}
	return rideString(v.v.String()), nil
}

func stringToBigInt(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringToBigInt")
	}
	const maxBitIntStringSize = 155 // The maximum allowed length of math.MinBigInt string.
	if l := utf8.RuneCountInString(string(s)); l > maxBitIntStringSize {
		return nil, errors.Errorf("stringToBigInt: string is too long (%d symbols) for a BigInt", l)
	}
	ns, err := normalizeDigits(string(s))
	if err != nil {
		return nil, errors.Wrap(err, "stringToBigInt")
	}
	r, ok := new(big.Int).SetString(ns, base10)
	if !ok {
		return nil, errors.Errorf("stringToBigInt: failed to convert string '%s' to BigInt", s)
	}
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.New("stringToBigInt: value too big for a BigInt")
	}
	return rideBigInt{v: r}, nil
}

func stringToBigIntOpt(env environment, args ...rideType) (rideType, error) {
	v, err := stringToBigInt(env, args...)
	if err != nil {
		return newUnit(env), nil
	}
	return v, nil
}

func medianListBigInt(_ environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "medianListBigInt")
	}
	size := len(list)
	if size > maxListSize || size < 2 {
		return nil, errors.Errorf("medianListBigInt: invalid list size %d", size)
	}
	items, err := toBigIntSlice(list)
	if err != nil {
		return nil, errors.Wrap(err, "medianListBigInt")
	}
	slices.SortFunc(items, (*big.Int).Cmp)
	half := size / 2
	if size%2 == 1 {
		return rideBigInt{v: items[half]}, nil
	}
	// FloorDivBigInt modifies its arguments, so both of them must be new values.
	x := new(big.Int).Add(items[half-1], items[half])
	r := math.FloorDivBigInt(x, big.NewInt(2))
	return rideBigInt{v: r}, nil
}

func sqrtBigInt(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 4); err != nil {
		return nil, errors.Wrap(err, "sqrtBigInt")
	}
	n, ok := args[0].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("sqrtBigInt: unexpected argument type '%s'", args[0].instanceOf())
	}
	np, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("sqrtBigInt: unexpected argument type '%s'", args[1].instanceOf())
	}
	rp, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("sqrtBigInt: unexpected argument type '%s'", args[2].instanceOf())
	}
	round, err := roundingMode(args[3])
	if err != nil {
		return nil, errors.Wrap(err, "sqrtBigInt")
	}
	r, err := math.SqrtBigInt(n.v, int(np), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "sqrtBigInt")
	}
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.New("sqrtBigInt: result is out of range")
	}
	return rideBigInt{v: r}, nil
}

// minMaxListBigInt returns the minimum and maximum values of non-empty BigInt list without copying them.
func minMaxListBigInt(list rideList) (*big.Int, *big.Int, error) {
	var minV, maxV *big.Int
	for _, el := range list {
		item, ok := el.(rideBigInt)
		if !ok {
			return nil, nil, errors.Errorf("unexpected type of list element '%s'", el.instanceOf())
		}
		i := item.v
		if maxV == nil {
			minV, maxV = i, i
			continue
		}
		if i.Cmp(maxV) > 0 {
			maxV = i
		}
		if i.Cmp(minV) < 0 {
			minV = i
		}
	}
	if maxV == nil {
		panic("empty list")
	}
	return minV, maxV, nil
}

// toBigIntSlice returns the values of BigInt list elements.
// The values are shared with the list elements and must not be modified.
func toBigIntSlice(list rideList) ([]*big.Int, error) {
	items := make([]*big.Int, len(list))
	for i, el := range list {
		item, ok := el.(rideBigInt)
		if !ok {
			return nil, errors.Errorf("unexpected type of list element '%s'", el.instanceOf())
		}
		items[i] = item.v
	}
	return items, nil
}
