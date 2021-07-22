package ride

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGE(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(5), RideInt(5)}, false, RideBoolean(true)},
		{[]RideType{RideInt(1), RideInt(5)}, false, RideBoolean(false)},
		{[]RideType{RideInt(1), rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideInt(2), RideInt(3)}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(5), RideInt(4)}, false, RideBoolean(true)},
		{[]RideType{RideInt(5), RideInt(5)}, false, RideBoolean(false)},
		{[]RideType{RideInt(1), RideInt(5)}, false, RideBoolean(false)},
		{[]RideType{RideInt(1), rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideInt(2), RideInt(3)}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(math.MaxInt64)}, false, RideString("9223372036854775807")},
		{[]RideType{RideInt(12345)}, false, RideString("12345")},
		{[]RideType{RideInt(1)}, false, RideString("1")},
		{[]RideType{RideInt(0)}, false, RideString("0")},
		{[]RideType{RideInt(-67890)}, false, RideString("-67890")},
		{[]RideType{RideInt(math.MinInt64)}, false, RideString("-9223372036854775808")},
		{[]RideType{RideString("0")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{}, true, nil},
		{[]RideType{RideString("x")}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(math.MaxInt64)}, false, RideInt(-math.MaxInt64)},
		{[]RideType{RideInt(5)}, false, RideInt(-5)},
		{[]RideType{RideInt(0)}, false, RideInt(0)},
		{[]RideType{RideInt(-5)}, false, RideInt(5)},
		{[]RideType{RideInt(math.MinInt64)}, false, RideInt(math.MinInt64)},
		{[]RideType{RideInt(1), RideInt(5)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{}, true, nil},
		{[]RideType{RideString("x")}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(5), RideInt(5)}, false, RideInt(10)},
		{[]RideType{RideInt(-5), RideInt(5)}, false, RideInt(0)},
		{[]RideType{RideInt(0), RideInt(0)}, false, RideInt(0)},
		{[]RideType{RideInt(math.MaxInt64), RideInt(math.MinInt64)}, false, RideInt(-1)},
		{[]RideType{RideInt(1), rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(5), RideInt(4)}, false, RideInt(1)},
		{[]RideType{RideInt(5), RideInt(5)}, false, RideInt(0)},
		{[]RideType{RideInt(-5), RideInt(5)}, false, RideInt(-10)},
		{[]RideType{RideInt(0), RideInt(0)}, false, RideInt(0)},
		{[]RideType{RideInt(math.MaxInt64), RideInt(math.MaxInt64)}, false, RideInt(0)},
		{[]RideType{RideInt(1), rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(5), RideInt(4)}, false, RideInt(20)},
		{[]RideType{RideInt(5), RideInt(5)}, false, RideInt(25)},
		{[]RideType{RideInt(-5), RideInt(5)}, false, RideInt(-25)},
		{[]RideType{RideInt(0), RideInt(0)}, false, RideInt(0)},
		{[]RideType{RideInt(math.MaxInt64), RideInt(math.MaxInt64)}, false, RideInt(1)},
		{[]RideType{RideInt(1), rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(10), RideInt(2)}, false, RideInt(5)},
		{[]RideType{RideInt(25), RideInt(5)}, false, RideInt(5)},
		{[]RideType{RideInt(-25), RideInt(5)}, false, RideInt(-5)},
		{[]RideType{RideInt(math.MaxInt64), RideInt(math.MaxInt64)}, false, RideInt(1)},
		{[]RideType{RideInt(10), RideInt(0)}, true, nil},
		{[]RideType{RideInt(1), rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(10), RideInt(6)}, false, RideInt(4)},
		{[]RideType{RideInt(-10), RideInt(6)}, false, RideInt(2)},
		{[]RideType{RideInt(10), RideInt(-6)}, false, RideInt(-2)},
		{[]RideType{RideInt(-10), RideInt(-6)}, false, RideInt(-4)},
		{[]RideType{RideInt(2), RideInt(2)}, false, RideInt(0)},
		{[]RideType{RideInt(10), RideInt(0)}, true, nil},
		{[]RideType{RideInt(1), rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(math.MaxInt64), RideInt(4), RideInt(6)}, false, RideInt(6148914691236517204)},
		{[]RideType{RideInt(8), RideInt(4), RideInt(2)}, false, RideInt(16)},
		{[]RideType{RideInt(math.MaxInt64), RideInt(math.MinInt64), RideInt(math.MinInt64)}, false, RideInt(math.MaxInt64)},
		{[]RideType{RideInt(1), RideInt(math.MinInt64), RideInt(1)}, false, RideInt(math.MinInt64)},

		{[]RideType{RideInt(math.MaxInt64), RideInt(4), RideInt(1)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(4), RideInt(0)}, true, nil},
		{[]RideType{RideInt(1), RideInt(-1), RideInt(0)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(math.MinInt64), RideInt(1)}, true, nil},

		{[]RideType{RideInt(2), RideInt(2)}, true, nil},
		{[]RideType{RideInt(1), RideInt(2), rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideInt(2), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(math.MaxInt64), RideInt(4), RideInt(6), newDown(nil)}, false, RideInt(6148914691236517204)},
		{[]RideType{RideInt(8), RideInt(4), RideInt(2), newDown(nil)}, false, RideInt(16)},
		{[]RideType{RideInt(math.MaxInt64), RideInt(math.MinInt64), RideInt(math.MinInt64), newHalfEven(nil)}, false, RideInt(math.MaxInt64)},
		{[]RideType{RideInt(1), RideInt(math.MinInt64), RideInt(1), newHalfEven(nil)}, false, RideInt(math.MinInt64)},
		{[]RideType{RideInt(5), RideInt(1), RideInt(2), newDown(nil)}, false, RideInt(2)},
		{[]RideType{RideInt(5), RideInt(1), RideInt(2), newHalfUp(nil)}, false, RideInt(3)},
		{[]RideType{RideInt(5), RideInt(1), RideInt(2), newHalfEven(nil)}, false, RideInt(2)},
		{[]RideType{RideInt(5), RideInt(1), RideInt(2), newCeiling(nil)}, false, RideInt(3)},
		{[]RideType{RideInt(5), RideInt(1), RideInt(2), newFloor(nil)}, false, RideInt(2)},
		{[]RideType{RideInt(2), RideInt(4), RideInt(5), newDown(nil)}, false, RideInt(1)},
		{[]RideType{RideInt(2), RideInt(4), RideInt(5), newHalfUp(nil)}, false, RideInt(2)},
		{[]RideType{RideInt(2), RideInt(4), RideInt(5), newHalfEven(nil)}, false, RideInt(2)},
		{[]RideType{RideInt(2), RideInt(4), RideInt(5), newCeiling(nil)}, false, RideInt(2)},
		{[]RideType{RideInt(2), RideInt(4), RideInt(5), newFloor(nil)}, false, RideInt(1)},
		{[]RideType{RideInt(-2), RideInt(4), RideInt(5), newDown(nil)}, false, RideInt(-1)},
		{[]RideType{RideInt(-2), RideInt(4), RideInt(5), newHalfUp(nil)}, false, RideInt(-2)},
		{[]RideType{RideInt(-2), RideInt(4), RideInt(5), newHalfEven(nil)}, false, RideInt(-2)},
		{[]RideType{RideInt(-2), RideInt(4), RideInt(5), newCeiling(nil)}, false, RideInt(-1)},
		{[]RideType{RideInt(-2), RideInt(4), RideInt(5), newFloor(nil)}, false, RideInt(-2)},
		{[]RideType{RideInt(-5), RideInt(11), RideInt(10), newDown(nil)}, false, RideInt(-5)},
		{[]RideType{RideInt(-5), RideInt(11), RideInt(10), newHalfUp(nil)}, false, RideInt(-6)},
		{[]RideType{RideInt(-5), RideInt(11), RideInt(10), newHalfEven(nil)}, false, RideInt(-6)},
		{[]RideType{RideInt(-5), RideInt(11), RideInt(10), newCeiling(nil)}, false, RideInt(-5)},
		{[]RideType{RideInt(-5), RideInt(11), RideInt(10), newFloor(nil)}, false, RideInt(-6)},
		{[]RideType{RideInt(math.MaxInt64), RideInt(4), RideInt(1), newDown(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(4), RideInt(0), newDown(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(math.MinInt64), RideInt(1), newHalfEven(nil)}, true, nil},
		{[]RideType{RideInt(1), RideInt(-1), RideInt(0), newHalfEven(nil)}, true, nil},
		{[]RideType{RideInt(2), RideInt(2), newDown(nil)}, true, nil},
		{[]RideType{RideInt(1), RideInt(2), rideUnit{}, newDown(nil)}, true, nil},
		{[]RideType{RideInt(1), RideInt(2), RideInt(4)}, true, nil},
		{[]RideType{RideInt(1), RideInt(2), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(0)}, false, RideBytes{0, 0, 0, 0, 0, 0, 0, 0}},
		{[]RideType{RideInt(1)}, false, RideBytes{0, 0, 0, 0, 0, 0, 0, 1}},
		{[]RideType{RideInt(math.MaxInt64)}, false, RideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{[]RideType{RideInt(math.MaxInt64), RideInt(4)}, true, nil},
		{[]RideType{RideString("0")}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(12), RideInt(1), RideInt(3456), RideInt(3), RideInt(2), newDown(nil)}, false, RideInt(187)},
		{[]RideType{RideInt(12), RideInt(1), RideInt(3456), RideInt(3), RideInt(2), newUp(nil)}, false, RideInt(188)},
		{[]RideType{RideInt(12), RideInt(1), RideInt(3456), RideInt(3), RideInt(2), newUp(nil), newDown(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideInt(0), RideInt(0), newUp(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideInt(0), RideInt(0), newNoAlg(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideString("0"), RideInt(0), newUp(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideInt(0), RideInt(0)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideInt(0)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideInt(16), RideInt(0), RideInt(2), RideInt(0), RideInt(0), newUp(nil)}, false, RideInt(4)},
		{[]RideType{RideInt(100), RideInt(0), RideInt(10), RideInt(0), RideInt(0), newUp(nil)}, false, RideInt(2)},
		{[]RideType{RideInt(100), RideInt(0), RideInt(10), RideInt(0), RideInt(0), newUp(nil), newDown(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideInt(0), RideInt(0), newNoAlg(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideString("0"), RideInt(0), newUp(nil)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideInt(0), RideInt(0)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100), RideInt(0)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0), RideInt(100)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64), RideInt(0)}, true, nil},
		{[]RideType{RideInt(math.MaxInt64)}, true, nil},
		{[]RideType{}, true, nil},
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
