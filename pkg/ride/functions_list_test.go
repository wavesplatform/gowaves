package ride

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRemoveByIndex(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideList{rideInt(1), rideInt(2), rideInt(3)}, rideInt(0)}, false, rideList{rideInt(2), rideInt(3)}},
		{[]rideType{rideList{rideInt(1), rideInt(2), rideInt(3)}, rideInt(1)}, false, rideList{rideInt(1), rideInt(3)}},
		{[]rideType{rideList{rideInt(1), rideInt(2), rideInt(3)}, rideInt(2)}, false, rideList{rideInt(1), rideInt(2)}},
		{[]rideType{rideList{rideInt(1), rideString("two"), rideBoolean(true)}, rideInt(2)}, false, rideList{rideInt(1), rideString("two")}},
		{[]rideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]rideType{rideList{}, rideInt(0)}, true, nil},
		{[]rideType{rideList{rideString("a")}, rideInt(-1)}, true, nil},
		{[]rideType{rideList{rideString("a")}, rideInt(1)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
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
