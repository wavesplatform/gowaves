package math

import (
	math0 "math"
	"math/big"

	"github.com/ericlagergren/decimal"
	"github.com/ericlagergren/decimal/math"
	"github.com/pkg/errors"
)

const (
	maxIntScale    = 8
	maxBigIntScale = 18
)

var (
	longContext = decimal.Context{
		Precision:     19,
		RoundingMode:  decimal.ToNearestEven,
		OperatingMode: decimal.GDA,
		MaxScale:      6144,
		MinScale:      -6143,
	}
	bigIntContext = decimal.Context{
		Precision:     154,
		RoundingMode:  decimal.ToNearestEven,
		OperatingMode: decimal.GDA,
		MaxScale:      6144,
		MinScale:      -6143,
	}

	zero            = decimal.New(0, 0)
	one             = decimal.New(1, 0)
	pointFiveInt    = decimal.WithContext(longContext).SetMantScale(5, 1)
	pointFiveBigInt = decimal.WithContext(bigIntContext).SetMantScale(5, 1)
	OneBigInt       = big.NewInt(1)
	MinBigInt       = minBigInt()
	MaxBigInt       = maxBigInt()
)

func maxBigInt() *big.Int {
	max := big.NewInt(0)
	max = max.Exp(big.NewInt(2), big.NewInt(511), nil)
	max = max.Sub(max, OneBigInt)
	return max
}

func minBigInt() *big.Int {
	min := big.NewInt(0)
	min = min.Neg(maxBigInt())
	min = min.Sub(min, OneBigInt)
	return min
}

func checkScales(scales ...int) bool {
	// 8 is the maximum scale for RIDE Int values
	for _, s := range scales {
		if s < 0 || s > maxIntScale {
			return false
		}
	}
	return true
}

func checkScalesBigInt(scales ...int) bool {
	// 18 is the maximum scale for RIDE BigInt values
	for _, s := range scales {
		if s < 0 || s > maxBigIntScale {
			return false
		}
	}
	return true
}

func pow10(a int, context decimal.Context) *decimal.Big {
	ten := decimal.WithContext(context).SetMantScale(10, 0)
	return math.Pow(decimal.WithContext(context), ten, decimal.WithContext(context).SetMantScale(int64(a), 0))
}

// checkScale function performs check of BigDecimal's scale as in Java implementation
func checkScale(v int) error {
	if v > math0.MaxInt32 {
		return errors.New("scale underflow")
	}
	if v < math0.MinInt32 {
		return errors.New("scale overflow")
	}
	return nil
}

// rescale performs pre-rounding conversion of BigDecimal result to exclude unnecessary but heavy calculations
func rescale(value *decimal.Big, scale, precision int, context decimal.Context) (*decimal.Big, error) {
	s := value.Scale()
	if err := checkScale(s); err != nil {
		return nil, err
	}
	v := decimal.WithContext(context)
	v.Copy(value)
	v.SetScale(0)
	if s > scale {
		if s-scale > precision-1 {
			return zero, nil
		} else {
			return v.Quo(v, pow10(s-scale, context)), nil
		}
	} else {
		if scale-s > precision-1 {
			return nil, errors.New("value overflow")
		} else {
			return v.Mul(v, pow10(scale-s, context)), nil
		}
	}
}

func convertToIntResult(v *decimal.Big, scale int, mode decimal.RoundingMode) (int64, error) {
	context := decimal.Context128
	context.RoundingMode = mode
	r := decimal.WithContext(context).Set(v)
	r.Mul(r, pow10(scale, decimal.Context128))
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
	context := bigIntContext
	context.RoundingMode = mode
	r := decimal.WithContext(context).Set(v)
	r.Mul(r, pow10(scale, bigIntContext)) // r = v * 10^s
	return r.RoundToInt().Int(nil), nil
}

func pow(base, exponent *decimal.Big, context decimal.Context) (*decimal.Big, error) {
	if base.IsInt() && exponent.Cmp(zero) == 0 {
		return one, nil
	}
	r := decimal.WithContext(context)
	r = math.Pow(r, base, exponent)
	if r.Context.Err() != nil {
		return nil, errors.New(r.Context.Conditions.Error())
	}
	return r, nil
}

