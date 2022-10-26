package ride

import (
	"math/big"
	"sort"

	"github.com/ericlagergren/decimal"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/math"
)

var (
	zeroBigInt = big.NewInt(0)
)

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
	b := new(big.Int).Set(base.v)
	e := new(big.Int).Set(exponent.v)
	r, err := math.PowBigInt(b, e, int(bp), int(ep), int(rp), round)
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
	b := new(big.Int).Set(base.v)
	e := new(big.Int).Set(exponent.v)
	r, err := math.LogBigInt(b, e, int(bp), int(ep), int(rp), round)
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
	i1 := new(big.Int).Set(a.v)
	i2 := new(big.Int).Set(b.v)
	r := i1.Add(i1, i2)
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
	i1 := new(big.Int).Set(a.v)
	i2 := new(big.Int).Set(b.v)
	r := i1.Sub(i1, i2)
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
	i1 := new(big.Int).Set(a.v)
	i2 := new(big.Int).Set(b.v)
	r := i1.Mul(i1, i2)
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
	i1 := new(big.Int).Set(a.v)
	i2 := new(big.Int).Set(b.v)
	if i2.Cmp(zeroBigInt) == 0 {
		return nil, errors.New("divideBigInt: division by zero")
	}
	r := i1.Quo(i1, i2)
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
	i1 := new(big.Int).Set(a.v)
	i2 := new(big.Int).Set(b.v)
	if i2.Cmp(zeroBigInt) == 0 {
		return nil, errors.New("moduloBigInt: division by zero")
	}
	r := i1.Rem(i1, i2)
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
	v := new(big.Int).Set(a.v)
	n := new(big.Int).Set(b.v)
	d := new(big.Int).Set(c.v)
	if d.Cmp(zeroBigInt) == 0 {
		return nil, errors.New("fractionBigInt: division by zero")
	}
	r := v.Mul(v, n)
	r = r.Quo(r, d)
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
	v := new(big.Int).Set(v1.v)
	v2, ok := args[1].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("fractionBigIntRounds: unexpected argument type '%s'", args[1].instanceOf())
	}
	n := new(big.Int).Set(v2.v)
	v3, ok := args[2].(rideBigInt)
	if !ok {
		return nil, errors.Errorf("fractionBigIntRounds: unexpected argument type '%s'", args[2].instanceOf())
	}
	d := new(big.Int).Set(v3.v)
	round, err := roundingMode(args[3])
	if err != nil {
		return nil, errors.Wrap(err, "fractionBigIntRounds")
	}
	r, err := fractionBigIntLikeInScala(v, n, d, round)
	if err != nil {
		return nil, errors.Wrap(err, "fractionBigIntRounds")
	}
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("fractionBigIntRounds: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

// fractionBigIntLikeInScala the algo is fully taken from Scala implementation.
func fractionBigIntLikeInScala(v, n, d *big.Int, roundingMode decimal.RoundingMode) (*big.Int, error) {
	if d.Cmp(zeroBigInt) == 0 {
		return nil, errors.New("division by zero")
	}
	p := v.Mul(v, n)
	s := big.NewInt(int64(p.Sign() * d.Sign()))
	pa := p.Abs(p)
	da := d.Abs(d)
	r, m := pa.QuoRem(pa, da, big.NewInt(0))
	ms := big.NewInt(int64(m.Sign()))
	switch roundingMode {
	case decimal.ToZero: // Down
		r = r.Mul(r, s)
	case decimal.AwayFromZero: // Up
		r = r.Add(r, ms)
		r = r.Mul(r, s)
	case decimal.ToNearestAway: // HalfUp
		x := d.Abs(d)
		y := m.Mul(m, big.NewInt(2))
		x = x.Sub(x, y)
		switch x.Cmp(zeroBigInt) {
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
		if x.Cmp(zeroBigInt) < 0 {
			r = r.Add(r, big.NewInt(1))
			r = r.Mul(r, s)
		} else {
			r = r.Mul(r, s)
		}
	case decimal.ToPositiveInf: // Ceiling
		if s.Cmp(zeroBigInt) > 0 {
			r = r.Add(r, ms)
		}
		r = r.Mul(r, s)
	case decimal.ToNegativeInf: // Floor
		if s.Cmp(zeroBigInt) < 0 {
			r = r.Add(r, ms)
		}
		r = r.Mul(r, s)
	case decimal.ToNearestEven: // HalfEven
		x := d.Abs(d)
		y := m.Mul(m, big.NewInt(2))
		x = x.Sub(x, y)
		switch x.Cmp(zeroBigInt) {
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
		return nil, errors.New("unsupported rounding mode")
	}
	return r, nil
}

func unaryMinusBigInt(_ environment, args ...rideType) (rideType, error) {
	v, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "unaryMinusBigInt")
	}
	i := new(big.Int).Set(v.v)
	if i.Cmp(math.MinBigInt) == 0 {
		return nil, errors.New("unaryMinusBigInt: positive BigInt overflow")
	}
	r := i.Neg(i)
	return rideBigInt{v: r}, nil
}

func gtBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "gtBigInt")
	}
	i1 := new(big.Int).Set(a.v)
	i2 := new(big.Int).Set(b.v)
	r := i1.Cmp(i2)
	return rideBoolean(r > 0), nil
}

