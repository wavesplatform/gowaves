package ride

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetType(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(5)}, false, rideString("Int")},
		{[]rideType{rideString("xxx")}, false, rideString("String")},
		{[]rideType{rideBoolean(true)}, false, rideString("Boolean")},
		{[]rideType{tuple2{el1: rideString("xxx"), el2: rideInt(123)}}, false, rideString("(String, Int)")},
		{[]rideType{rideList{rideString("xxx"), rideInt(123)}}, false, rideString("List[Any]")},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideInt(3)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := getType(nil, nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
