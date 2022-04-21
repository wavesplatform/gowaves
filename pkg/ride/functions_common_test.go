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
		r, err := getType(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSizeTuple(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{tuple2{}}, false, rideInt(2)},
		{[]rideType{tuple3{}}, false, rideInt(3)},
		{[]rideType{tuple4{}}, false, rideInt(4)},
		{[]rideType{tuple5{}}, false, rideInt(5)},
		{[]rideType{tuple6{}}, false, rideInt(6)},
		{[]rideType{tuple7{}}, false, rideInt(7)},
		{[]rideType{tuple8{}}, false, rideInt(8)},
		{[]rideType{tuple9{}}, false, rideInt(9)},
		{[]rideType{tuple10{}}, false, rideInt(10)},
		{[]rideType{tuple11{}}, false, rideInt(11)},
		{[]rideType{tuple12{}}, false, rideInt(12)},
		{[]rideType{tuple13{}}, false, rideInt(13)},
		{[]rideType{tuple14{}}, false, rideInt(14)},
		{[]rideType{tuple15{}}, false, rideInt(15)},
		{[]rideType{tuple16{}}, false, rideInt(16)},
		{[]rideType{tuple17{}}, false, rideInt(17)},
		{[]rideType{tuple18{}}, false, rideInt(18)},
		{[]rideType{tuple19{}}, false, rideInt(19)},
		{[]rideType{tuple20{}}, false, rideInt(20)},
		{[]rideType{tuple21{}}, false, rideInt(21)},
		{[]rideType{tuple22{}}, false, rideInt(22)},
		{[]rideType{rideString("xxx")}, true, nil},
		{[]rideType{rideBoolean(true)}, true, nil},
		{[]rideType{rideList{rideString("xxx"), rideInt(123)}}, true, nil},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideInt(3)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := sizeTuple(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
