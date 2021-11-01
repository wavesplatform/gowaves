package ride

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBooleanToByte(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{rideBoolean(true)}, false, rideBytes{1}},
		{[]RideType{rideBoolean(false)}, false, rideBytes{0}},
		{[]RideType{rideBoolean(true), rideBoolean(false)}, true, nil},
		{[]RideType{rideBytes{1}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{rideBoolean(true)}, false, rideString("true")},
		{[]RideType{rideBoolean(false)}, false, rideString("false")},
		{[]RideType{rideBoolean(true), rideBoolean(false)}, true, nil},
		{[]RideType{rideBytes{1}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{rideBoolean(true)}, false, rideBoolean(false)},
		{[]RideType{rideBoolean(false)}, false, rideBoolean(true)},
		{[]RideType{rideBoolean(true), rideBoolean(false)}, true, nil},
		{[]RideType{rideBytes{1}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
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
