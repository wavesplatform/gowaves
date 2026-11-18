package ride //nolint:testpackage // White-box tests of the unexported numeric helpers.

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/ericlagergren/decimal"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rideMath "github.com/wavesplatform/gowaves/pkg/ride/math"
)

func allRoundingModes() []decimal.RoundingMode {
	return []decimal.RoundingMode{
		decimal.ToZero, decimal.AwayFromZero, decimal.ToNearestAway, decimal.ToNearestTowardZero,
		decimal.ToPositiveInf, decimal.ToNegativeInf, decimal.ToNearestEven,
	}
}

// referenceFractionBigIntLikeInScala is the original implementation of fractionBigIntLikeInScala,
// it's used to check that the optimized version gives the same results.
func referenceFractionBigIntLikeInScala(v, n, d *big.Int, roundingMode decimal.RoundingMode) (*big.Int, error) {
	v, n, d = new(big.Int).Set(v), new(big.Int).Set(n), new(big.Int).Set(d)
	zero := big.NewInt(0)
	if d.Cmp(zero) == 0 {
		return nil, errors.New("division by zero")
	}
	p := v.Mul(v, n)
	s := big.NewInt(int64(p.Sign() * d.Sign()))
	pa := p.Abs(p)
	da := d.Abs(d)
	r, m := pa.QuoRem(pa, da, big.NewInt(0))
	ms := big.NewInt(int64(m.Sign()))
	x := new(big.Int).Sub(da, new(big.Int).Mul(m, big.NewInt(2)))
	switch roundingMode {
	case decimal.ToZero:
	case decimal.AwayFromZero:
		r = r.Add(r, ms)
	case decimal.ToNearestAway:
		if x.Cmp(zero) <= 0 {
			r = r.Add(r, big.NewInt(1))
		}
	case decimal.ToNearestTowardZero:
		if x.Cmp(zero) < 0 {
			r = r.Add(r, big.NewInt(1))
		}
	case decimal.ToPositiveInf:
		if s.Cmp(zero) > 0 {
			r = r.Add(r, ms)
		}
	case decimal.ToNegativeInf:
		if s.Cmp(zero) < 0 {
			r = r.Add(r, ms)
		}
	case decimal.ToNearestEven:
		switch x.Cmp(zero) {
		case -1:
			r = r.Add(r, big.NewInt(1))
		case 0:
			r = r.Add(r, new(big.Int).Mod(r, big.NewInt(2)))
		}
	default:
		return nil, errors.New("unsupported rounding mode")
	}
	return r.Mul(r, s), nil
}

func int64EdgeValues() []int64 {
	return []int64{
		0, 1, -1, 2, -2, 3, -3, 5, -5, 7, -7, 10, -10, 100, -100,
		math.MaxInt32, math.MinInt32, 1 << 32, -(1 << 32), 3037000499, -3037000499, 3037000500, -3037000500,
		math.MaxInt64 / 2, math.MinInt64 / 2, math.MaxInt64 - 1, math.MinInt64 + 1, math.MaxInt64, math.MinInt64,
	}
}

func randomInt64(r *rand.Rand) int64 {
	switch r.IntN(4) {
	case 0:
		return r.Int64N(21) - 10
	case 1:
		return r.Int64N(2_000_001) - 1_000_000
	case 2:
		return int64(r.Uint64() >> r.UintN(64))
	default:
		return -int64(r.Uint64() >> r.UintN(64))
	}
}

func TestFractionBigIntLikeInScalaMatchesReference(t *testing.T) {
	check := func(v, n, d *big.Int, mode decimal.RoundingMode) {
		vc, nc, dc := new(big.Int).Set(v), new(big.Int).Set(n), new(big.Int).Set(d)
		expected, expErr := referenceFractionBigIntLikeInScala(v, n, d, mode)
		actual, err := fractionBigIntLikeInScala(v, n, d, mode)
		if expErr != nil {
			require.EqualError(t, err, expErr.Error())
		} else {
			require.NoError(t, err)
			require.Zero(t, expected.Cmp(actual), "%s * %s / %s (%v): %s != %s", v, n, d, mode, expected, actual)
		}
		// Arguments must not be modified.
		require.Zero(t, vc.Cmp(v))
		require.Zero(t, nc.Cmp(n))
		require.Zero(t, dc.Cmp(d))
	}
	edges := int64EdgeValues()
	for _, mode := range allRoundingModes() {
		for _, v := range edges {
			for _, n := range edges {
				for _, d := range edges {
					check(big.NewInt(v), big.NewInt(n), big.NewInt(d), mode)
				}
			}
		}
		check(rideMath.MaxBigInt, rideMath.MaxBigInt, rideMath.MinBigInt, mode)
		check(rideMath.MinBigInt, big.NewInt(3), big.NewInt(-2), mode)
	}
	_, err := fractionBigIntLikeInScala(big.NewInt(1), big.NewInt(1), big.NewInt(1), decimal.RoundingMode(255))
	assert.EqualError(t, err, "unsupported rounding mode")
}

