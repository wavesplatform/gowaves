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
