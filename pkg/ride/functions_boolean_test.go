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
		{[]RideType{RideBoolean(true)}, false, RideBytes{1}},
		{[]RideType{RideBoolean(false)}, false, RideBytes{0}},
		{[]RideType{RideBoolean(true), RideBoolean(false)}, true, nil},
		{[]RideType{RideBytes{1}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
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
		{[]RideType{RideBoolean(true)}, false, RideString("true")},
		{[]RideType{RideBoolean(false)}, false, RideString("false")},
		{[]RideType{RideBoolean(true), RideBoolean(false)}, true, nil},
		{[]RideType{RideBytes{1}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
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
		{[]RideType{RideBoolean(true)}, false, RideBoolean(false)},
		{[]RideType{RideBoolean(false)}, false, RideBoolean(true)},
		{[]RideType{RideBoolean(true), RideBoolean(false)}, true, nil},
		{[]RideType{RideBytes{1}}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
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
