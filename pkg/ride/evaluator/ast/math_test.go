package ast

import (
	"fmt"
	"testing"

	"github.com/ericlagergren/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFraction(t *testing.T) {
	tests := []struct {
		value       int64
		numerator   int64
		denominator int64
		error       bool
		expected    int64
	}{
		{-6, 6301369, 100, false, -378082},
		{6, 6301369, 100, false, 378082},
		{6, 6301369, 0, true, 0},
	}
	for i, tc := range tests {
		r, err := fraction(tc.value, tc.numerator, tc.denominator)
		if tc.error {
			assert.Error(t, err, i)
			fmt.Println(err)
			continue
		}
		assert.NoError(t, err, i)
		assert.Equal(t, tc.expected, r, i)
	}
}

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
		{0, 0, 0, 0, 0, decimal.ToNearestAway, false, 1},
		{0, 8, 0, 8, 8, decimal.ToNearestAway, false, 100000000},
	}
	for i, tc := range tests {
		r, err := pow(tc.base, tc.exponent, tc.basePrecision, tc.exponentPrecision, tc.resultPrecision, tc.mode)
		if tc.error {
			assert.Error(t, err, i)
			continue
		}
		assert.NoError(t, err, i)
		assert.Equal(t, tc.expected, r, i)
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

func TestTestNetDApp_HGT44HrsSSD5cjANV6wtWNB9VKS3y7hhoNXEDWB56Lu9(t *testing.T) {
	//{-# STDLIB_VERSION 3 #-}
	//{-# SCRIPT_TYPE ACCOUNT #-}
	//{-# CONTENT_TYPE DAPP #-}
	//let FACTOR = 100000000
	//
	//let FACTORDECIMALS = 8
	//
	//let E = 271828182
	//
	//@Callable(i)
	//func coxRossRubinsteinCall (T,S,K,r,sigma,n) = {
	//	let deltaT = fraction(T, FACTOR, (365 * n))
	//	let sqrtDeltaT = pow(deltaT, FACTORDECIMALS, 5, 1, FACTORDECIMALS, HALFUP)
	//	let up = pow(E, FACTORDECIMALS, fraction(sigma, sqrtDeltaT, 100), FACTORDECIMALS, FACTORDECIMALS, HALFUP)
	//	let down = fraction(1, (FACTOR * FACTOR), up)
	//	let df = pow(E, FACTORDECIMALS, fraction(-(r), deltaT, 100), FACTORDECIMALS, FACTORDECIMALS, HALFUP)
	//	let pUp = fraction((pow(E, FACTORDECIMALS, fraction(r, deltaT, 100), FACTORDECIMALS, FACTORDECIMALS, HALFUP) - down), FACTOR, (up - down))
	//	let pDown = (FACTOR - pUp)
	//	let firstProjectedPrice = ((S * pow(fraction(up, 1, FACTOR), FACTORDECIMALS, 4, 0, FACTORDECIMALS, HALFUP)) * pow(fraction(down, 1, FACTOR), FACTORDECIMALS, 0, 0, FACTORDECIMALS, HALFUP))
	//	WriteSet([DataEntry("deltaT", deltaT), DataEntry("sqrtDeltaT", sqrtDeltaT), DataEntry("up", up), DataEntry("down", down), DataEntry("df", df), DataEntry("pUp", pUp), DataEntry("pDown", pDown), DataEntry("firstProjectedPrice", firstProjectedPrice)])
	//}

	var factor int64 = 100000000
	factorDecimals := 8
	var e int64 = 271828182

	//Call parameters
	var T int64 = 92
	var S int64 = 1000
	var r int64 = 6
	var sigma int64 = 20
	var n int64 = 4

	deltaT, err := fraction(T, factor, int64(365*n))
	require.NoError(t, err)
	assert.Equal(t, 6301369, int(deltaT))

	sqrtDeltaT, err := pow(deltaT, 5, factorDecimals, 1, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	assert.Equal(t, 25102528, int(sqrtDeltaT))

	p, err := fraction(sigma, sqrtDeltaT, 100)
	require.NoError(t, err)
	up, err := pow(e, p, factorDecimals, factorDecimals, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	assert.Equal(t, 105148668, int(up))

	down, err := fraction(1, factor*factor, up)
	require.NoError(t, err)
	assert.Equal(t, 95103439, int(down))

	p, err = fraction(-r, deltaT, 100)
	require.NoError(t, err)
	assert.Equal(t, -378082, int(p))
	df, err := pow(e, p, factorDecimals, factorDecimals, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	assert.True(t, df > 0)
	assert.Equal(t, 99622632, int(df))

	p, err = fraction(r, deltaT, 100)
	require.NoError(t, err)
	p0, err := pow(e, p, factorDecimals, factorDecimals, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	pUp, err := fraction(p0-down, factor, up-down)
	require.NoError(t, err)
	assert.Equal(t, 52516065, int(pUp))

	pDown := factor - pUp
	assert.Equal(t, 47483935, int(pDown))

	a0, err := fraction(up, 1, factor)
	require.NoError(t, err)
	a1, err := pow(a0, 4, factorDecimals, 0, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	a2, err := fraction(down, 1, factor)
	require.NoError(t, err)
	a3, err := pow(a2, 0, factorDecimals, 0, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	firstProjectedPrice := S * a1 * a3
	assert.Equal(t, 0, int(firstProjectedPrice))
}
