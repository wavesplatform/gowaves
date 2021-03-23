package ride

import (
	"math/big"
	"sort"

	"github.com/ericlagergren/decimal"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/math"
)

var (
	minBigInt, maxBigInt = initBoundaries()
)

func initBoundaries() (*big.Int, *big.Int) {
	var two, e, one, zero = big.NewInt(2), big.NewInt(511), big.NewInt(1), big.NewInt(0)
	max := two.Exp(two, e, nil)
	max = max.Sub(two, one)
	min := zero.Sub(zero, max)
	min = min.Sub(min, one)
	return min, max
}

func bigIntArg(args []rideType) (rideBigInt, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument 1 is empty")
	}
	l, ok := args[0].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("argument 1 is not of type 'BigInt' but '%s'", args[0].instanceOf())
	}
	return l, nil
}

func twoBigIntArgs(args []rideType) (rideBigInt, rideBigInt, error) {
	if len(args) != 2 {
		return nil, nil, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, nil, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, nil, errors.Errorf("argument 2 is empty")
	}
	v1, ok := args[0].(rideBigInt)
	if !ok {
		return nil, nil, errors.Errorf("argument 1 is not of type 'BigInt' but '%s'", args[0].instanceOf())
	}
	v2, ok := args[1].(rideBigInt)
	if !ok {
		return nil, nil, errors.Errorf("argument 2 is not of type 'BigInt' but '%s'", args[1].instanceOf())
	}
	return v1, v2, nil
}

func threeBigIntArgs(args []rideType) (rideBigInt, rideBigInt, rideBigInt, error) {
	if len(args) != 3 {
		return nil, nil, nil, errors.Errorf("%d is invalid number of arguments, expected 3", len(args))
	}
	if args[0] == nil {
		return nil, nil, nil, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, nil, nil, errors.Errorf("argument 2 is empty")
	}
	if args[2] == nil {
		return nil, nil, nil, errors.Errorf("argument 3 is empty")
	}
	v1, ok := args[0].(rideBigInt)
	if !ok {
		return nil, nil, nil, errors.Errorf("argument 1 is not of type 'BigInt' but '%s'", args[0].instanceOf())
	}
	v2, ok := args[1].(rideBigInt)
	if !ok {
		return nil, nil, nil, errors.Errorf("argument 2 is not of type 'BigInt' but '%s'", args[1].instanceOf())
	}
	v3, ok := args[2].(rideBigInt)
	if !ok {
		return nil, nil, nil, errors.Errorf("argument 3 is not of type 'BigInt' but '%s'", args[2].instanceOf())
	}
	return v1, v2, v3, nil
}

func powBigInt(_ Environment, args ...rideType) (rideType, error) {
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
	b := big.NewInt(0).SetBytes(base)
	e := big.NewInt(0).SetBytes(exponent)
	r, err := math.PowBigInt(b, e, int(bp), int(ep), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "powBigInt")
	}
	return rideBigInt(r.Bytes()), nil
}

func logBigInt(_ Environment, args ...rideType) (rideType, error) {
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
	b := big.NewInt(0).SetBytes(base)
	e := big.NewInt(0).SetBytes(exponent)
	r, err := math.LogBigInt(b, e, int(bp), int(ep), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "logBigInt")
	}
	return rideBigInt(r.Bytes()), nil
}

func toBigInt(_ Environment, args ...rideType) (rideType, error) {
	i, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "toBigInt")
	}
	v := big.NewInt(int64(i))
	return rideBigInt(v.Bytes()), nil
}

func sumBigInt(_ Environment, args ...rideType) (rideType, error) {
	v1, v2, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sumBigInt")
	}
	i1 := big.NewInt(0).SetBytes(v1)
	i2 := big.NewInt(0).SetBytes(v2)
	r := i1.Add(i1, i2)
	if r.Cmp(maxBigInt) > 0 || r.Cmp(minBigInt) < 0 {
		return nil, errors.Errorf("sumBigInt: %s result is out of range", r.String())
	}
	return rideBigInt(r.Bytes()), nil
}

