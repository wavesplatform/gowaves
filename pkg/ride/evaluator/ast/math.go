package ast

import (
	"github.com/ericlagergren/decimal"
	"github.com/ericlagergren/decimal/math"
	"github.com/pkg/errors"
)

var (
	zero = decimal.New(0, 0)
	one  = decimal.New(1, 0)
	ten  = decimal.New(10, 0)
)

func convertToResult(v *decimal.Big, scale int, mode decimal.RoundingMode) (int64, error) {
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

func pow(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if baseScale < 0 || baseScale > 8 ||
		exponentScale < 0 || exponentScale > 8 ||
		resultScale < 0 || resultScale > 8 {
		return 0, errors.New("pow: invalid scale")
	}
	b := decimal.WithContext(decimal.Context128).SetMantScale(base, baseScale)
	e := decimal.WithContext(decimal.Context128).SetMantScale(exponent, exponentScale)
	if b.IsInt() && e.Cmp(zero) == 0 {
		res, err := convertToResult(one, resultScale, mode)
		if err != nil {
			return 0, errors.Wrap(err, "pow")
		}
		return res, nil
	}
	r := decimal.WithContext(decimal.Context128)
	r = math.Pow(r, b, e)
	if r.Context.Err() != nil {
		return 0, errors.Errorf("pow: %s", r.Context.Conditions.Error())
	}
	res, err := convertToResult(r, resultScale, mode)
	if err != nil {
		return 0, errors.Wrap(err, "pow")
	}
	return res, nil
}

func fraction(value, numerator, denominator int64) (int64, error) {
	v := decimal.WithContext(decimal.Context128).SetMantScale(value, 0)
	n := decimal.WithContext(decimal.Context128).SetMantScale(numerator, 0)
	d := decimal.WithContext(decimal.Context128).SetMantScale(denominator, 0)

	v.Mul(v, n)
	v.Quo(v, d)
	if err := v.Context.Err(); err != nil {
		return 0, errors.Wrap(err, "fraction")
	}
	res, err := convertToResult(v, 0, decimal.ToZero)
	if err != nil {
		return 0, errors.Wrap(err, "fraction")
	}
	return res, nil
}

func log(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if baseScale < 0 || baseScale > 8 ||
		exponentScale < 0 || exponentScale > 8 ||
		resultScale < 0 || resultScale > 8 {
		return 0, errors.New("log: invalid scale")
	}
	b := decimal.WithContext(decimal.Context128).SetMantScale(base, baseScale)
	e := decimal.WithContext(decimal.Context128).SetMantScale(exponent, exponentScale)
	r := decimal.WithContext(decimal.Context128).SetMantScale(0, resultScale)
	bl := decimal.WithContext(decimal.Context128)
	math.Log(bl, b)
	if bl.Context.Err() != nil {
		return 0, errors.New(bl.Context.Conditions.Error())
	}
	el := decimal.WithContext(decimal.Context128)
	math.Log(el, e)
	if el.Context.Err() != nil {
		return 0, errors.New(el.Context.Conditions.Error())
	}
	r.Quo(bl, el)
	if r.Context.Err() != nil {
		return 0, errors.New(r.Context.Conditions.Error())
	}
	res, err := convertToResult(r, resultScale, mode)
	if err != nil {
		return 0, errors.Wrap(err, "log")
	}
	return res, nil
}

func modDivision(x int64, y int64) int64 {
	return x - floorDiv(x, y)*y
}

func floorDiv(x int64, y int64) int64 {
	r := x / y
	// if the signs are different and modulo not zero, round down
	if (x^y) < 0 && (r*y != x) {
		r--
	}
	return r
}

func mathLong(funcName string, f func(int64, int64) (Expr, error), s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	first, ok := rs[0].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s first argument expected to be *LongExpr, got %T", funcName, rs[0])
	}
	second, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s second argument expected to be *LongExpr, got %T", funcName, rs[1])
	}
	return f(first.Value, second.Value)
}

func roundingMode(e Expr) (decimal.RoundingMode, error) {
	switch e.InstanceOf() {
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
		return 0, errors.Errorf("unsupported rounding mode %s", e.InstanceOf())
	}
}
