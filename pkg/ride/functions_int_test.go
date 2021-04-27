package ride

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGE(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(5), rideInt(5)}, false, rideBoolean(true)},
		{[]rideType{rideInt(1), rideInt(5)}, false, rideBoolean(false)},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideInt(3)}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := ge(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestGT(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(5), rideInt(4)}, false, rideBoolean(true)},
		{[]rideType{rideInt(5), rideInt(5)}, false, rideBoolean(false)},
		{[]rideType{rideInt(1), rideInt(5)}, false, rideBoolean(false)},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideInt(3)}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := gt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestIntToString(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(math.MaxInt64)}, false, rideString("9223372036854775807")},
		{[]rideType{rideInt(12345)}, false, rideString("12345")},
		{[]rideType{rideInt(1)}, false, rideString("1")},
		{[]rideType{rideInt(0)}, false, rideString("0")},
		{[]rideType{rideInt(-67890)}, false, rideString("-67890")},
		{[]rideType{rideInt(math.MinInt64)}, false, rideString("-9223372036854775808")},
		{[]rideType{rideString("0")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("x")}, true, nil},
	} {
		r, err := intToString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestUnaryMinus(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(math.MaxInt64)}, false, rideInt(-math.MaxInt64)},
		{[]rideType{rideInt(5)}, false, rideInt(-5)},
		{[]rideType{rideInt(0)}, false, rideInt(0)},
		{[]rideType{rideInt(-5)}, false, rideInt(5)},
		{[]rideType{rideInt(math.MinInt64)}, false, rideInt(math.MinInt64)},
		{[]rideType{rideInt(1), rideInt(5)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("x")}, true, nil},
	} {
		r, err := unaryMinus(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSum(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(5), rideInt(5)}, false, rideInt(10)},
		{[]rideType{rideInt(-5), rideInt(5)}, false, rideInt(0)},
		{[]rideType{rideInt(0), rideInt(0)}, false, rideInt(0)},
		{[]rideType{rideInt(math.MaxInt64), rideInt(math.MinInt64)}, false, rideInt(-1)},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := sum(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSub(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(5), rideInt(4)}, false, rideInt(1)},
		{[]rideType{rideInt(5), rideInt(5)}, false, rideInt(0)},
		{[]rideType{rideInt(-5), rideInt(5)}, false, rideInt(-10)},
		{[]rideType{rideInt(0), rideInt(0)}, false, rideInt(0)},
		{[]rideType{rideInt(math.MaxInt64), rideInt(math.MaxInt64)}, false, rideInt(0)},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := sub(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestMul(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(5), rideInt(4)}, false, rideInt(20)},
		{[]rideType{rideInt(5), rideInt(5)}, false, rideInt(25)},
		{[]rideType{rideInt(-5), rideInt(5)}, false, rideInt(-25)},
		{[]rideType{rideInt(0), rideInt(0)}, false, rideInt(0)},
		{[]rideType{rideInt(math.MaxInt64), rideInt(math.MaxInt64)}, false, rideInt(1)},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := mul(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestDiv(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(10), rideInt(2)}, false, rideInt(5)},
		{[]rideType{rideInt(25), rideInt(5)}, false, rideInt(5)},
		{[]rideType{rideInt(-25), rideInt(5)}, false, rideInt(-5)},
		{[]rideType{rideInt(math.MaxInt64), rideInt(math.MaxInt64)}, false, rideInt(1)},
		{[]rideType{rideInt(10), rideInt(0)}, true, nil},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := div(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestMod(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(10), rideInt(6)}, false, rideInt(4)},
		{[]rideType{rideInt(-10), rideInt(6)}, false, rideInt(2)},
		{[]rideType{rideInt(10), rideInt(-6)}, false, rideInt(-2)},
		{[]rideType{rideInt(-10), rideInt(-6)}, false, rideInt(-4)},
		{[]rideType{rideInt(2), rideInt(2)}, false, rideInt(0)},
		{[]rideType{rideInt(10), rideInt(0)}, true, nil},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := mod(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestFraction(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(math.MaxInt64), rideInt(4), rideInt(6)}, false, rideInt(6148914691236517204)},
		{[]rideType{rideInt(8), rideInt(4), rideInt(2)}, false, rideInt(16)},
		{[]rideType{rideInt(math.MaxInt64), rideInt(math.MinInt64), rideInt(math.MinInt64)}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideInt(1), rideInt(math.MinInt64), rideInt(1)}, false, rideInt(math.MinInt64)},

		{[]rideType{rideInt(math.MaxInt64), rideInt(4), rideInt(1)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(4), rideInt(0)}, true, nil},
		{[]rideType{rideInt(1), rideInt(-1), rideInt(0)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(math.MinInt64), rideInt(1)}, true, nil},

		{[]rideType{rideInt(2), rideInt(2)}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := fraction(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestFractionIntRounds(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(math.MaxInt64), rideInt(4), rideInt(6), newDown(nil)}, false, rideInt(6148914691236517204)},
		{[]rideType{rideInt(8), rideInt(4), rideInt(2), newDown(nil)}, false, rideInt(16)},
		{[]rideType{rideInt(math.MaxInt64), rideInt(math.MinInt64), rideInt(math.MinInt64), newHalfEven(nil)}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideInt(1), rideInt(math.MinInt64), rideInt(1), newHalfEven(nil)}, false, rideInt(math.MinInt64)},
		{[]rideType{rideInt(5), rideInt(1), rideInt(2), newDown(nil)}, false, rideInt(2)},
		{[]rideType{rideInt(5), rideInt(1), rideInt(2), newHalfUp(nil)}, false, rideInt(3)},
		{[]rideType{rideInt(5), rideInt(1), rideInt(2), newHalfEven(nil)}, false, rideInt(2)},
		{[]rideType{rideInt(5), rideInt(1), rideInt(2), newCeiling(nil)}, false, rideInt(3)},
		{[]rideType{rideInt(5), rideInt(1), rideInt(2), newFloor(nil)}, false, rideInt(2)},
		{[]rideType{rideInt(2), rideInt(4), rideInt(5), newDown(nil)}, false, rideInt(1)},
		{[]rideType{rideInt(2), rideInt(4), rideInt(5), newHalfUp(nil)}, false, rideInt(2)},
		{[]rideType{rideInt(2), rideInt(4), rideInt(5), newHalfEven(nil)}, false, rideInt(2)},
		{[]rideType{rideInt(2), rideInt(4), rideInt(5), newCeiling(nil)}, false, rideInt(2)},
		{[]rideType{rideInt(2), rideInt(4), rideInt(5), newFloor(nil)}, false, rideInt(1)},
		{[]rideType{rideInt(-2), rideInt(4), rideInt(5), newDown(nil)}, false, rideInt(-1)},
		{[]rideType{rideInt(-2), rideInt(4), rideInt(5), newHalfUp(nil)}, false, rideInt(-2)},
		{[]rideType{rideInt(-2), rideInt(4), rideInt(5), newHalfEven(nil)}, false, rideInt(-2)},
		{[]rideType{rideInt(-2), rideInt(4), rideInt(5), newCeiling(nil)}, false, rideInt(-1)},
		{[]rideType{rideInt(-2), rideInt(4), rideInt(5), newFloor(nil)}, false, rideInt(-2)},
		{[]rideType{rideInt(-5), rideInt(11), rideInt(10), newDown(nil)}, false, rideInt(-5)},
		{[]rideType{rideInt(-5), rideInt(11), rideInt(10), newHalfUp(nil)}, false, rideInt(-6)},
		{[]rideType{rideInt(-5), rideInt(11), rideInt(10), newHalfEven(nil)}, false, rideInt(-6)},
		{[]rideType{rideInt(-5), rideInt(11), rideInt(10), newCeiling(nil)}, false, rideInt(-5)},
		{[]rideType{rideInt(-5), rideInt(11), rideInt(10), newFloor(nil)}, false, rideInt(-6)},
		{[]rideType{rideInt(math.MaxInt64), rideInt(4), rideInt(1), newDown(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(4), rideInt(0), newDown(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(math.MinInt64), rideInt(1), newHalfEven(nil)}, true, nil},
		{[]rideType{rideInt(1), rideInt(-1), rideInt(0), newHalfEven(nil)}, true, nil},
		{[]rideType{rideInt(2), rideInt(2), newDown(nil)}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideUnit{}, newDown(nil)}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideInt(4)}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := fractionIntRounds(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestIntToBytes(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(0)}, false, rideBytes{0, 0, 0, 0, 0, 0, 0, 0}},
		{[]rideType{rideInt(1)}, false, rideBytes{0, 0, 0, 0, 0, 0, 0, 1}},
		{[]rideType{rideInt(math.MaxInt64)}, false, rideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{[]rideType{rideInt(math.MaxInt64), rideInt(4)}, true, nil},
		{[]rideType{rideString("0")}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := intToBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestPow(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(12), rideInt(1), rideInt(3456), rideInt(3), rideInt(2), newDown(nil)}, false, rideInt(187)},
		{[]rideType{rideInt(12), rideInt(1), rideInt(3456), rideInt(3), rideInt(2), newUp(nil)}, false, rideInt(188)},
		{[]rideType{rideInt(12), rideInt(1), rideInt(3456), rideInt(3), rideInt(2), newUp(nil), newDown(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideInt(0), rideInt(0), newUp(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideInt(0), rideInt(0), newNoAlg(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideString("0"), rideInt(0), newUp(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideInt(0), rideInt(0)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideInt(0)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := pow(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestLog(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(16), rideInt(0), rideInt(2), rideInt(0), rideInt(0), newUp(nil)}, false, rideInt(4)},
		{[]rideType{rideInt(100), rideInt(0), rideInt(10), rideInt(0), rideInt(0), newUp(nil)}, false, rideInt(2)},
		{[]rideType{rideInt(100), rideInt(0), rideInt(10), rideInt(0), rideInt(0), newUp(nil), newDown(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideInt(0), rideInt(0), newNoAlg(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideString("0"), rideInt(0), newUp(nil)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideInt(0), rideInt(0)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100), rideInt(0)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0), rideInt(100)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64), rideInt(0)}, true, nil},
		{[]rideType{rideInt(math.MaxInt64)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := log(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
