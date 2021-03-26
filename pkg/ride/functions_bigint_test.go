package ride

import (
	"math"
	"math/big"
	"testing"

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
			assert.Equal(t, test.r, r)
		}
	}
}

func TestLogBigInt(t *testing.T) {
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
			assert.Equal(t, test.r, r)
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
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSumBigInt(t *testing.T) {
	t.Fail()

}

func TestSubtractBigInt(t *testing.T) {
	t.Fail()

}

func TestMultiplyBigInt(t *testing.T) {
	t.Fail()

}

func TestDivideBigInt(t *testing.T) {
	t.Fail()

}

func TestModuloBigInt(t *testing.T) {
	t.Fail()

}

func TestFractionBigInt(t *testing.T) {
	t.Fail()

}

func TestFractionBigIntRounds(t *testing.T) {
	t.Fail()

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
		{[]rideType{toRideBigInt(math.MinInt64)}, false, rideBigInt(*big.NewInt(0).Neg(big.NewInt(math.MinInt64)))},
		{[]rideType{rideBigInt(*minBigInt)}, true, nil},
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
			assert.Equal(t, test.r, r)
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
			assert.Equal(t, test.r, r)
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
			assert.Equal(t, test.r, r)
		}
	}
}

func TestMaxListBigInt(t *testing.T) {
	t.Fail()

}

func TestMinListBigInt(t *testing.T) {
	t.Fail()

}

func TestBigIntToBytes(t *testing.T) {
	t.Fail()

}

func TestBytesToBigInt(t *testing.T) {
	t.Fail()

}

func TestBytesToBigIntLim(t *testing.T) {
	t.Fail()

}

func TestBigIntToInt(t *testing.T) {
	t.Fail()

}

func TestBigIntToString(t *testing.T) {
	t.Fail()

}

func TestStringToBigInt(t *testing.T) {
	t.Fail()

}

func TestStringToBigIntOpt(t *testing.T) {
	t.Fail()

}

func TestMedianListBigInt(t *testing.T) {
	t.Fail()

}

func toRideBigInt(i int) rideBigInt {
	v := big.NewInt(int64(i))
	return rideBigInt(*v)
}