func subtractBigInt(_ Environment, args ...rideType) (rideType, error) {
	v1, v2, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "subtractBigInt")
	}
	i1 := big.NewInt(0).SetBytes(v1)
	i2 := big.NewInt(0).SetBytes(v2)
	r := i1.Sub(i1, i2)
	if r.Cmp(maxBigInt) > 0 || r.Cmp(minBigInt) < 0 {
		return nil, errors.Errorf("subtractBigInt: %s result is out of range", r.String())
	}
	return rideBigInt(r.Bytes()), nil
}

func multiplyBigInt(_ Environment, args ...rideType) (rideType, error) {
	v1, v2, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "multiplyBigInt")
	}
	i1 := big.NewInt(0).SetBytes(v1)
	i2 := big.NewInt(0).SetBytes(v2)
	r := i1.Mul(i1, i2)
	if r.Cmp(maxBigInt) > 0 || r.Cmp(minBigInt) < 0 {
		return nil, errors.Errorf("multiplyBigInt: %s result is out of range", r.String())
	}
	return rideBigInt(r.Bytes()), nil
}

func divideBigInt(_ Environment, args ...rideType) (rideType, error) {
	v1, v2, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "divideBigInt")
	}
	i1 := big.NewInt(0).SetBytes(v1)
	i2 := big.NewInt(0).SetBytes(v2)
	if i2.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("divideBigInt: division by zero")
	}
	r := i1.Div(i1, i2)
	if r.Cmp(maxBigInt) > 0 || r.Cmp(minBigInt) < 0 {
		return nil, errors.Errorf("divideBigInt: %s result is out of range", r.String())
	}
	return rideBigInt(r.Bytes()), nil
}

func moduloBigInt(_ Environment, args ...rideType) (rideType, error) {
	v1, v2, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "moduloBigInt")
	}
	i1 := big.NewInt(0).SetBytes(v1)
	i2 := big.NewInt(0).SetBytes(v2)
	if i2.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("moduloBigInt: division by zero")
	}
	r := i1.Mod(i1, i2)
	if r.Cmp(maxBigInt) > 0 || r.Cmp(minBigInt) < 0 {
		return nil, errors.Errorf("moduloBigInt: %s result is out of range", r.String())
	}
	return rideBigInt(r.Bytes()), nil
}

func fractionBigInt(_ Environment, args ...rideType) (rideType, error) {
	v1, v2, v3, err := threeBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "fractionBigInt")
	}
	v := big.NewInt(0).SetBytes(v1)
	n := big.NewInt(0).SetBytes(v2)
	d := big.NewInt(0).SetBytes(v3)
	if d.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("fractionBigInt: division by zero")
	}
	r := v.Mul(v, n)
	r = r.Div(r, d)
	if r.Cmp(maxBigInt) > 0 || r.Cmp(minBigInt) < 0 {
		return nil, errors.Errorf("fractionBigInt: %s result is out of range", r.String())
	}
	return rideBigInt(r.Bytes()), nil
}

