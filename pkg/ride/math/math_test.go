package math

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ericlagergren/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFraction(t *testing.T) {
	for i, tc := range []struct {
		value       int64
		numerator   int64
		denominator int64
		error       bool
		expected    int64
	}{
		{-6, 6301369, 100, false, -378082},
		{6, 6301369, 100, false, 378082},
		{6, 6301369, 0, true, 0},
	} {
		r, err := Fraction(tc.value, tc.numerator, tc.denominator)
		if tc.error {
			assert.Error(t, err, i)
			fmt.Println(err)
			continue
		}
		assert.NoError(t, err, i)
		assert.Equal(t, tc.expected, r, i)
	}
}

func TestPowV1(t *testing.T) {
	for i, tc := range []struct {
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
		{2, 0, 2, 0, 9, decimal.ToZero, true, 0},
		{2, -2, 2, 0, 5, decimal.ToZero, true, 0},
		{2, 0, 62, 0, 0, decimal.ToZero, false, 4611686018427387904},
		{2, 0, 63, 0, 0, decimal.ToZero, true, 0},
		{10, 0, -8, 0, 8, decimal.ToNearestAway, false, 1},
		{10, 0, -9, 0, 8, decimal.ToNearestAway, false, 0},
		{10, 6, 6, 0, 0, decimal.AwayFromZero, false, 1},
		{9, 0, 999999, 0, 0, decimal.ToNegativeInf, true, 0},
		{987654321, 8, 98765432101234, 8, 8, decimal.ToZero, true, 0},
		{98765432, 8, 145998765432, 8, 8, decimal.ToNearestAway, false, 1},
		{198765432, 8, 6298765432, 8, 0, decimal.ToZero, false, 6191427136334512235},
		{math.MaxInt64, 0, math.MaxInt64, 8, 8, decimal.ToZero, true, 0},
		{math.MaxInt64, 0, math.MaxInt64, 0, 0, decimal.ToZero, true, 0},
		{-math.MaxInt64, 0, math.MaxInt64, 0, 0, decimal.ToZero, true, 0},
		{math.MaxInt64, 0, -math.MaxInt64, 0, 8, decimal.ToZero, true, 0},
		{1, 8, -math.MaxInt64, 0, 8, decimal.ToZero, true, 0},
		{98765432, 8, -math.MaxInt64, 8, 8, decimal.ToZero, true, 0},
		{98765432, 8, -math.MaxInt64, 0, 8, decimal.ToZero, true, 0},
		{math.MaxInt64, 0, 5, 1, 8, decimal.ToZero, false, 303700049997604969},
		{math.MaxInt64, 8, 5, 1, 8, decimal.ToZero, false, 30370004999760},
	} {
		r, err := PowV1(tc.base, tc.exponent, tc.basePrecision, tc.exponentPrecision, tc.resultPrecision, tc.mode)
		if tc.error {
			assert.Error(t, err, i)
			continue
		}
		assert.NoError(t, err, i)
		assert.Equal(t, tc.expected, r, i)
	}
}

