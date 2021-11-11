package ride

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"testing"

	rideMath "github.com/wavesplatform/gowaves/pkg/ride/math"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var env Environment = &MockRideEnvironment{
	validateInternalPaymentsFunc: func() bool {
		return false
	},
}

func TestPowBigInt(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(12), RideInt(1), toRideBigInt(3456), RideInt(3), RideInt(2), newDown(nil)}, false, toRideBigInt(187)},
		{[]RideType{toRideBigInt(12), RideInt(1), toRideBigInt(3456), RideInt(3), RideInt(2), newUp(nil)}, false, toRideBigInt(188)},
		{[]RideType{toRideBigInt(0), RideInt(1), toRideBigInt(3456), RideInt(3), RideInt(2), newUp(nil)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(0), RideInt(1), toRideBigInt(3456), RideInt(3), RideInt(2), newDown(nil)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(20), RideInt(1), toRideBigInt(-1), RideInt(0), RideInt(4), newDown(nil)}, false, toRideBigInt(5000)},
		{[]RideType{toRideBigInt(-20), RideInt(1), toRideBigInt(-1), RideInt(0), RideInt(4), newDown(nil)}, false, toRideBigInt(-5000)},
		{[]RideType{toRideBigInt(0), RideInt(1), toRideBigInt(-1), RideInt(0), RideInt(4), newDown(nil)}, true, nil},
		{[]RideType{toRideBigInt(2), RideInt(0), toRideBigInt(512), RideInt(0), RideInt(0), newDown(nil)}, true, nil},
		{[]RideType{toRideBigInt(12), RideInt(1), toRideBigInt(3456), RideInt(3), RideInt(2), newUp(nil), newDown(nil)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(10), RideInt(0), RideInt(0), newUp(nil)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(1), RideInt(0), RideInt(0), newNoAlg(nil)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(1), RideString("0"), RideInt(0), newUp(nil)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(1), RideInt(0), RideInt(0)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(1), RideInt(0)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(1)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := powBigInt(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestLogBigInt(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(16), RideInt(0), toRideBigInt(2), RideInt(0), RideInt(0), newCeiling(nil)}, false, toRideBigInt(4)},
		{[]RideType{toRideBigInt(1), RideInt(4), toRideBigInt(1), RideInt(1), RideInt(0), newHalfEven(nil)}, false, toRideBigInt(4)},
		{[]RideType{toRideBigInt(16), RideInt(0), toRideBigInt(-2), RideInt(0), RideInt(0), newCeiling(nil)}, true, nil},
		{[]RideType{toRideBigInt(-16), RideInt(0), toRideBigInt(2), RideInt(0), RideInt(0), newCeiling(nil)}, true, nil},
		{[]RideType{toRideBigInt(1), RideInt(16), toRideBigInt(10), RideInt(0), RideInt(0), newCeiling(nil)}, false, toRideBigInt(-16)},
		{[]RideType{toRideBigInt(100), RideInt(0), toRideBigInt(10), RideInt(0), RideInt(0), newUp(nil)}, false, toRideBigInt(2)},
		{[]RideType{toRideBigInt(100), RideInt(0), toRideBigInt(10), RideInt(0), RideInt(0), newUp(nil), newDown(nil)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(100), RideInt(0), RideInt(0), newNoAlg(nil)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(100), RideString("0"), RideInt(0), newUp(nil)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(100), RideInt(0), RideInt(0)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(100), RideInt(0)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0), toRideBigInt(100)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64), RideInt(0)}, true, nil},
		{[]RideType{toRideBigInt(math.MaxInt64)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(0)}, false, toRideBigInt(0)},
		{[]RideType{RideInt(-1)}, false, toRideBigInt(-1)},
		{[]RideType{RideInt(1)}, false, toRideBigInt(1)},
		{[]RideType{RideInt(-1234567890)}, false, toRideBigInt(-1234567890)},
		{[]RideType{RideInt(1234567890)}, false, toRideBigInt(1234567890)},
		{[]RideType{RideInt(math.MaxInt64)}, false, toRideBigInt(math.MaxInt64)},
		{[]RideType{RideInt(math.MinInt64)}, false, toRideBigInt(math.MinInt64)},
		{[]RideType{}, true, nil},
		{[]RideType{RideString("12345")}, true, nil},
		{[]RideType{toRideBigInt(12345)}, true, nil},
		{[]RideType{RideInt(12345), RideInt(67890)}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(5), toRideBigInt(5)}, false, toRideBigInt(10)},
		{[]RideType{toRideBigInt(-5), toRideBigInt(5)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(0), toRideBigInt(0)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MinInt64)}, false, toRideBigInt(-1)},
		{[]RideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MaxInt64)}, false, RideBigInt{V: doubleMaxInt64}},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, toRideBigInt(1)}, true, nil},
		{[]RideType{toRideBigInt(1), toRideBigInt(1), toRideBigInt(1)}, true, nil},
		{[]RideType{toRideBigInt(1), RideInt(1)}, true, nil},
		{[]RideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(5), toRideBigInt(4)}, false, toRideBigInt(1)},
		{[]RideType{toRideBigInt(5), toRideBigInt(5)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(-5), toRideBigInt(5)}, false, toRideBigInt(-10)},
		{[]RideType{toRideBigInt(0), toRideBigInt(0)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MaxInt64)}, false, toRideBigInt(0)},
		{[]RideType{RideBigInt{V: rideMath.MinBigInt}, toRideBigInt(1)}, true, nil},
		{[]RideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(5), toRideBigInt(4)}, false, toRideBigInt(20)},
		{[]RideType{toRideBigInt(5), toRideBigInt(5)}, false, toRideBigInt(25)},
		{[]RideType{toRideBigInt(-5), toRideBigInt(5)}, false, toRideBigInt(-25)},
		{[]RideType{toRideBigInt(0), toRideBigInt(0)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MaxInt64)}, false, RideBigInt{V: n}},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, toRideBigInt(2)}, true, nil},
		{[]RideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(10), toRideBigInt(2)}, false, toRideBigInt(5)},
		{[]RideType{toRideBigInt(25), toRideBigInt(5)}, false, toRideBigInt(5)},
		{[]RideType{toRideBigInt(-25), toRideBigInt(5)}, false, toRideBigInt(-5)},
		{[]RideType{toRideBigInt(math.MaxInt64), toRideBigInt(math.MaxInt64)}, false, toRideBigInt(1)},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MaxBigInt}}, false, toRideBigInt(1)},
		{[]RideType{RideBigInt{V: rideMath.MinBigInt}, RideBigInt{V: rideMath.MinBigInt}}, false, toRideBigInt(1)},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MinBigInt}}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(10), toRideBigInt(0)}, true, nil},
		{[]RideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := divideBigInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.True(t, test.r.eq(r), fmt.Sprintf("%s != %s", test.r, r))
		}
	}
}

