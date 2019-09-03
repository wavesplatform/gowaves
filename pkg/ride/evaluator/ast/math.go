package ast

import (
	"github.com/ericlagergren/decimal"
	"github.com/ericlagergren/decimal/math"
	"github.com/pkg/errors"
)

var ten = decimal.New(10, 0)

// rescale changes the scale of given decimal value. While rescaling the value is rounded according to context of v.
func rescale(v *decimal.Big, scale int64) *decimal.Big {
	s := decimal.New(scale, 0)
	m := new(decimal.Big)
	math.Pow(m, ten, s)
	v.Mul(v, m)
	return v
}

func pow(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if baseScale < 0 || baseScale > 8 ||
		exponentScale < 0 || exponentScale > 8 ||
		resultScale < 0 || resultScale > 8 {
		return 0, errors.New("pow: invalid scale")
	}
	b := decimal.New(base, baseScale)
	e := decimal.New(exponent, exponentScale)
	r := decimal.New(0, resultScale)
	r.Context.RoundingMode = mode
	math.Pow(r, b, e)
	if r.Context.Err() != nil {
		return 0, errors.Errorf("pow: %s", r.Context.Conditions.Error())
	}
	rescale(r, int64(resultScale))
	i, ok := r.RoundToInt().Int64()
	if !ok {
		return 0, errors.New("pow: result out of int64 range")
	}
	return i, nil
}

func log(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error) {
	if baseScale < 0 || baseScale > 8 ||
		exponentScale < 0 || exponentScale > 8 ||
		resultScale < 0 || resultScale > 8 {
		return 0, errors.New("pow: invalid scale")
	}
	b := decimal.New(base, baseScale)
	e := decimal.New(exponent, exponentScale)
	r := decimal.New(0, resultScale)
	r.Context.RoundingMode = mode
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
	rescale(r, int64(resultScale))
	i, ok := r.RoundToInt().Int64()
	if !ok {
		return 0, errors.New("pow: result out of int64 range")
	}
	return i, nil
}

func modDivision(x int64, y int64) int64 {
	return x - floorDiv(x, y)*y
}

func floorDiv(x int64, y int64) int64 {
	r := x / y
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
		// TODO: Enable this branch after PR https://github.com/ericlagergren/decimal/pull/136 is accepted. Before that this using this rounding mode will panic.
		// TODO: return decimal.ToNearestToZero, nil
		panic("not implemented rounding mode")
	default:
		return 0, errors.Errorf("unsupported rounding mode %s", e.InstanceOf())
	}
}
