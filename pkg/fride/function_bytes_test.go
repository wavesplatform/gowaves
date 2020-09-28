package fride

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytesToInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 0}}, false, rideInt(0)},
		{[]rideType{rideBytes{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideBytes{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5}}, false, rideInt(1)},
		{[]rideType{rideBytes{}}, true, nil},
		{[]rideType{rideBytes{0, 0, 0, 0}}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := bytesToInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