func TestModuloBigInt(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(10), toRideBigInt(6)}, false, toRideBigInt(4)},
		{[]RideType{toRideBigInt(-10), toRideBigInt(6)}, false, toRideBigInt(2)},
		{[]RideType{toRideBigInt(10), toRideBigInt(-6)}, false, toRideBigInt(-2)},
		{[]RideType{toRideBigInt(-10), toRideBigInt(-6)}, false, toRideBigInt(-4)},
		{[]RideType{toRideBigInt(2), toRideBigInt(2)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(10), toRideBigInt(0)}, true, nil},
		{[]RideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
	r1 := big.NewInt(0).Set(rideMath.MaxBigInt)
	r1 = r1.Mul(r1, big.NewInt(2))
	r1 = r1.Div(r1, big.NewInt(3))
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(math.MaxInt64), toRideBigInt(4), toRideBigInt(6)}, false, toRideBigInt(6148914691236517204)},
		{[]RideType{toRideBigInt(8), toRideBigInt(4), toRideBigInt(2)}, false, toRideBigInt(16)},
		{[]RideType{toRideBigInt(8), toRideBigInt(-2), toRideBigInt(-3)}, false, toRideBigInt(5)},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, toRideBigInt(-2), toRideBigInt(-3)}, false, RideBigInt{V: r1}},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MaxBigInt}}, false, RideBigInt{V: rideMath.MaxBigInt}},
		{[]RideType{RideBigInt{V: rideMath.MinBigInt}, RideBigInt{V: rideMath.MinBigInt}, RideBigInt{V: rideMath.MinBigInt}}, false, RideBigInt{V: rideMath.MinBigInt}},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, toRideBigInt(4), toRideBigInt(1)}, true, nil},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, toRideBigInt(4), toRideBigInt(0)}, true, nil},
		{[]RideType{toRideBigInt(2), toRideBigInt(2)}, true, nil},
		{[]RideType{toRideBigInt(1), toRideBigInt(2), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), toRideBigInt(2), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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

func TestFractionBigIntRounds(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(math.MaxInt64), toRideBigInt(4), toRideBigInt(6), newFloor(nil)}, false, toRideBigInt(6148914691236517204)},
		{[]RideType{toRideBigInt(8), toRideBigInt(4), toRideBigInt(2), newFloor(nil)}, false, toRideBigInt(16)},
		{[]RideType{toRideBigInt(8), toRideBigInt(-2), toRideBigInt(-3), newFloor(nil)}, false, toRideBigInt(5)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newDown(nil)}, false, toRideBigInt(4)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newDown(nil)}, false, toRideBigInt(-4)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newDown(nil)}, false, toRideBigInt(-4)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newDown(nil)}, false, toRideBigInt(4)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newCeiling(nil)}, false, toRideBigInt(5)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newCeiling(nil)}, false, toRideBigInt(-4)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newCeiling(nil)}, false, toRideBigInt(-4)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newCeiling(nil)}, false, toRideBigInt(5)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newFloor(nil)}, false, toRideBigInt(4)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newFloor(nil)}, false, toRideBigInt(-5)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newFloor(nil)}, false, toRideBigInt(-5)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newFloor(nil)}, false, toRideBigInt(4)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newHalfUp(nil)}, false, toRideBigInt(5)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newHalfUp(nil)}, false, toRideBigInt(-5)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newHalfUp(nil)}, false, toRideBigInt(-5)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newHalfUp(nil)}, false, toRideBigInt(5)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(2), newHalfEven(nil)}, false, toRideBigInt(4)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(2), newHalfEven(nil)}, false, toRideBigInt(-4)},
		{[]RideType{toRideBigInt(9), toRideBigInt(1), toRideBigInt(-2), newHalfEven(nil)}, false, toRideBigInt(-4)},
		{[]RideType{toRideBigInt(-9), toRideBigInt(1), toRideBigInt(-2), newHalfEven(nil)}, false, toRideBigInt(4)},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MaxBigInt}, newCeiling(nil)}, false, RideBigInt{V: rideMath.MaxBigInt}},
		{[]RideType{RideBigInt{V: rideMath.MinBigInt}, RideBigInt{V: rideMath.MinBigInt}, RideBigInt{V: rideMath.MinBigInt}, newCeiling(nil)}, false, RideBigInt{V: rideMath.MinBigInt}},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, toRideBigInt(4), toRideBigInt(1), newFloor(nil)}, true, nil},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}, toRideBigInt(4), toRideBigInt(0), newFloor(nil)}, true, nil},
		{[]RideType{toRideBigInt(2), toRideBigInt(2), toRideBigInt(3)}, true, nil},
		{[]RideType{toRideBigInt(1), toRideBigInt(2), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), toRideBigInt(2), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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

func TestUnaryMinusBigInt(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(math.MaxInt64)}, false, toRideBigInt(-math.MaxInt64)},
		{[]RideType{toRideBigInt(5)}, false, toRideBigInt(-5)},
		{[]RideType{toRideBigInt(0)}, false, toRideBigInt(0)},
		{[]RideType{toRideBigInt(-5)}, false, toRideBigInt(5)},
		{[]RideType{toRideBigInt(math.MinInt64)}, false, RideBigInt{V: big.NewInt(0).Neg(big.NewInt(math.MinInt64))}},
		{[]RideType{RideBigInt{V: rideMath.MinBigInt}}, true, nil},
		{[]RideType{toRideBigInt(1), toRideBigInt(5)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{}, true, nil},
		{[]RideType{RideString("x")}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(5), toRideBigInt(4)}, false, RideBoolean(true)},
		{[]RideType{toRideBigInt(16), toRideBigInt(2)}, false, RideBoolean(true)},
		{[]RideType{toRideBigInt(5), toRideBigInt(5)}, false, RideBoolean(false)},
		{[]RideType{toRideBigInt(1), toRideBigInt(5)}, false, RideBoolean(false)},
		{[]RideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), toRideBigInt(2), toRideBigInt(3)}, true, nil},
		{[]RideType{toRideBigInt(1), RideInt(2)}, true, nil},
		{[]RideType{toRideBigInt(1), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(15), toRideBigInt(5)}, false, RideBoolean(true)},
		{[]RideType{toRideBigInt(5), toRideBigInt(5)}, false, RideBoolean(true)},
		{[]RideType{toRideBigInt(1), toRideBigInt(5)}, false, RideBoolean(false)},
		{[]RideType{toRideBigInt(1), rideUnit{}}, true, nil},
		{[]RideType{toRideBigInt(1), toRideBigInt(2), toRideBigInt(3)}, true, nil},
		{[]RideType{toRideBigInt(1), RideInt(2)}, true, nil},
		{[]RideType{toRideBigInt(1), RideString("x")}, true, nil},
		{[]RideType{toRideBigInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideList(toRideBigInt(1), toRideBigInt(2), toRideBigInt(3))}, false, toRideBigInt(3)},
		{[]RideType{toRideList(toRideBigInt(-1), toRideBigInt(-2), toRideBigInt(-3))}, false, toRideBigInt(-1)},
		{[]RideType{toRideList(toRideBigInt(0), toRideBigInt(0), toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]RideType{toRideList(toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]RideType{toRideList(RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MinBigInt}, toRideBigInt(0), toRideBigInt(-10), toRideBigInt(10))}, false, RideBigInt{V: rideMath.MaxBigInt}},
		{[]RideType{toRideList(toRideBigInt(0)), RideInt(1)}, true, nil},
		{[]RideType{toRideList()}, true, nil},
		{[]RideType{toRideBigInt(0)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideList(toRideBigInt(1), toRideBigInt(2), toRideBigInt(3))}, false, toRideBigInt(1)},
		{[]RideType{toRideList(toRideBigInt(-1), toRideBigInt(-2), toRideBigInt(-3))}, false, toRideBigInt(-3)},
		{[]RideType{toRideList(toRideBigInt(0), toRideBigInt(0), toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]RideType{toRideList(toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]RideType{toRideList(RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MinBigInt}, toRideBigInt(0), toRideBigInt(-10), toRideBigInt(10))}, false, RideBigInt{V: rideMath.MinBigInt}},
		{[]RideType{toRideList(toRideBigInt(0)), RideInt(1)}, true, nil},
		{[]RideType{toRideList()}, true, nil},
		{[]RideType{toRideBigInt(0)}, true, nil},
		{[]RideType{}, true, nil},
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
	v, ok := big.NewInt(0).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(-1)}, false, toRideBytes("ff")},
		{[]RideType{toRideBigInt(0)}, false, toRideBytes("00")},
		{[]RideType{toRideBigInt(1)}, false, toRideBytes("01")},
		{[]RideType{toRideBigInt(1234567890)}, false, toRideBytes("499602d2")},
		{[]RideType{toRideBigInt(-1234567890)}, false, toRideBytes("b669fd2e")},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}}, false, toRideBytes("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")},
		{[]RideType{RideBigInt{V: rideMath.MinBigInt}}, false, toRideBytes("80000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")},
		{[]RideType{toRideBigInt(math.MaxInt64)}, false, toRideBytes("7fffffffffffffff")},
		{[]RideType{toRideBigInt(math.MinInt64)}, false, toRideBytes("8000000000000000")},
		{[]RideType{RideBigInt{V: v}}, false, toRideBytes("0102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F202122232425262728292A2B2C2D2E2F303132333435363738393A3B3C3D3E3F40")},
		{[]RideType{toRideBigInt(0), RideInt(4)}, true, nil},
		{[]RideType{RideString("0")}, true, nil},
		{[]RideType{}, true, nil},
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
	v, ok := big.NewInt(0).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBytes("ff")}, false, toRideBigInt(-1)},
		{[]RideType{toRideBytes("00")}, false, toRideBigInt(0)},
		{[]RideType{toRideBytes("01")}, false, toRideBigInt(1)},
		{[]RideType{toRideBytes("499602d2")}, false, toRideBigInt(1234567890)},
		{[]RideType{toRideBytes("b669fd2e")}, false, toRideBigInt(-1234567890)},
		{[]RideType{toRideBytes("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")}, false, RideBigInt{V: rideMath.MaxBigInt}},
		{[]RideType{toRideBytes("80000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")}, false, RideBigInt{V: rideMath.MinBigInt}},
		{[]RideType{toRideBytes("7fffffffffffffff")}, false, toRideBigInt(math.MaxInt64)},
		{[]RideType{toRideBytes("8000000000000000")}, false, toRideBigInt(math.MinInt64)},
		{[]RideType{toRideBytes("0102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F202122232425262728292A2B2C2D2E2F303132333435363738393A3B3C3D3E3F40")}, false, RideBigInt{V: v}},
		{[]RideType{toRideBytes("ff"), RideInt(4)}, true, nil},
		{[]RideType{RideString("0")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBytes("cafebebeff"), RideInt(4), RideInt(1)}, false, toRideBigInt(-1)},
		{[]RideType{toRideBytes("cafebebeff"), RideInt(4), RideInt(4)}, false, toRideBigInt(-1)},
		{[]RideType{toRideBytes("00deadbeef"), RideInt(0), RideInt(1)}, false, toRideBigInt(0)},
		{[]RideType{toRideBytes("cafe01bebe"), RideInt(2), RideInt(1)}, false, toRideBigInt(1)},
		{[]RideType{toRideBytes("deadbeef499602d2"), RideInt(4), RideInt(4)}, false, toRideBigInt(1234567890)},
		{[]RideType{toRideBytes("deadbeefb669fd2e"), RideInt(4), RideInt(4)}, false, toRideBigInt(-1234567890)},
		{[]RideType{toRideBytes("cafebebe7fffffffffffffff"), RideInt(4), RideInt(8)}, false, toRideBigInt(math.MaxInt64)},
		{[]RideType{toRideBytes("8000000000000000cafebebe"), RideInt(0), RideInt(8)}, false, toRideBigInt(math.MinInt64)},
		{[]RideType{toRideBytes("deadbeef00"), RideInt(5), RideInt(1)}, true, nil},
		{[]RideType{toRideBytes("deadbeef00"), RideInt(4), RideInt(65)}, true, nil},
		{[]RideType{toRideBytes("deadbeef00"), RideInt(-1), RideInt(5)}, true, nil},
		{[]RideType{toRideBytes("deadbeef00"), RideInt(4), RideInt(0)}, true, nil},
		{[]RideType{toRideBytes("deadbeef00"), RideInt(4), RideInt(-1)}, true, nil},
		{[]RideType{toRideBytes("ff"), RideInt(4)}, true, nil},
		{[]RideType{RideString("0")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(-1)}, false, RideInt(-1)},
		{[]RideType{toRideBigInt(0)}, false, RideInt(0)},
		{[]RideType{toRideBigInt(1)}, false, RideInt(1)},
		{[]RideType{toRideBigInt(1234567890)}, false, RideInt(1234567890)},
		{[]RideType{toRideBigInt(-1234567890)}, false, RideInt(-1234567890)},
		{[]RideType{toRideBigInt(math.MaxInt64)}, false, RideInt(math.MaxInt64)},
		{[]RideType{toRideBigInt(math.MinInt64)}, false, RideInt(math.MinInt64)},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}}, true, nil},
		{[]RideType{RideBigInt{V: rideMath.MinBigInt}}, true, nil},
		{[]RideType{toRideBigInt(0), RideInt(4)}, true, nil},
		{[]RideType{RideString("0")}, true, nil},
		{[]RideType{}, true, nil},
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
	v, ok := big.NewInt(0).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideBigInt(-1)}, false, RideString("-1")},
		{[]RideType{toRideBigInt(0)}, false, RideString("0")},
		{[]RideType{toRideBigInt(1)}, false, RideString("1")},
		{[]RideType{toRideBigInt(1234567890)}, false, RideString("1234567890")},
		{[]RideType{toRideBigInt(-1234567890)}, false, RideString("-1234567890")},
		{[]RideType{RideBigInt{V: rideMath.MaxBigInt}}, false, RideString("6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042047")},
		{[]RideType{RideBigInt{V: rideMath.MinBigInt}}, false, RideString("-6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042048")},
		{[]RideType{toRideBigInt(math.MaxInt64)}, false, RideString("9223372036854775807")},
		{[]RideType{toRideBigInt(math.MinInt64)}, false, RideString("-9223372036854775808")},
		{[]RideType{RideBigInt{V: v}}, false, RideString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480")},
		{[]RideType{toRideBigInt(0), RideInt(4)}, true, nil},
		{[]RideType{RideString("0")}, true, nil},
		{[]RideType{}, true, nil},
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

func TestStringToBigInt(t *testing.T) {
	v, ok := big.NewInt(0).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("-1")}, false, toRideBigInt(-1)},
		{[]RideType{RideString("0")}, false, toRideBigInt(0)},
		{[]RideType{RideString("1")}, false, toRideBigInt(1)},
		{[]RideType{RideString("1234567890")}, false, toRideBigInt(1234567890)},
		{[]RideType{RideString("-1234567890")}, false, toRideBigInt(-1234567890)},
		{[]RideType{RideString("6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042047")}, false, RideBigInt{V: rideMath.MaxBigInt}},
		{[]RideType{RideString("-6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042048")}, false, RideBigInt{V: rideMath.MinBigInt}},
		{[]RideType{RideString("9223372036854775807")}, false, toRideBigInt(math.MaxInt64)},
		{[]RideType{RideString("-9223372036854775808")}, false, toRideBigInt(math.MinInt64)},
		{[]RideType{RideString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480")}, false, RideBigInt{V: v}},
		{[]RideType{RideString("0"), RideInt(4)}, true, nil},
		{[]RideType{RideInt(0)}, true, nil},
		{[]RideType{}, true, nil},
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
	v, ok := big.NewInt(0).SetString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480", 10)
	require.True(t, ok)
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("-1")}, false, toRideBigInt(-1)},
		{[]RideType{RideString("0")}, false, toRideBigInt(0)},
		{[]RideType{RideString("1")}, false, toRideBigInt(1)},
		{[]RideType{RideString("1234567890")}, false, toRideBigInt(1234567890)},
		{[]RideType{RideString("-1234567890")}, false, toRideBigInt(-1234567890)},
		{[]RideType{RideString("6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042047")}, false, RideBigInt{V: rideMath.MaxBigInt}},
		{[]RideType{RideString("-6703903964971298549787012499102923063739682910296196688861780721860882015036773488400937149083451713845015929093243025426876941405973284973216824503042048")}, false, RideBigInt{V: rideMath.MinBigInt}},
		{[]RideType{RideString("9223372036854775807")}, false, toRideBigInt(math.MaxInt64)},
		{[]RideType{RideString("-9223372036854775808")}, false, toRideBigInt(math.MinInt64)},
		{[]RideType{RideString("52785833603464895924505196455835395749861094195642486808108138863402869537852026544579466671752822414281401856143643660416162921950916138504990605852480")}, false, RideBigInt{V: v}},
		{[]RideType{RideString("0"), RideInt(4)}, false, newUnit(nil)},
		{[]RideType{RideInt(0)}, false, newUnit(nil)},
		{[]RideType{}, false, newUnit(nil)},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{toRideList(toRideBigInt(1), toRideBigInt(2), toRideBigInt(3), toRideBigInt(4))}, false, toRideBigInt(3)},
		{[]RideType{toRideList(toRideBigInt(1), toRideBigInt(2), toRideBigInt(3), toRideBigInt(4), toRideBigInt(5))}, false, toRideBigInt(3)},
		{[]RideType{toRideList(toRideBigInt(-1), toRideBigInt(-2), toRideBigInt(-3))}, false, toRideBigInt(-2)},
		{[]RideType{toRideList(toRideBigInt(0), toRideBigInt(0), toRideBigInt(0))}, false, toRideBigInt(0)},
		{[]RideType{toRideList(toRideBigInt(0), toRideBigInt(1), toRideBigInt(1), toRideBigInt(1), toRideBigInt(1), toRideBigInt(2), toRideBigInt(3))}, false, toRideBigInt(1)},
		{[]RideType{toRideList(RideBigInt{V: rideMath.MaxBigInt}, RideBigInt{V: rideMath.MinBigInt}, toRideBigInt(0), toRideBigInt(-10), toRideBigInt(10))}, false, toRideBigInt(0)},
		{[]RideType{toRideList(toRideBigInt(0))}, true, nil},
		{[]RideType{toRideList(toRideBigInt(0)), RideInt(1)}, true, nil},
		{[]RideType{toRideList()}, true, nil},
		{[]RideType{toRideBigInt(0)}, true, nil},
		{[]RideType{}, true, nil},
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

func toRideBigInt(i int) RideBigInt {
	v := big.NewInt(int64(i))
	return RideBigInt{V: v}
}

func toRideBytes(s string) RideBytes {
	r, _ := hex.DecodeString(s)
	return r
}

func toRideList(args ...RideType) RideList {
	return args
}