func geBigInt(_ environment, args ...rideType) (rideType, error) {
	a, b, err := twoBigIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "geBigInt")
	}
	i1 := new(big.Int).Set(a.v)
	i2 := new(big.Int).Set(b.v)
	r := i1.Cmp(i2)
	return rideBoolean(r >= 0), nil
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
	items, err := toBigIntSlice(list)
	if err != nil {
		return nil, errors.Wrap(err, "maxListBigInt")
	}
	_, max := minMaxBigInt(items)
	return rideBigInt{v: max}, nil
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
	items, err := toBigIntSlice(list)
	if err != nil {
		return nil, errors.Wrap(err, "minListBigInt")
	}
	min, _ := minMaxBigInt(items)
	return rideBigInt{v: min}, nil
}

func bigIntToBytes(_ environment, args ...rideType) (rideType, error) {
	v, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bigIntToBytes")
	}
	i := new(big.Int).Set(v.v)
	return rideBytes(encode2CBigInt(i)), nil
}

func bytesToBigInt(_ environment, args ...rideType) (rideType, error) {
	bts, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesToBigInt")
	}
	if l := len(bts); l > 64 { // No more than 64 bytes can be converted to BigInt, max size of BigInt value is 512 bit.
		return nil, errors.Errorf("bytesToBigInt: bytes array is too long (%d) for a BigInt", l)
	}
	r := decode2CBigInt(bts)
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.Errorf("bytesToBigInt: %s result is out of range", r.String())
	}
	return rideBigInt{v: r}, nil
}

func bytesToBigIntLim(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "bytesToBigIntLim")
	}
	bts, ok := args[0].(rideBytes)
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
	end := int(offset + size)
	if end > l {
		end = l
	}
	r := decode2CBigInt(bts[offset:end])
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
	i := new(big.Int).Set(v.v)
	if !i.IsInt64() {
		return nil, errors.Errorf("bigIntToInt: value (%s) is too big for an Int", i.String())
	}
	return rideInt(i.Int64()), nil
}

func bigIntToString(_ environment, args ...rideType) (rideType, error) {
	v, err := bigIntArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "bigIntToString")
	}
	i := new(big.Int).Set(v.v)
	return rideString(i.String()), nil
}

func stringToBigInt(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringToBigInt")
	}
	if l := len(s); l > 155 { // 155 symbols is the length of math.MinBigInt value is string representation
		return nil, errors.Errorf("stringToBigInt: string is too long (%d symbols) for a BigInt", l)
	}
	r, ok := new(big.Int).SetString(string(s), 10)
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
	sort.Sort(items)
	half := size / 2
	if size%2 == 1 {
		return rideBigInt{v: items[half]}, nil
	}
	x := items[half-1]
	y := items[half]
	r := math.FloorDivBigInt(x.Add(x, y), big.NewInt(2))
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
	v := new(big.Int).Set(n.v)
	r, err := math.SqrtBigInt(v, int(np), int(rp), round)
	if err != nil {
		return nil, errors.Wrap(err, "sqrtBigInt")
	}
	if r.Cmp(math.MinBigInt) < 0 || r.Cmp(math.MaxBigInt) > 0 {
		return nil, errors.New("sqrtBigInt: result is out of range")
	}
	return rideBigInt{v: r}, nil
}

func minMaxBigInt(items []*big.Int) (*big.Int, *big.Int) {
	if len(items) == 0 {
		panic("empty slice")
	}
	max, min := items[0], items[0]
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
		items[i] = new(big.Int).Set(item.v)
	}
	return items, nil
}

// decode2CBigInt decodes two's complement representation of BigInt from bytes slice
func decode2CBigInt(bytes []byte) *big.Int {
	r := new(big.Int)
	if len(bytes) > 0 && bytes[0]&0x80 == 0x80 { // Decode a negative number
		notBytes := make([]byte, len(bytes))
		for i := range notBytes {
			notBytes[i] = ^bytes[i]
		}
		r.SetBytes(notBytes)
		r.Add(r, math.OneBigInt)
		r.Neg(r)
		return r
	}
	r.SetBytes(bytes)
	return r
}

// encode2CBigInt encodes BigInt into a two's compliment representation
func encode2CBigInt(n *big.Int) []byte {
	if n.Sign() < 0 {
		// Convert negative number into two's complement form
		// Subtract 1 and invert
		// If the most-significant-bit isn't set then we'll need to pad the beginning with 0xff in order to keep the number negative
		nMinus1 := new(big.Int).Neg(n)
		nMinus1.Sub(nMinus1, math.OneBigInt)
		bytes := nMinus1.Bytes()
		for i := range bytes {
			bytes[i] ^= 0xff
		}
		if l := len(bytes); l == 0 || bytes[0]&0x80 == 0 {
			return padBytes(0xff, bytes)
		}
		return bytes
	} else if n.Sign() == 0 { // Zero is written as a single 0 zero rather than no bytes
		return []byte{0x00}
	} else {
		bytes := n.Bytes()
		if len(bytes) > 0 && bytes[0]&0x80 != 0 { // We'll have to pad this with 0x00 in order to stop it looking like a negative number
			return padBytes(0x00, bytes)
		}
		return bytes
	}
}

func padBytes(p byte, bytes []byte) []byte {
	r := make([]byte, len(bytes)+1)
	r[0] = p
	copy(r[1:], bytes)
	return r
}