func fractionBigIntRounds(_ Environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 4); err != nil {
		return nil, errors.Wrap(err, "fractionBigIntRounds")
	}
	v1, ok := args[0].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("fractionBigIntRounds: unexpected argument type '%s'", args[0].instanceOf())
	}
	v := big.NewInt(0).SetBytes(v1)
	v2, ok := args[1].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("fractionBigIntRounds: unexpected argument type '%s'", args[1].instanceOf())
	}
	n := big.NewInt(0).SetBytes(v2)
	v3, ok := args[2].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("fractionBigIntRounds: unexpected argument type '%s'", args[2].instanceOf())
	}
	d := big.NewInt(0).SetBytes(v3)
	if d.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("fractionBigIntRounds: division by zero")
	}
	round, err := roundingMode(args[3])
	if err != nil {
		return nil, errors.Wrap(err, "fractionBigIntRounds")
	}
	p := v.Mul(v, n)
	s := big.NewInt(int64(p.Sign() * d.Sign()))
	pa := p.Abs(p)
	da := d.Abs(d)
	m := big.NewInt(0)
	r, m := pa.DivMod(pa, da, m)
	ms := big.NewInt(int64(m.Sign()))
	switch round {
	case decimal.ToZero: // Down
		r = r.Mul(r, s)
	case decimal.AwayFromZero: // Up
		r = r.Add(r, ms)
		r = r.Mul(r, s)
	case decimal.ToNearestAway: // HalfUp
		x := d.Abs(d)
		y := m.Mul(m, big.NewInt(2))
		x = x.Sub(x, y)
		switch x.Cmp(big.NewInt(0)) {
		case -1:
			r = r.Add(r, big.NewInt(1))
			r = r.Mul(r, s)
		case 0:
			r = r.Add(r, big.NewInt(1))
			r = r.Mul(r, s)
		case 1:
			r = r.Mul(r, s)
		}
	case decimal.ToNearestTowardZero: // RoundHalfDown
		x := d.Abs(d)
		y := m.Mul(m, big.NewInt(2))
		x = x.Sub(x, y)
		if x.Cmp(big.NewInt(0)) < 0 {
			r = r.Add(r, big.NewInt(1))
			r = r.Mul(r, s)
		} else {
			r = r.Mul(r, s)
		}
	case decimal.ToPositiveInf: // Ceiling
		if s.Cmp(big.NewInt(0)) > 0 {
			r = r.Add(r, ms)
		}
		r = r.Mul(r, s)
	case decimal.ToNegativeInf: // Floor
		if s.Cmp(big.NewInt(0)) < 0 {
			r = r.Add(r, ms)
		}
		r = r.Mul(r, s)
	case decimal.ToNearestEven: // HalfEven
		x := d.Abs(d)
		y := m.Mul(m, big.NewInt(2))
		x = x.Sub(x, y)
		switch x.Cmp(big.NewInt(0)) {
		case -1:
			r = r.Add(r, big.NewInt(1))
			r = r.Mul(r, s)
		case 1:
			r = r.Mul(r, s)
		case 0:
			r2 := big.NewInt(2)
			r2 = r2.Mod(r, r2)
			r = r.Add(r, r2)
			r = r.Mul(r, s)
		}
	default:
		return nil, errors.New("fractionBigIntRounds: unsupported rounding mode")
	}
	return rideBigInt(r.Bytes()), nil
}

func unaryMinusBigInt(_ Environment, args ...rideType) (rideType, error) {
	v, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryMinusBigInt")
	}
	i := big.NewInt(0).SetBytes(v)
	if i.Cmp(minBigInt) == 0 {
		return nil, errors.New("unaryMinusBigInt: positive BigInt overflow")
	}
	zero := big.NewInt(0)
	r := zero.Sub(zero, i)
	return rideBigInt(r.Bytes()), nil
}

func gtBigInt(_ Environment, args ...rideType) (rideType, error) {
	v1, v2, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "gtBigInt")
	}
	i1 := big.NewInt(0).SetBytes(v1)
	i2 := big.NewInt(0).SetBytes(v2)
	r := i1.Cmp(i2)
	return rideBoolean(r > 0), nil
}

func geBigInt(_ Environment, args ...rideType) (rideType, error) {
	v1, v2, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "geBigInt")
	}
	i1 := big.NewInt(0).SetBytes(v1)
	i2 := big.NewInt(0).SetBytes(v2)
	r := i1.Cmp(i2)
	return rideBoolean(r >= 0), nil
}

func maxListBigInt(_ Environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "maxListBigInt")
	}
	size := len(list)
	if size > maxListSize || size == 0 {
		return nil, errors.Errorf("maxListBigInt: invalid list size %d", size)
	}
	items, err := toBigIntSlice(list)
	if err != nil {
		return nil, errors.Wrap(err, "maxListBigInt")
	}
	_, max := minMaxBigInt(items)
	return rideBigInt(max.Bytes()), nil
}

func minListBigInt(_ Environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "minListBigInt")
	}
	size := len(list)
	if size > maxListSize || size == 0 {
		return nil, errors.Errorf("minListBigInt: invalid list size %d", size)
	}
	items, err := toBigIntSlice(list)
	if err != nil {
		return nil, errors.Wrap(err, "minListBigInt")
	}
	min, _ := minMaxBigInt(items)
	return rideBigInt(min.Bytes()), nil
}

func bigIntToBytes(_ Environment, args ...rideType) (rideType, error) {
	i, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bigIntToBytes")
	}
	return rideBytes(i), nil
}

func bytesToBigInt(_ Environment, args ...rideType) (rideType, error) {
	bts, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToBigInt")
	}
	if l := len(bts); l > 64 {
		return nil, errors.Errorf("bytesToBigInt: bytes array is too long (%d) for a BigInt", l)
	}
	i := big.NewInt(0).SetBytes(bts)
	return rideBigInt(i.Bytes()), nil
}