func TestPowV2(t *testing.T) {
	for i, tc := range []struct {
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
		{2, 0, 2, 0, 9, decimal.ToZero, true, 0},
		{2, -2, 2, 0, 5, decimal.ToZero, true, 0},
		{2, 0, 62, 0, 0, decimal.ToZero, false, 4611686018427387904},
		{2, 0, 63, 0, 0, decimal.ToZero, true, 0},
		{10, 0, -8, 0, 8, decimal.ToNearestAway, false, 1},
		{10, 0, -9, 0, 8, decimal.ToNearestAway, false, 0},
		{10, 6, 6, 0, 0, decimal.AwayFromZero, false, 0},
		{98765432, 8, math.MaxInt64, 8, 8, decimal.ToZero, false, 0},
		{987654, 8, 987654321, 0, 0, decimal.ToZero, false, 0},
		{9, 0, 999999, 0, 0, decimal.ToNegativeInf, true, 0},
		{987654321, 8, 98765432101234, 8, 8, decimal.ToZero, true, 0},
		{98765432, 8, 145998765432, 8, 8, decimal.ToNearestAway, false, 1},
		{198765432, 8, 6298765432, 8, 0, decimal.ToZero, false, 6191427136334512235},
		{math.MaxInt64, 0, math.MaxInt64, 8, 8, decimal.ToZero, true, 0},
		{math.MaxInt64, 0, math.MaxInt64, 0, 0, decimal.ToZero, true, 0},
		{-math.MaxInt64, 0, math.MaxInt64, 0, 0, decimal.ToZero, true, 0},
		{math.MaxInt64, 0, -math.MaxInt64, 0, 8, decimal.ToZero, true, 0},
		{1, 8, -math.MaxInt64, 0, 8, decimal.ToZero, true, 0},
		{98765432, 8, -math.MaxInt64, 8, 8, decimal.ToZero, true, 0},
		{98765432, 8, -math.MaxInt64, 0, 8, decimal.ToZero, true, 0},
		{98765432, 8, math.MaxInt64, 8, 8, decimal.ToZero, false, 0},
		{98765432, 8, math.MaxInt64, 0, 8, decimal.ToZero, true, 0},
		{math.MaxInt64, 0, 5, 1, 8, decimal.ToZero, false, 303700049997604969},
		{math.MaxInt64, 8, 5, 1, 8, decimal.ToZero, false, 30370004999760},
	} {
		r, err := PowV2(tc.base, tc.exponent, tc.basePrecision, tc.exponentPrecision, tc.resultPrecision, tc.mode)
		if tc.error {
			assert.Error(t, err, i)
			continue
		}
		assert.NoError(t, err, i)
		assert.Equal(t, tc.expected, r, i)
	}
}

