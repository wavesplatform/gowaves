package ast

import (
	"testing"

	"github.com/ericlagergren/decimal"
	"github.com/stretchr/testify/assert"
)

func TestPow(t *testing.T) {
	tests := []struct {
		base              int64
		basePrecision     int
		exponent          int64
		exponentPrecision int
		resultPrecision   int
		mode              decimal.RoundingMode
		error             bool
		expected          int64
	}{
		{12, 1, 3456, 3, 2, decimal.ToZero, false, 187},
		{12, 1, 3456, 3, 2, decimal.AwayFromZero, false, 188},
		{0, 1, 3456, 3, 2, decimal.AwayFromZero, false, 0},
		{20, 1, -1, 0, 4, decimal.ToZero, false, 5000},
		{-20, 1, -1, 0, 4, decimal.ToZero, false, -5000},
		{0, 1, -1, 0, 4, decimal.ToZero, true, 0},
		{0, 9, -1, 0, 2, decimal.ToZero, true, 0},
		{0, 1, -1, 9, 2, decimal.ToZero, true, 0},
		{0, 1, -1, 0, 9, decimal.ToZero, true, 0},
		{0, -1, -1, 0, 4, decimal.ToZero, true, 0},
		{0, 1, -1, -1, 4, decimal.ToZero, true, 0},
		{0, 1, -1, 0, -4, decimal.ToZero, true, 0},
	}
	for _, tc := range tests {
		r, err := pow(tc.base, tc.exponent, tc.basePrecision, tc.exponentPrecision, tc.resultPrecision, tc.mode)
		if tc.error {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, r)
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		base              int64
		basePrecision     int
		exponent          int64
		exponentPrecision int
		resultPrecision   int
		mode              decimal.RoundingMode
		error             bool
		expected          int64
	}{
		{16, 0, 2, 0, 0, decimal.ToPositiveInf, false, 4},
		{100, 0, 10, 0, 0, decimal.ToPositiveInf, false, 2},
		{16, 0, -2, 0, 0, decimal.ToPositiveInf, true, 0},
		{-16, 0, 2, 0, 0, decimal.ToPositiveInf, true, 0},
		{16, 9, 2, 0, 0, decimal.ToPositiveInf, true, 0},
		{16, 0, 2, 9, 0, decimal.ToPositiveInf, true, 0},
		{16, 0, 2, 0, 9, decimal.ToPositiveInf, true, 0},
		{16, -1, 2, 0, 0, decimal.ToPositiveInf, true, 0},
		{16, 0, 2, -1, 0, decimal.ToPositiveInf, true, 0},
		{16, 0, 2, 0, -1, decimal.ToPositiveInf, true, 0},
	}
	for _, tc := range tests {
		r, err := log(tc.base, tc.exponent, tc.basePrecision, tc.exponentPrecision, tc.resultPrecision, tc.mode)
		if tc.error {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, r)
	}
}
