package math

import (
	"math/big"

	"github.com/ericlagergren/decimal"
	"github.com/ericlagergren/decimal/math"
	"github.com/pkg/errors"
)

var (
	zero = decimal.New(0, 0)
	one  = decimal.New(1, 0)
	ten  = decimal.New(10, 0)
)

func checkScales(baseScale, exponentScale, resultScale int) bool {
	// 8 is the maximum scale for RIDE Int values
	return baseScale >= 0 && baseScale <= 8 && exponentScale >= 0 && exponentScale <= 8 && resultScale >= 0 && resultScale <= 8
}

func checkScalesBigInt(baseScale, exponentScale, resultScale int) bool {
	// 18 is the maximum scale for RIDE BigInt values
	return baseScale >= 0 && baseScale <= 18 && exponentScale >= 0 && exponentScale <= 18 && resultScale >= 0 && resultScale <= 18
}

func convertToIntResult(v *decimal.Big, scale int, mode decimal.RoundingMode) (int64, error) {
	context := decimal.Context128
	context.RoundingMode = mode
	r := decimal.WithContext(context).Set(v)
	s := decimal.WithContext(decimal.Context128).SetMantScale(int64(scale), 0)
	m := decimal.WithContext(decimal.Context128)
	math.Pow(m, ten, s)
	r.Mul(r, m)
	res, ok := r.RoundToInt().Int64()
	if !ok {
		return 0, errors.New("result out of int64 range")
	}
	return res, nil
}

func convertToBigIntResult(v *decimal.Big, scale int, mode decimal.RoundingMode) (*big.Int, error) {
	if v.IsNaN(0) || v.IsInf(0) {
		return nil, errors.New("result is NaN or Infinity")
	}
	context := decimal.Context128
	context.RoundingMode = mode
	// r = v * 10^s
	r := decimal.WithContext(context).Set(v)
	s := decimal.WithContext(decimal.Context128).SetMantScale(int64(scale), 0)
	m := decimal.WithContext(decimal.Context128)
	math.Pow(m, ten, s)
	r.Mul(r, m)
	return r.RoundToInt().Int(nil), nil
}

func pow(base, exponent *decimal.Big) (*decimal.Big, error) {
	if base.IsInt() && exponent.Cmp(zero) == 0 {
		return one, nil
	}
	r := decimal.WithContext(decimal.Context128)
	r = math.Pow(r, base, exponent)
	if r.Context.Err() != nil {
		return nil, errors.New(r.Context.Conditions.Error())
	}
	return r, nil
}

func Pow(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if !checkScales(baseScale, exponentScale, resultScale) {
		return 0, errors.New("invalid scale")
	}
	b := decimal.WithContext(decimal.Context128).SetMantScale(base, baseScale)
	e := decimal.WithContext(decimal.Context128).SetMantScale(exponent, exponentScale)
	r, err := pow(b, e)
	if err != nil {
		return 0, err
	}
	return convertToIntResult(r, resultScale, mode)
}

func PowBigInt(base, exponent *big.Int, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (*big.Int, error) {
	if !checkScalesBigInt(baseScale, exponentScale, resultScale) {
		return nil, errors.New("invalid scale")
	}
	b := decimal.WithContext(decimal.Context128).SetBigMantScale(base, baseScale)
	e := decimal.WithContext(decimal.Context128).SetBigMantScale(exponent, exponentScale)
	r, err := pow(b, e)
	if err != nil {
		return nil, err
	}
	return convertToBigIntResult(r, resultScale, mode)
}

func Fraction(value, numerator, denominator int64) (int64, error) {
	v := decimal.WithContext(decimal.Context128).SetMantScale(value, 0)
	n := decimal.WithContext(decimal.Context128).SetMantScale(numerator, 0)
	d := decimal.WithContext(decimal.Context128).SetMantScale(denominator, 0)

	v.Mul(v, n)
	v.Quo(v, d)
	if err := v.Context.Err(); err != nil {
		return 0, errors.Wrap(err, "Fraction")
	}
	res, err := convertToIntResult(v, 0, decimal.ToZero)
	if err != nil {
		return 0, errors.Wrap(err, "Fraction")
	}
	return res, nil
}

func log(base, exponent *decimal.Big, resultScale int) (*decimal.Big, error) {
	r := decimal.WithContext(decimal.Context128).SetMantScale(0, resultScale)
	bl := decimal.WithContext(decimal.Context128)
	math.Log(bl, base)
	if bl.Context.Err() != nil {
		return nil, errors.New(bl.Context.Conditions.Error())
	}
	el := decimal.WithContext(decimal.Context128)
	math.Log(el, exponent)
	if el.Context.Err() != nil {
		return nil, errors.New(el.Context.Conditions.Error())
	}
	r.Quo(bl, el)
	if r.Context.Err() != nil {
		return nil, errors.New(r.Context.Conditions.Error())
	}
	return r, nil
}

func Log(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if !checkScales(baseScale, exponentScale, resultScale) {
		return 0, errors.New("invalid scale")
	}
	b := decimal.WithContext(decimal.Context128).SetMantScale(base, baseScale)
	e := decimal.WithContext(decimal.Context128).SetMantScale(exponent, exponentScale)
	r, err := log(b, e, resultScale)
	if err != nil {
		return 0, err
	}
	return convertToIntResult(r, resultScale, mode)
}

func LogBigInt(base, exponent *big.Int, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (*big.Int, error) {
	if !checkScalesBigInt(baseScale, exponentScale, resultScale) {
		return nil, errors.New("invalid scale")
	}
	b := decimal.WithContext(decimal.Context128).SetBigMantScale(base, baseScale)
	e := decimal.WithContext(decimal.Context128).SetBigMantScale(exponent, exponentScale)
	r, err := log(b, e, resultScale)
	if err != nil {
		return nil, err
	}
	return convertToBigIntResult(r, resultScale, mode)
}

func ModDivision(x, y int64) int64 {
	return x - FloorDiv(x, y)*y
}

func FloorDiv(x, y int64) int64 {
	r := x / y
	// if the signs are different and modulo not zero, round down
	if (x^y) < 0 && (r*y != x) {
		r--
	}
	return r
}

func FloorDivBigInt(x, y *big.Int) *big.Int {
	var r *big.Int
	if x.Sign() == y.Sign() {
		if x.Cmp(y) < 0 {
			// abs(y-x)/2 + x
			r = y.Sub(y, x)
			r = r.Abs(r)
			r = r.Div(r, big.NewInt(2))
			r = r.Add(r, x)
			return r
		}
		// abs(x-y)/2 + y
		r = x.Sub(x, y)
		r = r.Abs(r)
		r = r.Div(r, big.NewInt(2))
		r = r.Add(r, y)
		return r
	}
	d := x.Add(x, y)
	two := big.NewInt(2)
	zero := big.NewInt(0)
	d2 := big.NewInt(0).Mod(d, two)
	if d.Cmp(zero) >= 0 || d2.Cmp(zero) == 0 {
		r = d.Div(d, two)
		return r
	}
	r = d.Sub(d, big.NewInt(1))
	r = r.Div(r, two)
	return r
}