func bytesToBigIntLim(_ Environment, args ...rideType) (rideType, error) {
	if len(args) != 3 {
		return nil, errors.Errorf("bytesToBigIntLim: %d is invalid number of arguments, expected 3", len(args))
	}
	if args[0] == nil {
		return nil, errors.New("bytesToBigIntLim: argument 1 is empty")
	}
	if args[1] == nil {
		return nil, errors.New("bytesToBigIntLim: argument 2 is empty")
	}
	if args[2] == nil {
		return nil, errors.New("bytesToBigIntLim: argument 3 is empty")
	}
	bts, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("bytesToBigIntLim: argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	offset, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("bytesToBigIntLim: argument 2 is not of type 'Int' but '%s'", args[1].instanceOf())
	}
	if offset < 0 || int(offset) >= len(bts) {
		return nil, errors.Errorf("bytesToBigIntLim: offset %d is out of range [0; %d]", offset, len(bts)-1)
	}
	size, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("bytesToBigIntLim: argument 3 is not of type 'Int' but '%s'", args[2].instanceOf())
	}
	if size < 0 || size > 64 {
		return nil, errors.Errorf("bytesToBigIntLim: size %d is out of ranger [0; 64]", size)
	}
	end := int(offset + size)
	if last := len(bts) - 1; end > last {
		end = last
	}
	i := big.NewInt(0).SetBytes(bts[offset:end])
	return rideBigInt(i.Bytes()), nil
}

func bigIntToInt(_ Environment, args ...rideType) (rideType, error) {
	b, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bigIntToInt")
	}
	i := big.NewInt(0).SetBytes(b)
	if !i.IsInt64() {
		return nil, errors.Errorf("bigIntToInt: value (%s) is too big for an Int", i.String())
	}
	return rideInt(i.Int64()), nil
}

func bigIntToString(_ Environment, args ...rideType) (rideType, error) {
	b, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bigIntToString")
	}
	i := big.NewInt(0).SetBytes(b)
	return rideString(i.String()), nil
}

func stringToBigInt(_ Environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringToBigInt")
	}
	if l := len(s); l > 155 {
		return nil, errors.Errorf("stringToBigInt: string is too long (%d symbols) for a BigInt", l)
	}
	i, ok := big.NewInt(0).SetString(string(s), 10)
	if !ok {
		return nil, errors.Errorf("stringToBigInt: failed to convert string '%s' to BigInt", s)
	}
	if i.Cmp(minBigInt) < 0 || i.Cmp(maxBigInt) > 0 {
		return nil, errors.New("stringToBigInt: value too big for a BigInt")
	}
	return rideBigInt(i.Bytes()), nil
}

func stringToBigIntOpt(env Environment, args ...rideType) (rideType, error) {
	v, err := stringToBigInt(env, args...)
	if err != nil {
		return newUnit(env), nil
	}
	return v, nil
}

func medianListBigInt(_ Environment, args ...rideType) (rideType, error) {
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
	sort.Sort(items)
	half := size / 2
	if size%2 == 1 {
		return rideBigInt(items[half].Bytes()), nil
	} else {
		x := items[half-1]
		y := items[half]
		r := math.FloorDivBigInt(x.Add(x, y), big.NewInt(2))
		return rideBigInt(r.Bytes()), nil
	}
}

func minMaxBigInt(items []*big.Int) (*big.Int, *big.Int) {
	if len(items) == 0 {
		panic("empty slice")
	}
	max := items[0]
	min := items[0]
	for _, i := range items {
		if i.Cmp(max) > 0 {
			max = i
		}
		if i.Cmp(min) < 0 {
			min = i
		}
	}
	return min, max
}

type bigIntSlice []*big.Int

func (x bigIntSlice) Len() int           { return len(x) }
func (x bigIntSlice) Less(i, j int) bool { return x[i].Cmp(x[j]) < 0 }
func (x bigIntSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func toBigIntSlice(list rideList) (bigIntSlice, error) {
	items := make([]*big.Int, len(list))
	for i, el := range list {
		item, ok := el.(rideBigInt)
		if !ok {
			return nil, errors.Errorf("unexpected type of list element '%s'", el.instanceOf())
		}
		items[i] = big.NewInt(0).SetBytes(item)
	}
	return items, nil
}