func TestFractionInt64LikeInScalaMatchesBigInt(t *testing.T) {
	check := func(v, n, d int64, mode decimal.RoundingMode) {
		r, ok := fractionInt64LikeInScala(v, n, d, mode)
		expected, err := fractionBigIntLikeInScala(big.NewInt(v), big.NewInt(n), big.NewInt(d), mode)
		if err != nil || !expected.IsInt64() {
			require.False(t, ok, "%d * %d / %d (%v)", v, n, d, mode)
			return
		}
		if _, fits := rideMath.MulInt64(v, n); !fits {
			require.False(t, ok, "%d * %d / %d (%v)", v, n, d, mode)
			return
		}
		require.True(t, ok, "%d * %d / %d (%v)", v, n, d, mode)
		require.Equal(t, expected.Int64(), r, "%d * %d / %d (%v)", v, n, d, mode)
	}
	edges := int64EdgeValues()
	rnd := rand.New(rand.NewPCG(1, 2))
	for _, mode := range allRoundingModes() {
		for _, v := range edges {
			for _, n := range edges {
				for _, d := range edges {
					check(v, n, d, mode)
				}
			}
		}
		for range 20000 {
			check(randomInt64(rnd), randomInt64(rnd), randomInt64(rnd), mode)
		}
	}
	_, ok := fractionInt64LikeInScala(1, 1, 1, decimal.RoundingMode(255))
	assert.False(t, ok)
}