func TestLog(t *testing.T) {
	for _, tc := range []struct {
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
	} {
		r, err := Log(tc.base, tc.exponent, tc.basePrecision, tc.exponentPrecision, tc.resultPrecision, tc.mode)
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

	deltaT, err := Fraction(T, factor, 365*n)
	require.NoError(t, err)
	assert.Equal(t, 6301369, int(deltaT))

	sqrtDeltaT, err := PowV1(deltaT, 5, factorDecimals, 1, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	assert.Equal(t, 25102528, int(sqrtDeltaT))

	p, err := Fraction(sigma, sqrtDeltaT, 100)
	require.NoError(t, err)
	up, err := PowV1(e, p, factorDecimals, factorDecimals, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	assert.Equal(t, 105148668, int(up))

	down, err := Fraction(1, factor*factor, up)
	require.NoError(t, err)
	assert.Equal(t, 95103439, int(down))

	p, err = Fraction(-r, deltaT, 100)
	require.NoError(t, err)
	assert.Equal(t, -378082, int(p))
	df, err := PowV1(e, p, factorDecimals, factorDecimals, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	assert.True(t, df > 0)
	assert.Equal(t, 99622632, int(df))

	p, err = Fraction(r, deltaT, 100)
	require.NoError(t, err)
	p0, err := PowV1(e, p, factorDecimals, factorDecimals, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	pUp, err := Fraction(p0-down, factor, up-down)
	require.NoError(t, err)
	assert.Equal(t, 52516065, int(pUp))

	pDown := factor - pUp
	assert.Equal(t, 47483935, int(pDown))

	a0, err := Fraction(up, 1, factor)
	require.NoError(t, err)
	a1, err := PowV1(a0, 4, factorDecimals, 0, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	a2, err := Fraction(down, 1, factor)
	require.NoError(t, err)
	a3, err := PowV1(a2, 0, factorDecimals, 0, factorDecimals, decimal.ToNearestAway /*HALFUP*/)
	require.NoError(t, err)
	firstProjectedPrice := S * a1 * a3
	assert.Equal(t, 0, int(firstProjectedPrice))
}

func BenchmarkPow(b *testing.B) {
	for i, p := range []struct {
		base int64
		exp  int64
		bs   int
		es   int
		rs   int
		rm   decimal.RoundingMode
	}{
		{math.MaxInt64, math.MaxInt64, 0, 8, 8, decimal.ToZero},
		{math.MaxInt64, math.MaxInt64, 0, 0, 0, decimal.ToZero},
		{-math.MaxInt64, math.MaxInt64, 0, 0, 0, decimal.ToZero},
		{math.MaxInt64, -math.MaxInt64, 0, 0, 0, decimal.ToZero},
		{1, -math.MaxInt64, 8, 0, 8, decimal.ToZero},
		{98765432, -math.MaxInt64, 8, 0, 8, decimal.ToZero},
		{98765432, math.MaxInt64, 8, 0, 8, decimal.ToZero},
		//{98765432, math.MaxInt64, 8, 8, 8, decimal.ToZero},
		{98765432, 145998765432, 8, 8, 8, decimal.ToNearestAway},
		{198765432, 6298765432, 8, 8, 0, decimal.ToZero},
	} {
		b.Run(fmt.Sprintf("%d", i+1), func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_, _ = PowV2(p.base, p.exp, p.bs, p.es, p.rs, p.rm)
			}
		})
	}
}

func TestPowInvariant(t *testing.T) {
	// https://wavesexplorer.com/tx/BWK74YoQRUpp7BqXCCahP4LSuy3Z7FJg4bteect1u6Qp
	// let digits8 = 8
	// let alpha = 50
	// let alphaDigits = 2
	// let beta = 46000000
	// let scale8 = 100000000
	// let scale12 = 1000000000000
	// let x = 2661956191736
	// let y = 2554192264270
	// let sk = (((fraction(scale12, x, y) + fraction(scale12, y, x)) / 2) / 10000)
	// (fraction((x + y), scale8, pow(sk, digits8, alpha, alphaDigits, digits8, CEILING)) + (2 * fraction(pow(fraction(x, y, scale8), 0, 5, 1, (digits8 / 2), DOWN), pow((sk - beta), digits8, alpha, alphaDigits, digits8, DOWN), scale8)))
	for _, tc := range []struct {
		f func(base, exponent int64, baseScale, exponentScale, resultScale int, mode decimal.RoundingMode) (int64, error)
		r int
	}{
		{PowV1, 9049204201489},
		{PowV2, 9049204201491},
	} {
		digits8 := 8
		alpha := int64(50)
		alphaDigits := 2
		beta := int64(46000000)
		scale8 := int64(100000000)
		scale12 := int64(1000000000000)
		x := int64(2661956191736)
		y := int64(2554192264270)

		fr1, err := Fraction(scale12, x, y)
		require.NoError(t, err)
		fr2, err := Fraction(scale12, y, x)
		require.NoError(t, err)

		a1 := FloorDiv(fr1+fr2, 2)
		sk := FloorDiv(a1, 10000)

		b1, err := tc.f(sk, alpha, digits8, alphaDigits, digits8, decimal.ToPositiveInf)
		require.NoError(t, err)
		r1, err := Fraction(x+y, scale8, b1)
		require.NoError(t, err)
		b2, err := Fraction(x, y, scale8)
		require.NoError(t, err)
		b3, err := tc.f(b2, 5, 0, 1, digits8/2, decimal.ToZero)
		require.NoError(t, err)
		b4, err := tc.f(sk-beta, alpha, digits8, alphaDigits, digits8, decimal.ToZero)
		require.NoError(t, err)
		r2, err := Fraction(b3, b4, scale8)
		require.NoError(t, err)
		r3 := int(r1 + 2*r2)
		assert.Equal(t, tc.r, r3)
	}
}

func fromString(t *testing.T, s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok)
	return v
}

func TestPowBigInt(t *testing.T) {
	d18 := fromString(t, "987654321012345678")
	d19 := fromString(t, "1987654321012345678")
	e1 := fromString(t, "3259987654320123456789")
	e2 := fromString(t, "515598765432101234567")
	e3 := func() *big.Int {
		v := big.NewInt(math.MaxInt64)
		for i := 0; i < 6; i++ {
			v = v.Mul(v, big.NewInt(math.MaxInt64))
		}
		v = v.Div(v, big.NewInt(4))
		m := new(big.Int).Set(MaxBigInt)
		r := m.Div(m, v)
		return r
	}
	r := fromString(t, "6670795527762621906375444802568692078004471712158714717165576501880318489264376534028344582079701518666593922923767238664173166263805614917588045354008642")

	for i, tc := range []struct {
		base              *big.Int
		basePrecision     int
		exponent          *big.Int
		exponentPrecision int
		resultPrecision   int
		mode              decimal.RoundingMode
		error             bool
		expected          *big.Int
	}{
		{MaxBigInt, 0, MaxBigInt, 18, 18, decimal.ToZero, true, nil},
		{MaxBigInt, 0, MaxBigInt, 0, 0, decimal.ToZero, true, nil},
		{MaxBigInt, 0, MaxBigInt, 0, 0, decimal.ToZero, true, nil},
		{MaxBigInt, 0, MinBigInt, 0, 18, decimal.ToZero, true, nil},
		{big.NewInt(1), 18, MinBigInt, 0, 18, decimal.ToZero, true, nil},
		{d18, 18, MinBigInt, 0, 18, decimal.ToZero, true, nil},
		{d18, 18, MaxBigInt, 0, 18, decimal.ToZero, true, nil},
		{d18, 18, e3(), 18, 18, decimal.ToZero, false, big.NewInt(0)},
		{d18, 18, e1, 18, 18, decimal.ToNearestAway, false, big.NewInt(3)},
		{d19, 18, e2, 18, 0, decimal.ToZero, false, r},
	} {
		r, err := PowBigInt(tc.base, tc.exponent, tc.basePrecision, tc.exponentPrecision, tc.resultPrecision, tc.mode)
		if tc.error {
			assert.Error(t, err, i)
			continue
		}
		assert.NoError(t, err, i)
		assert.Equal(t, tc.expected, r, i)
	}
}

func TestSqrt(t *testing.T) {
	for i, tc := range []struct {
		number          int64
		numberPrecision int
		resultPrecision int
		mode            decimal.RoundingMode
		error           bool
		expected        int64
	}{
		{math.MaxInt64, 0, 8, decimal.ToZero, false, 303700049997604969},
		{math.MaxInt64, 8, 8, decimal.ToZero, false, 30370004999760},
	} {
		r, err := Sqrt(tc.number, tc.numberPrecision, tc.resultPrecision, tc.mode)
		if tc.error {
			assert.Error(t, err, i)
			continue
		}
		assert.NoError(t, err, i)
		assert.Equal(t, tc.expected, r, i)
	}
}

func TestSqrtBigInt(t *testing.T) {
	r1 := fromString(t, "81877371507464127617551201542979628307507432471243237061821853600756754782485292915524036944801")
	r2 := fromString(t, "81877371507464127617551201542979628307507432471243237061821853600756754782485292915524")

	for i, tc := range []struct {
		number          *big.Int
		numberPrecision int
		resultPrecision int
		mode            decimal.RoundingMode
		error           bool
		expected        *big.Int
	}{
		{MaxBigInt, 0, 18, decimal.ToZero, false, r1},
		{MaxBigInt, 18, 18, decimal.ToZero, false, r2},
	} {
		r, err := SqrtBigInt(tc.number, tc.numberPrecision, tc.resultPrecision, tc.mode)
		if tc.error {
			assert.Error(t, err, i)
			continue
		}
		assert.NoError(t, err, i)
		assert.Equal(t, tc.expected, r, i)
	}
}
