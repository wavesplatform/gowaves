package ride

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRemoveByIndex(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideList{RideInt(1), RideInt(2), RideInt(3)}, RideInt(0)}, false, RideList{RideInt(2), RideInt(3)}},
		{[]RideType{RideList{RideInt(1), RideInt(2), RideInt(3)}, RideInt(1)}, false, RideList{RideInt(1), RideInt(3)}},
		{[]RideType{RideList{RideInt(1), RideInt(2), RideInt(3)}, RideInt(2)}, false, RideList{RideInt(1), RideInt(2)}},
		{[]RideType{RideList{RideInt(1), RideString("two"), RideBoolean(true)}, RideInt(2)}, false, RideList{RideInt(1), RideString("two")}},
		{[]RideType{RideString("abc"), RideInt(0)}, true, nil},
		{[]RideType{RideList{}, RideInt(0)}, true, nil},
		{[]RideType{RideList{RideString("a")}, RideInt(-1)}, true, nil},
		{[]RideType{RideList{RideString("a")}, RideInt(1)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := listRemoveByIndex(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