func TestFractionIntRoundsEdgeCases(t *testing.T) {
	for _, test := range []struct {
		v, n, d int64
		round   rideType
		r       int64
		err     string
	}{
		{5, 1, 2, newHalfEven(nil), 2, ""},
		{7, 1, 2, newHalfEven(nil), 4, ""},
		{-5, 1, 2, newHalfEven(nil), -2, ""},
		{-5, 1, 2, newHalfUp(nil), -3, ""},
		{-5, 1, 2, newHalfDown(nil), -2, ""},
		{-5, 1, 2, newCeiling(nil), -2, ""},
		{-5, 1, 2, newFloor(nil), -3, ""},
		{-5, 1, 2, newDown(nil), -2, ""},
		{-5, 1, 2, newUp(nil), -3, ""},
		{0, 1, -2, newUp(nil), 0, ""},
		{math.MinInt64, 1, 1, newDown(nil), math.MinInt64, ""},
		{math.MinInt64, -1, -1, newDown(nil), math.MinInt64, ""},
		{math.MaxInt64, math.MaxInt64, math.MaxInt64, newDown(nil), math.MaxInt64, ""},
		{math.MinInt64, math.MinInt64, math.MinInt64, newDown(nil), math.MinInt64, ""},
		{math.MaxInt64, 3, 2, newDown(nil), 0, "fraction: result is out of int64 range"},
		{math.MinInt64, 1, -1, newDown(nil), 0, "fraction: result is out of int64 range"},
		{math.MinInt64, 1, 0, newDown(nil), 0, "fraction: division by zero"},
		{1, 1, 0, newHalfEven(nil), 0, "fraction: division by zero"},
	} {
		r, err := fractionIntRounds(nil, rideInt(test.v), rideInt(test.n), rideInt(test.d), test.round)
		if test.err != "" {
			assert.EqualError(t, err, test.err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, rideInt(test.r), r, "%d * %d / %d (%s)", test.v, test.n, test.d, test.round)
	}
}

func TestFractionIntEdgeCases(t *testing.T) {
	for _, test := range []struct {
		v, n, d int64
		r       int64
		fail    bool
	}{
		{7, 1, 2, 3, false},
		{-7, 1, 2, -3, false},
		{7, -1, -2, 3, false},
		{math.MinInt64, 1, 1, math.MinInt64, false},
		{math.MinInt64, 1, -1, 0, true},
		{math.MaxInt64, math.MaxInt64, math.MaxInt64, math.MaxInt64, false},
		{math.MaxInt64, 3, 2, 0, true},
		{1, 1, 0, 0, true},
	} {
		r, err := fraction(nil, rideInt(test.v), rideInt(test.n), rideInt(test.d))
		if test.fail {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, rideInt(test.r), r)
	}
	_, err := fraction(nil, rideInt(1), rideInt(2))
	assert.EqualError(t, err, "fraction: 2 is invalid number of arguments, expected 3")
	_, err = fraction(nil, rideInt(1), nil, rideInt(2))
	assert.EqualError(t, err, "fraction: argument 2 is empty")
	_, err = fraction(nil, rideInt(1), rideInt(2), rideString("3"))
	assert.EqualError(t, err, "fraction: argument 3 is not of type 'Int' but 'String'")
}

func TestBigIntFunctionsDoNotModifyArguments(t *testing.T) {
	values := func() []rideType {
		return []rideType{
			rideBigInt{v: big.NewInt(-12345)}, rideBigInt{v: big.NewInt(678)}, rideBigInt{v: big.NewInt(9)},
		}
	}
	maxCopy, minCopy := new(big.Int).Set(rideMath.MaxBigInt), new(big.Int).Set(rideMath.MinBigInt)
	for _, test := range []struct {
		name string
		fn   rideFunction
		args func(a, b, c rideType) []rideType
	}{
		{"sumBigInt", sumBigInt, func(a, b, _ rideType) []rideType { return []rideType{a, b} }},
		{"subtractBigInt", subtractBigInt, func(a, b, _ rideType) []rideType { return []rideType{a, b} }},
		{"multiplyBigInt", multiplyBigInt, func(a, b, _ rideType) []rideType { return []rideType{a, b} }},
		{"divideBigInt", divideBigInt, func(a, b, _ rideType) []rideType { return []rideType{a, b} }},
		{"moduloBigInt", moduloBigInt, func(a, b, _ rideType) []rideType { return []rideType{a, b} }},
		{"gtBigInt", gtBigInt, func(a, b, _ rideType) []rideType { return []rideType{a, b} }},
		{"geBigInt", geBigInt, func(a, b, _ rideType) []rideType { return []rideType{a, b} }},
		{"fractionBigInt", fractionBigInt, func(a, b, c rideType) []rideType { return []rideType{a, b, c} }},
		{"fractionBigIntRounds", fractionBigIntRounds, func(a, b, c rideType) []rideType {
			return []rideType{a, b, c, newHalfEven(nil)}
		}},
		{"unaryMinusBigInt", unaryMinusBigInt, func(a, _, _ rideType) []rideType { return []rideType{a} }},
		{"bigIntToBytes", bigIntToBytes, func(a, _, _ rideType) []rideType { return []rideType{a} }},
		{"bigIntToString", bigIntToString, func(a, _, _ rideType) []rideType { return []rideType{a} }},
		{"bigIntToInt", bigIntToInt, func(a, _, _ rideType) []rideType { return []rideType{a} }},
		{"powBigInt", powBigInt, func(_, b, c rideType) []rideType {
			return []rideType{b, rideInt(0), c, rideInt(0), rideInt(0), newHalfEven(nil)}
		}},
		{"logBigInt", logBigInt, func(_, b, c rideType) []rideType {
			return []rideType{b, rideInt(0), c, rideInt(0), rideInt(4), newHalfEven(nil)}
		}},
		{"sqrtBigInt", sqrtBigInt, func(_, b, _ rideType) []rideType {
			return []rideType{b, rideInt(0), rideInt(4), newHalfEven(nil)}
		}},
		{"maxListBigInt", maxListBigInt, func(a, b, c rideType) []rideType { return []rideType{rideList{a, b, c}} }},
		{"minListBigInt", minListBigInt, func(a, b, c rideType) []rideType { return []rideType{rideList{a, b, c}} }},
		{"medianListBigInt/odd", medianListBigInt, func(a, b, c rideType) []rideType {
			return []rideType{rideList{a, b, c}}
		}},
		{"medianListBigInt/even", medianListBigInt, func(a, b, c rideType) []rideType {
			return []rideType{rideList{a, b, c, rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MinBigInt}, c}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			vs := values()
			a, b, c := vs[0], vs[1], vs[2]
			_, err := test.fn(nil, test.args(a, b, c)...)
			require.NoError(t, err)
			expected := values()
			for i := range vs {
				assert.True(t, expected[i].eq(vs[i]), "argument %d modified: %s", i+1, vs[i])
			}
			assert.Zero(t, maxCopy.Cmp(rideMath.MaxBigInt))
			assert.Zero(t, minCopy.Cmp(rideMath.MinBigInt))
		})
	}
}

func TestBigIntListFunctionsEdgeCases(t *testing.T) {
	maxV, minV := rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MinBigInt}
	single := rideList{toRideBigInt(-7)}
	for _, test := range []struct {
		name string
		fn   rideFunction
		list rideList
		r    rideType
		err  string
	}{
		{"max/single", maxListBigInt, single, toRideBigInt(-7), ""},
		{"min/single", minListBigInt, single, toRideBigInt(-7), ""},
		{"max/limits", maxListBigInt, rideList{minV, toRideBigInt(0), maxV}, maxV, ""},
		{"min/limits", minListBigInt, rideList{maxV, toRideBigInt(0), minV}, minV, ""},
		{"max/duplicates", maxListBigInt, rideList{toRideBigInt(3), toRideBigInt(3), toRideBigInt(-3)}, toRideBigInt(3), ""},
		{"min/negative", minListBigInt, rideList{toRideBigInt(-1), toRideBigInt(-3), toRideBigInt(-2)}, toRideBigInt(-3), ""},
		{"max/empty", maxListBigInt, rideList{}, nil, "maxListBigInt: invalid list size 0"},
		{"min/empty", minListBigInt, rideList{}, nil, "minListBigInt: invalid list size 0"},
		{"max/wrong type", maxListBigInt, rideList{toRideBigInt(1), rideInt(2)}, nil,
			"maxListBigInt: unexpected type of list element 'Int'"},
		{"min/wrong first", minListBigInt, rideList{rideString("x"), toRideBigInt(1)}, nil,
			"minListBigInt: unexpected type of list element 'String'"},
		{"median/even negative", medianListBigInt,
			rideList{toRideBigInt(-4), toRideBigInt(-1), toRideBigInt(-3), toRideBigInt(-2)},
			toRideBigInt(-2), ""},
		{"median/wrong type", medianListBigInt, rideList{toRideBigInt(1), rideInt(2)}, nil,
			"medianListBigInt: unexpected type of list element 'Int'"},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, err := test.fn(nil, test.list)
			if test.err != "" {
				assert.EqualError(t, err, test.err)
				return
			}
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), "%s != %s", test.r, r)
		})
	}
}

