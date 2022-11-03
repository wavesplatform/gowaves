package ride

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	rideMath "github.com/wavesplatform/gowaves/pkg/ride/math"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPowBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(12), rideInt(1), toRideBigInt(3456), rideInt(3), rideInt(2), newDown(nil)}, false, toRideBigInt(187)},
		{[]rideType{toRideBigInt(12), rideInt(1), toRideBigInt(3456), rideInt(3), rideInt(2), newUp(nil)}, false, toRideBigInt(188)},
		{[]rideType{toRideBigInt(0), rideInt(1), toRideBigInt(3456), rideInt(3), rideInt(2), newUp(nil)}, false, toRideBigInt(0)},
		{[]rideType{toRideBigInt(0), rideInt(1), toRideBigInt(3456), rideInt(3), rideInt(2), newDown(nil)}, false, toRideBigInt(0)},
		{[]rideType{toRideBigInt(20), rideInt(1), toRideBigInt(-1), rideInt(0), rideInt(4), newDown(nil)}, false, toRideBigInt(5000)},
		{[]rideType{toRideBigInt(-20), rideInt(1), toRideBigInt(-1), rideInt(0), rideInt(4), newDown(nil)}, false, toRideBigInt(-5000)},
		{[]rideType{toRideBigInt(0), rideInt(1), toRideBigInt(-1), rideInt(0), rideInt(4), newDown(nil)}, true, nil},
		{[]rideType{toRideBigInt(2), rideInt(0), toRideBigInt(512), rideInt(0), rideInt(0), newDown(nil)}, true, nil},
		{[]rideType{toRideBigInt(12), rideInt(1), toRideBigInt(3456), rideInt(3), rideInt(2), newUp(nil), newDown(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(10), rideInt(0), rideInt(0), newUp(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(1), rideInt(0), rideInt(0), newNoAlg(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(1), rideString("0"), rideInt(0), newUp(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(1), rideInt(0), rideInt(0)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(1), rideInt(0)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(1)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := powBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func BenchmarkPowBigInt(b *testing.B) {
	//pow(d18, 18, max, 0, 18) -> error
	d18, ok := new(big.Int).SetString("987654321012345678", 10)
	require.True(b, ok)
	args := []rideType{rideBigInt{v: d18}, rideInt(18), rideBigInt{v: rideMath.MaxBigInt}, rideInt(0), rideInt(18), newDown(nil)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := powBigInt(nil, args...)
		require.Error(b, err)
		require.Nil(b, r)
	}
}

func TestLogBigInt(t *testing.T) {
	v1, ok := big.NewInt(0).SetString("999996034266679907751935378141784045", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(16), rideInt(0), toRideBigInt(2), rideInt(0), rideInt(0), newCeiling(nil)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(1), rideInt(4), toRideBigInt(1), rideInt(1), rideInt(0), newHalfEven(nil)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(16), rideInt(0), toRideBigInt(-2), rideInt(0), rideInt(0), newCeiling(nil)}, true, nil},
		{[]rideType{toRideBigInt(-16), rideInt(0), toRideBigInt(2), rideInt(0), rideInt(0), newCeiling(nil)}, true, nil},
		{[]rideType{toRideBigInt(1), rideInt(16), toRideBigInt(10), rideInt(0), rideInt(0), newCeiling(nil)}, false, toRideBigInt(-16)},
		{[]rideType{toRideBigInt(100), rideInt(0), toRideBigInt(10), rideInt(0), rideInt(0), newUp(nil)}, false, toRideBigInt(2)},
		{[]rideType{rideBigInt{v: v1}, rideInt(18), toRideBigInt(10001), rideInt(4), rideInt(0), newDown(nil)}, false, toRideBigInt(414485)},
		{[]rideType{toRideBigInt(100), rideInt(0), toRideBigInt(10), rideInt(0), rideInt(0), newUp(nil), newDown(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(100), rideInt(0), rideInt(0), newNoAlg(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(100), rideString("0"), rideInt(0), newUp(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(100), rideInt(0), rideInt(0)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(100), rideInt(0)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(100)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := logBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestToBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(0)}, false, toRideBigInt(0)},
		{[]rideType{rideInt(-1)}, false, toRideBigInt(-1)},
		{[]rideType{rideInt(1)}, false, toRideBigInt(1)},
		{[]rideType{rideInt(-1234567890)}, false, toRideBigInt(-1234567890)},
		{[]rideType{rideInt(1234567890)}, false, toRideBigInt(1234567890)},
		{[]rideType{rideInt(math.MaxInt64)}, false, toRideBigInt(math.MaxInt64)},
		{[]rideType{rideInt(math.MinInt64)}, false, toRideBigInt(math.MinInt64)},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("12345")}, true, nil},
		{[]rideType{toRideBigInt(12345)}, true, nil},
		{[]rideType{rideInt(12345), rideInt(67890)}, true, nil},
	} {
		r, err := toBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestSumBigInt(t *testing.T) {
	doubleMaxInt64 := big.NewInt(math.MaxInt64)
	doubleMaxInt64 = doubleMaxInt64.Add(doubleMaxInt64, doubleMaxInt64)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(5), toRideBigInt(5)}, false, toRideBigInt(10)},
		{[]rideType{toRideBigInt(-5), toRideBigInt(5)}, false, toRideBigInt(0)},
		{[]rideType{toRideBigInt(0), toRideBigInt(0)}, false, toRideBigInt(0)},
		{[]rideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MinInt64)}, false, toRideBigInt(-1)},
		{[]rideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MaxInt64)}, false, rideBigInt{v: doubleMaxInt64}},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, toRideBigInt(1)}, true, nil},
		{[]rideType{toRideBigInt(1), toRideBigInt(1), toRideBigInt(1)}, true, nil},
		{[]rideType{toRideBigInt(1), rideInt(1)}, true, nil},
		{[]rideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := sumBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestSubtractBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(5), toRideBigInt(4)}, false, toRideBigInt(1)},
		{[]rideType{toRideBigInt(5), toRideBigInt(5)}, false, toRideBigInt(0)},
		{[]rideType{toRideBigInt(-5), toRideBigInt(5)}, false, toRideBigInt(-10)},
		{[]rideType{toRideBigInt(0), toRideBigInt(0)}, false, toRideBigInt(0)},
		{[]rideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MaxInt64)}, false, toRideBigInt(0)},
		{[]rideType{rideBigInt{v: rideMath.MinBigInt}, toRideBigInt(1)}, true, nil},
		{[]rideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := subtractBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestMultiplyBigInt(t *testing.T) {
	n := big.NewInt(math.MaxInt64)
	n = n.Mul(n, n)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(5), toRideBigInt(4)}, false, toRideBigInt(20)},
		{[]rideType{toRideBigInt(5), toRideBigInt(5)}, false, toRideBigInt(25)},
		{[]rideType{toRideBigInt(-5), toRideBigInt(5)}, false, toRideBigInt(-25)},
		{[]rideType{toRideBigInt(0), toRideBigInt(0)}, false, toRideBigInt(0)},
		{[]rideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MaxInt64)}, false, rideBigInt{v: n}},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, toRideBigInt(2)}, true, nil},
		{[]rideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := multiplyBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestDivideBigInt(t *testing.T) {
	v1, ok := big.NewInt(0).SetString("-23493686343227100000", 10)
	require.True(t, ok)
	for i, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(10), toRideBigInt(2)}, false, toRideBigInt(5)},
		{[]rideType{toRideBigInt(25), toRideBigInt(5)}, false, toRideBigInt(5)},
		{[]rideType{toRideBigInt(-25), toRideBigInt(5)}, false, toRideBigInt(-5)},
		{[]rideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MaxInt64)}, false, toRideBigInt(1)},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}}, false, toRideBigInt(1)},
		{[]rideType{rideBigInt{v: rideMath.MinBigInt}, rideBigInt{v: rideMath.MinBigInt}}, false, toRideBigInt(1)},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MinBigInt}}, false, toRideBigInt(0)},
		{[]rideType{rideBigInt{v: v1}, toRideBigInt(100000000)}, false, toRideBigInt(-234936863432)},
		{[]rideType{toRideBigInt(10), toRideBigInt(0)}, true, nil},
		{[]rideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := divideBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s, testcase %d", test.r, r, i))
		}
	}
}

func TestModuloBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(10), toRideBigInt(6)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(-10), toRideBigInt(6)}, false, toRideBigInt(-4)},
		{[]rideType{toRideBigInt(10), toRideBigInt(-6)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(-10), toRideBigInt(-6)}, false, toRideBigInt(-4)},
		{[]rideType{toRideBigInt(2), toRideBigInt(2)}, false, toRideBigInt(0)},
		{[]rideType{rideBigInt{v: decode2CBigInt(crypto.MustBytesFromBase58("A98D6oABd9yshGm29dpxXzSeMi1LhCBSPeGxm7MHVB1c"))}, toRideBigInt(330)}, false, toRideBigInt(-243)},
		{[]rideType{toRideBigInt(10), toRideBigInt(0)}, true, nil},
		{[]rideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideBigInt{v: decode2CBigInt(crypto.MustBytesFromBase58("EGhEd4At3siPKgnKdgEgtZvBUFNYn7EoKnsSx35HwJ4a"))}, toRideBigInt(100)}, false, toRideBigInt(-53)},
	} {
		r, err := moduloBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestFractionBigInt(t *testing.T) {
	r1 := new(big.Int).Set(rideMath.MaxBigInt)
	r1 = r1.Mul(r1, big.NewInt(2))
	r1 = r1.Div(r1, big.NewInt(3))
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(math.MaxInt64), toRideBigInt(4), toRideBigInt(6)}, false, toRideBigInt(6148914691236517204)},
		{[]rideType{toRideBigInt(8), toRideBigInt(4), toRideBigInt(2)}, false, toRideBigInt(16)},
		{[]rideType{toRideBigInt(8), toRideBigInt(-2), toRideBigInt(-3)}, false, toRideBigInt(5)},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, toRideBigInt(-2), toRideBigInt(-3)}, false, rideBigInt{v: r1}},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}}, false, rideBigInt{v: rideMath.MaxBigInt}},
		{[]rideType{rideBigInt{v: rideMath.MinBigInt}, rideBigInt{v: rideMath.MinBigInt}, rideBigInt{v: rideMath.MinBigInt}}, false, rideBigInt{v: rideMath.MinBigInt}},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, toRideBigInt(4), toRideBigInt(1)}, true, nil},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, toRideBigInt(4), toRideBigInt(0)}, true, nil},
		{[]rideType{toRideBigInt(2), toRideBigInt(2)}, true, nil},
		{[]rideType{toRideBigInt(1), toRideBigInt(2), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), toRideBigInt(2), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := fractionBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func BenchmarkFractionBigInt(b *testing.B) {
	args := []rideType{rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := fractionBigInt(nil, args...)
		require.NoError(b, err)
		require.NotNil(b, r)
	}
}

func TestFractionBigIntRounds(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(math.MaxInt64), toRideBigInt(4), toRideBigInt(6), newFloor(nil)}, false, toRideBigInt(6148914691236517204)},
		{[]rideType{toRideBigInt(8), toRideBigInt(4), toRideBigInt(2), newFloor(nil)}, false, toRideBigInt(16)},
		{[]rideType{toRideBigInt(8), toRideBigInt(-2), toRideBigInt(-3), newFloor(nil)}, false, toRideBigInt(5)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newDown(nil)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newDown(nil)}, false, toRideBigInt(-4)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newDown(nil)}, false, toRideBigInt(-4)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newDown(nil)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newCeiling(nil)}, false, toRideBigInt(5)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newCeiling(nil)}, false, toRideBigInt(-4)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newCeiling(nil)}, false, toRideBigInt(-4)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newCeiling(nil)}, false, toRideBigInt(5)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newFloor(nil)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newFloor(nil)}, false, toRideBigInt(-5)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newFloor(nil)}, false, toRideBigInt(-5)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newFloor(nil)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newHalfUp(nil)}, false, toRideBigInt(5)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newHalfUp(nil)}, false, toRideBigInt(-5)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newHalfUp(nil)}, false, toRideBigInt(-5)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newHalfUp(nil)}, false, toRideBigInt(5)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newHalfEven(nil)}, false, toRideBigInt(4)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newHalfEven(nil)}, false, toRideBigInt(-4)},
		{[]rideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newHalfEven(nil)}, false, toRideBigInt(-4)},
		{[]rideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newHalfEven(nil)}, false, toRideBigInt(4)},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}, newCeiling(nil)}, false, rideBigInt{v: rideMath.MaxBigInt}},
		{[]rideType{rideBigInt{v: rideMath.MinBigInt}, rideBigInt{v: rideMath.MinBigInt}, rideBigInt{v: rideMath.MinBigInt}, newCeiling(nil)}, false, rideBigInt{v: rideMath.MinBigInt}},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, toRideBigInt(4), toRideBigInt(1), newFloor(nil)}, true, nil},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}, toRideBigInt(4), toRideBigInt(0), newFloor(nil)}, true, nil},
		{[]rideType{toRideBigInt(2), toRideBigInt(2), toRideBigInt(3)}, true, nil},
		{[]rideType{toRideBigInt(1), toRideBigInt(2), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), toRideBigInt(2), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := fractionBigIntRounds(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func BenchmarkFractionBigIntRounds(b *testing.B) {
	args := []rideType{rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MaxBigInt}, newCeiling(nil)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := fractionBigIntRounds(nil, args...)
		require.NoError(b, err)
		require.NotNil(b, r)
	}
}

func TestUnaryMinusBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(math.MaxInt64)}, false, toRideBigInt(-math.MaxInt64)},
		{[]rideType{toRideBigInt(5)}, false, toRideBigInt(-5)},
		{[]rideType{toRideBigInt(0)}, false, toRideBigInt(0)},
		{[]rideType{toRideBigInt(-5)}, false, toRideBigInt(5)},
		{[]rideType{toRideBigInt(math.MinInt64)}, false, rideBigInt{v: big.NewInt(0).Neg(big.NewInt(math.MinInt64))}},
		{[]rideType{rideBigInt{v: rideMath.MinBigInt}}, true, nil},
		{[]rideType{toRideBigInt(1), toRideBigInt(5)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("x")}, true, nil},
	} {
		r, err := unaryMinusBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestGTBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(5), toRideBigInt(4)}, false, rideBoolean(true)},
		{[]rideType{toRideBigInt(16), toRideBigInt(2)}, false, rideBoolean(true)},
		{[]rideType{toRideBigInt(5), toRideBigInt(5)}, false, rideBoolean(false)},
		{[]rideType{toRideBigInt(1), toRideBigInt(5)}, false, rideBoolean(false)},
		{[]rideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), toRideBigInt(2), toRideBigInt(3)}, true, nil},
		{[]rideType{toRideBigInt(1), rideInt(2)}, true, nil},
		{[]rideType{toRideBigInt(1), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := gtBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestGEBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(15), toRideBigInt(5)}, false, rideBoolean(true)},
		{[]rideType{toRideBigInt(5), toRideBigInt(5)}, false, rideBoolean(true)},
		{[]rideType{toRideBigInt(1), toRideBigInt(5)}, false, rideBoolean(false)},
		{[]rideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]rideType{toRideBigInt(1), toRideBigInt(2), toRideBigInt(3)}, true, nil},
		{[]rideType{toRideBigInt(1), rideInt(2)}, true, nil},
		{[]rideType{toRideBigInt(1), rideString("x")}, true, nil},
		{[]rideType{toRideBigInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := geBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestMaxListBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideList(toRideBigInt(1), toRideBigInt(2), toRideBigInt(3))}, false, toRideBigInt(3)},
		{[]rideType{toRideList(toRideBigInt(-1), toRideBigInt(-2), toRideBigInt(-3))}, false, toRideBigInt(-1)},
		{[]rideType{toRideList(toRideBigInt(0), toRideBigInt(0), toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]rideType{toRideList(toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]rideType{toRideList(rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MinBigInt}, toRideBigInt(0), toRideBigInt(-10), toRideBigInt(10))}, false, rideBigInt{v: rideMath.MaxBigInt}},
		{[]rideType{toRideList(toRideBigInt(0)), rideInt(1)}, true, nil},
		{[]rideType{toRideList()}, true, nil},
		{[]rideType{toRideBigInt(0)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := maxListBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestMinListBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideList(toRideBigInt(1), toRideBigInt(2), toRideBigInt(3))}, false, toRideBigInt(1)},
		{[]rideType{toRideList(toRideBigInt(-1), toRideBigInt(-2), toRideBigInt(-3))}, false, toRideBigInt(-3)},
		{[]rideType{toRideList(toRideBigInt(0), toRideBigInt(0), toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]rideType{toRideList(toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]rideType{toRideList(rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MinBigInt}, toRideBigInt(0), toRideBigInt(-10), toRideBigInt(10))}, false, rideBigInt{v: rideMath.MinBigInt}},
		{[]rideType{toRideList(toRideBigInt(0)), rideInt(1)}, true, nil},
		{[]rideType{toRideList()}, true, nil},
		{[]rideType{toRideBigInt(0)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := minListBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestBigIntToBytes(t *testing.T) {
	v, ok := new(big.Int).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(-1)}, false, toRideBytes("ff")},
		{[]rideType{toRideBigInt(0)}, false, toRideBytes("00")},
		{[]rideType{toRideBigInt(1)}, false, toRideBytes("01")},
		{[]rideType{toRideBigInt(1234567890)}, false, toRideBytes("499602d2")},
		{[]rideType{toRideBigInt(-1234567890)}, false, toRideBytes("b669fd2e")},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}}, false, toRideBytes("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")},
		{[]rideType{rideBigInt{v: rideMath.MinBigInt}}, false, toRideBytes("80000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")},
		{[]rideType{toRideBigInt(math.MaxInt64)}, false, toRideBytes("7fffffffffffffff")},
		{[]rideType{toRideBigInt(math.MinInt64)}, false, toRideBytes("8000000000000000")},
		{[]rideType{rideBigInt{v: v}}, false, toRideBytes("0102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F202122232425262728292A2B2C2D2E2F303132333435363738393A3B3C3D3E3F40")},
		{[]rideType{toRideBigInt(0), rideInt(4)}, true, nil},
		{[]rideType{rideString("0")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bigIntToBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBytesToBigInt(t *testing.T) {
	v, ok := new(big.Int).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBytes("ff")}, false, toRideBigInt(-1)},
		{[]rideType{toRideBytes("00")}, false, toRideBigInt(0)},
		{[]rideType{toRideBytes("01")}, false, toRideBigInt(1)},
		{[]rideType{toRideBytes("499602d2")}, false, toRideBigInt(1234567890)},
		{[]rideType{toRideBytes("b669fd2e")}, false, toRideBigInt(-1234567890)},
		{[]rideType{toRideBytes("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")}, false, rideBigInt{v: rideMath.MaxBigInt}},
		{[]rideType{toRideBytes("80000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")}, false, rideBigInt{v: rideMath.MinBigInt}},
		{[]rideType{toRideBytes("7fffffffffffffff")}, false, toRideBigInt(math.MaxInt64)},
		{[]rideType{toRideBytes("8000000000000000")}, false, toRideBigInt(math.MinInt64)},
		{[]rideType{toRideBytes("0102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F202122232425262728292A2B2C2D2E2F303132333435363738393A3B3C3D3E3F40")}, false, rideBigInt{v: v}},
		{[]rideType{toRideBytes("ff"), rideInt(4)}, true, nil},
		{[]rideType{rideString("0")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bytesToBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestBytesToBigIntLim(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBytes("cafebebeff"), rideInt(4), rideInt(1)}, false, toRideBigInt(-1)},
		{[]rideType{toRideBytes("cafebebeff"), rideInt(4), rideInt(4)}, false, toRideBigInt(-1)},
		{[]rideType{toRideBytes("00deadbeef"), rideInt(0), rideInt(1)}, false, toRideBigInt(0)},
		{[]rideType{toRideBytes("cafe01bebe"), rideInt(2), rideInt(1)}, false, toRideBigInt(1)},
		{[]rideType{toRideBytes("deadbeef499602d2"), rideInt(4), rideInt(4)}, false, toRideBigInt(1234567890)},
		{[]rideType{toRideBytes("deadbeefb669fd2e"), rideInt(4), rideInt(4)}, false, toRideBigInt(-1234567890)},
		{[]rideType{toRideBytes("cafebebe7fffffffffffffff"), rideInt(4), rideInt(8)}, false, toRideBigInt(math.MaxInt64)},
		{[]rideType{toRideBytes("8000000000000000cafebebe"), rideInt(0), rideInt(8)}, false, toRideBigInt(math.MinInt64)},
		{[]rideType{toRideBytes("deadbeef00"), rideInt(5), rideInt(1)}, true, nil},
		{[]rideType{toRideBytes("deadbeef00"), rideInt(4), rideInt(65)}, true, nil},
		{[]rideType{toRideBytes("deadbeef00"), rideInt(-1), rideInt(5)}, true, nil},
		{[]rideType{toRideBytes("deadbeef00"), rideInt(4), rideInt(0)}, true, nil},
		{[]rideType{toRideBytes("deadbeef00"), rideInt(4), rideInt(-1)}, true, nil},
		{[]rideType{toRideBytes("ff"), rideInt(4)}, true, nil},
		{[]rideType{rideString("0")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bytesToBigIntLim(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestBigIntToInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(-1)}, false, rideInt(-1)},
		{[]rideType{toRideBigInt(0)}, false, rideInt(0)},
		{[]rideType{toRideBigInt(1)}, false, rideInt(1)},
		{[]rideType{toRideBigInt(1234567890)}, false, rideInt(1234567890)},
		{[]rideType{toRideBigInt(-1234567890)}, false, rideInt(-1234567890)},
		{[]rideType{toRideBigInt(math.MaxInt64)}, false, rideInt(math.MaxInt64)},
		{[]rideType{toRideBigInt(math.MinInt64)}, false, rideInt(math.MinInt64)},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}}, true, nil},
		{[]rideType{rideBigInt{v: rideMath.MinBigInt}}, true, nil},
		{[]rideType{toRideBigInt(0), rideInt(4)}, true, nil},
		{[]rideType{rideString("0")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bigIntToInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBigIntToString(t *testing.T) {
	v, ok := new(big.Int).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(-1)}, false, rideString("-1")},
		{[]rideType{toRideBigInt(0)}, false, rideString("0")},
		{[]rideType{toRideBigInt(1)}, false, rideString("1")},
		{[]rideType{toRideBigInt(1234567890)}, false, rideString("1234567890")},
		{[]rideType{toRideBigInt(-1234567890)}, false, rideString("-1234567890")},
		{[]rideType{rideBigInt{v: rideMath.MaxBigInt}}, false, rideString("6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042047")},
		{[]rideType{rideBigInt{v: rideMath.MinBigInt}}, false, rideString("-6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042048")},
		{[]rideType{toRideBigInt(math.MaxInt64)}, false, rideString("9223372036854775807")},
		{[]rideType{toRideBigInt(math.MinInt64)}, false, rideString("-9223372036854775808")},
		{[]rideType{rideBigInt{v: v}}, false, rideString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480")},
		{[]rideType{toRideBigInt(0), rideInt(4)}, true, nil},
		{[]rideType{rideString("0")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bigIntToString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func BenchmarkBigIntToString(b *testing.B) {
	v, ok := new(big.Int).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(b, ok)
	args := []rideType{rideBigInt{v: v}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := bigIntToString(nil, args...)
		require.NoError(b, err)
		require.NotNil(b, r)
	}
}

func TestStringToBigInt(t *testing.T) {
	v, ok := new(big.Int).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("-1")}, false, toRideBigInt(-1)},
		{[]rideType{rideString("0")}, false, toRideBigInt(0)},
		{[]rideType{rideString("1")}, false, toRideBigInt(1)},
		{[]rideType{rideString("1234567890")}, false, toRideBigInt(1234567890)},
		{[]rideType{rideString("-1234567890")}, false, toRideBigInt(-1234567890)},
		{[]rideType{rideString("6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042047")}, false, rideBigInt{v: rideMath.MaxBigInt}},
		{[]rideType{rideString("-6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042048")}, false, rideBigInt{v: rideMath.MinBigInt}},
		{[]rideType{rideString("9223372036854775807")}, false, toRideBigInt(math.MaxInt64)},
		{[]rideType{rideString("-9223372036854775808")}, false, toRideBigInt(math.MinInt64)},
		{[]rideType{rideString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480")}, false, rideBigInt{v: v}},
		{[]rideType{rideString("0"), rideInt(4)}, true, nil},
		{[]rideType{rideInt(0)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := stringToBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestStringToBigIntOpt(t *testing.T) {
	v, ok := new(big.Int).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("-1")}, false, toRideBigInt(-1)},
		{[]rideType{rideString("0")}, false, toRideBigInt(0)},
		{[]rideType{rideString("1")}, false, toRideBigInt(1)},
		{[]rideType{rideString("1234567890")}, false, toRideBigInt(1234567890)},
		{[]rideType{rideString("-1234567890")}, false, toRideBigInt(-1234567890)},
		{[]rideType{rideString("6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042047")}, false, rideBigInt{v: rideMath.MaxBigInt}},
		{[]rideType{rideString("-6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042048")}, false, rideBigInt{v: rideMath.MinBigInt}},
		{[]rideType{rideString("9223372036854775807")}, false, toRideBigInt(math.MaxInt64)},
		{[]rideType{rideString("-9223372036854775808")}, false, toRideBigInt(math.MinInt64)},
		{[]rideType{rideString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480")}, false, rideBigInt{v: v}},
		{[]rideType{rideString("0"), rideInt(4)}, false, newUnit(nil)},
		{[]rideType{rideInt(0)}, false, newUnit(nil)},
		{[]rideType{}, false, newUnit(nil)},
	} {
		r, err := stringToBigIntOpt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestMedianListBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideList(toRideBigInt(1), toRideBigInt(2), toRideBigInt(3), toRideBigInt(4))}, false, toRideBigInt(3)},
		{[]rideType{toRideList(toRideBigInt(1), toRideBigInt(2), toRideBigInt(3), toRideBigInt(4), toRideBigInt(5))}, false, toRideBigInt(3)},
		{[]rideType{toRideList(toRideBigInt(-1), toRideBigInt(-2), toRideBigInt(-3))}, false, toRideBigInt(-2)},
		{[]rideType{toRideList(toRideBigInt(0), toRideBigInt(0), toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]rideType{toRideList(toRideBigInt(0), toRideBigInt(1), toRideBigInt(1), toRideBigInt(1), toRideBigInt(1), toRideBigInt(2), toRideBigInt(3))}, false, toRideBigInt(1)},
		{[]rideType{toRideList(rideBigInt{v: rideMath.MaxBigInt}, rideBigInt{v: rideMath.MinBigInt}, toRideBigInt(0), toRideBigInt(-10), toRideBigInt(10))}, false, toRideBigInt(0)},
		{[]rideType{toRideList(toRideBigInt(0))}, true, nil},
		{[]rideType{toRideList(toRideBigInt(0)), rideInt(1)}, true, nil},
		{[]rideType{toRideList()}, true, nil},
		{[]rideType{toRideBigInt(0)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := medianListBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestSqrtBigInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{toRideBigInt(12), rideInt(1), rideInt(2), newDown(nil)}, false, toRideBigInt(109)},
		{[]rideType{toRideBigInt(12), rideInt(1), rideInt(2), newUp(nil)}, false, toRideBigInt(110)},
		{[]rideType{toRideBigInt(12), rideInt(1), rideInt(2), newUp(nil), newDown(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), rideInt(0), newNoAlg(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideString("0"), rideInt(0), newUp(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(20), rideInt(0), newUp(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), rideInt(-1), newUp(nil)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0), toRideBigInt(1)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64), rideInt(0)}, true, nil},
		{[]rideType{toRideBigInt(math.MaxInt64)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := sqrtBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func toRideBigInt(i int) rideBigInt {
	v := big.NewInt(int64(i))
	return rideBigInt{v: v}
}

func toRideBytes(s string) rideBytes {
	r, _ := hex.DecodeString(s)
	return r
}

func toRideList(args ...rideType) rideList {
	return args
}