func PowV1(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if !checkScales(baseScale, exponentScale, resultScale) {
		return 0, errors.New("invalid scale")
	}
	b := decimal.WithContext(decimal.Context128).SetMantScale(base, baseScale)
	e := decimal.WithContext(decimal.Context128).SetMantScale(exponent, exponentScale)
	r, err := pow(b, e, decimal.Context128)
	if err != nil {
		return 0, err
	}
	return convertToIntResult(r, resultScale, mode)
}

func PowV2(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if !checkScales(baseScale, exponentScale, resultScale) {
		return 0, errors.New("invalid scale")
	}
	b := decimal.WithContext(longContext).SetMantScale(base, baseScale)
	e := decimal.WithContext(longContext).SetMantScale(exponent, exponentScale)
	r, err := pow(b, e, longContext)
	if err != nil {
		return 0, err
	}
	r, err = rescale(r, resultScale, longContext.Precision, longContext)
	if err != nil {
		return 0, err
	}
	context := longContext
	context.RoundingMode = mode
	r = decimal.WithContext(context).Set(r)
	res, ok := r.RoundToInt().Int64()
	if !ok {
		return 0, errors.New("result out of int64 range")
	}
	return res, nil
}

func PowBigInt(base, exponent *big.Int, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (*big.Int, error) {
	if !checkScalesBigInt(baseScale, exponentScale, resultScale) {
		return nil, errors.New("invalid scale")
	}
	b := decimal.WithContext(bigIntContext).SetBigMantScale(base, baseScale)
	e := decimal.WithContext(bigIntContext).SetBigMantScale(exponent, exponentScale)
	r, err := pow(b, e, bigIntContext)
	if err != nil {
		return nil, err
	}
	r, err = rescale(r, resultScale, bigIntContext.Precision, bigIntContext)
	if err != nil {
		return nil, err
	}
	if !r.IsNormal() {
		return nil, errors.New("not normal")
	}
	context := bigIntContext
	context.RoundingMode = mode
	r = decimal.WithContext(context).Set(r)
	return r.RoundToInt().Int(nil), nil
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
	r := decimal.WithContext(bigIntContext).SetMantScale(0, resultScale)
	bl := decimal.WithContext(bigIntContext)
	math.Log(bl, base)
	if bl.Context.Err() != nil {
		return nil, errors.New(bl.Context.Conditions.Error())
	}
	el := decimal.WithContext(bigIntContext)
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
	b := decimal.WithContext(bigIntContext).SetBigMantScale(base, baseScale)
	e := decimal.WithContext(bigIntContext).SetBigMantScale(exponent, exponentScale)
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

func Sqrt(number int64, numberScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if !checkScales(numberScale, resultScale) {
		return 0, errors.New("invalid scale")
	}
	n := decimal.WithContext(longContext).SetMantScale(number, numberScale)
	r, err := pow(n, pointFiveInt, longContext)
	if err != nil {
		return 0, err
	}
	r, err = rescale(r, resultScale, longContext.Precision, longContext)
	if err != nil {
		return 0, err
	}
	context := longContext
	context.RoundingMode = mode
	r = decimal.WithContext(context).Set(r)
	res, ok := r.RoundToInt().Int64()
	if !ok {
		return 0, errors.New("result out of int64 range")
	}
	return res, nil
}

func SqrtBigInt(number *big.Int, numberScale, resultScale int, mode decimal.RoundingMode) (*big.Int, error) {
	if !checkScalesBigInt(numberScale, resultScale) {
		return nil, errors.New("invalid scale")
	}
	b := decimal.WithContext(bigIntContext).SetBigMantScale(number, numberScale)
	r, err := pow(b, pointFiveBigInt, bigIntContext)
	if err != nil {
		return nil, err
	}
	r, err = rescale(r, resultScale, bigIntContext.Precision, bigIntContext)
	if err != nil {
		return nil, err
	}
	if !r.IsNormal() {
		return nil, errors.New("not normal")
	}
	context := bigIntContext
	context.RoundingMode = mode
	r = decimal.WithContext(context).Set(r)
	return r.RoundToInt().Int(nil), nil
}
