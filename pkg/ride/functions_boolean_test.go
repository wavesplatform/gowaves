package ride

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBooleanToByte(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideBoolean(true)}, false, rideBytes{1}},
		{[]rideType{rideBoolean(false)}, false, rideBytes{0}},
		{[]rideType{rideBoolean(true), rideBoolean(false)}, true, nil},
		{[]rideType{rideBytes{1}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := booleanToBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestBooleanToString(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideBoolean(true)}, false, rideString("true")},
		{[]rideType{rideBoolean(false)}, false, rideString("false")},
		{[]rideType{rideBoolean(true), rideBoolean(false)}, true, nil},
		{[]rideType{rideBytes{1}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := booleanToString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestUnaryNot(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideBoolean(true)}, false, rideBoolean(false)},
		{[]rideType{rideBoolean(false)}, false, rideBoolean(true)},
		{[]rideType{rideBoolean(true), rideBoolean(false)}, true, nil},
		{[]rideType{rideBytes{1}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := unaryNot(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