func TestBigIntArithmeticErrors(t *testing.T) {
	maxV, minV := rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MinBigInt}
	for _, test := range []struct {
		fn   rideFunction
		args []rideType
		err  string
	}{
		{sumBigInt, []rideType{maxV, toRideBigInt(1)},
			"sumBigInt: " + new(big.Int).Add(rideMath.MaxBigInt, big.NewInt(1)).String() +
				" result is out of range"},
		{subtractBigInt, []rideType{minV, toRideBigInt(1)},
			"subtractBigInt: " + new(big.Int).Sub(rideMath.MinBigInt, big.NewInt(1)).String() +
				" result is out of range"},
		{divideBigInt, []rideType{toRideBigInt(1), toRideBigInt(0)}, "divideBigInt: division by zero"},
		{moduloBigInt, []rideType{toRideBigInt(1), toRideBigInt(0)}, "moduloBigInt: division by zero"},
		{fractionBigInt, []rideType{toRideBigInt(1), toRideBigInt(1), toRideBigInt(0)}, "fractionBigInt: division by zero"},
		{fractionBigIntRounds, []rideType{toRideBigInt(1), toRideBigInt(1), toRideBigInt(0), newDown(nil)},
			"fractionBigIntRounds: division by zero"},
		{unaryMinusBigInt, []rideType{minV}, "unaryMinusBigInt: positive BigInt overflow"},
		{divideBigInt, []rideType{minV, toRideBigInt(-1)}, "divideBigInt: " + new(big.Int).Neg(rideMath.MinBigInt).String() +
			" result is out of range"},
	} {
		_, err := test.fn(nil, test.args...)
		assert.EqualError(t, err, test.err)
	}
	r, err := unaryMinusBigInt(nil, maxV)
	require.NoError(t, err)
	assert.True(t, rideBigInt{v: new(big.Int).Neg(rideMath.MaxBigInt)}.eq(r))
	r, err = moduloBigInt(nil, toRideBigInt(-7), toRideBigInt(3))
	require.NoError(t, err)
	assert.True(t, toRideBigInt(-1).eq(r))
	r, err = divideBigInt(nil, toRideBigInt(-7), toRideBigInt(2))
	require.NoError(t, err)
	assert.True(t, toRideBigInt(-3).eq(r))
}
