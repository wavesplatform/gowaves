package ride

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcatStrings(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("abc"), RideString("def")}, false, RideString("abcdef")},
		{[]RideType{RideString("abc"), RideString("")}, false, RideString("abc")},
		{[]RideType{RideString(""), RideString("def")}, false, RideString("def")},
		{[]RideType{RideString(""), RideString("")}, false, RideString("")},
		{[]RideType{RideString("abc")}, true, nil},
		{[]RideType{RideString("abc"), RideInt(0)}, true, nil},
		{[]RideType{RideString("abc"), RideString("def"), RideString("ghi")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := concatStrings(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestTakeString(t *testing.T) {
	env := &MockRideEnvironment{
		takeStringFunc: v5takeString,
	}
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("abc"), RideInt(2)}, false, RideString("ab")},
		{[]RideType{RideString("abc"), RideInt(4)}, false, RideString("abc")},
		{[]RideType{RideString("abc"), RideInt(0)}, false, RideString("")},
		{[]RideType{RideString("abc"), RideInt(-4)}, false, RideString("")},
		{[]RideType{RideString(""), RideInt(0)}, false, RideString("")},
		{[]RideType{RideString(""), RideInt(3)}, false, RideString("")},
		{[]RideType{RideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		{[]RideType{RideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\nCeli, child of the first light. One of the main characters of the story, she is the first to see the vision of Cloudscape and its inhabitants from the Earth's dimension after the great destruction.\n\nDragorion - avatars sung into being by Eneria to bring sleep to the people of Cloudscape. They speak in dreams as lullabies, symphonies, hymns, arias and melodies. ~Legendarium\n\n©️Art of Monztre\n"), RideInt(50)}, false, RideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\n")},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{RideString("x冬x"), RideInt(2)}, false, RideString("x冬")}, // the result is `x?` but it should be `x冬`
	} {
		r, err := takeString(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestIncorrectTakeString(t *testing.T) {
	env := &MockRideEnvironment{
		takeStringFunc: takeRideStringWrong,
	}
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("abc"), RideInt(2)}, false, RideString("ab")},
		{[]RideType{RideString("abc"), RideInt(4)}, false, RideString("abc")},
		{[]RideType{RideString("abc"), RideInt(0)}, false, RideString("")},
		{[]RideType{RideString("abc"), RideInt(-4)}, false, RideString("")},
		{[]RideType{RideString(""), RideInt(0)}, false, RideString("")},
		{[]RideType{RideString(""), RideInt(3)}, false, RideString("")},
		{[]RideType{RideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		{[]RideType{RideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\nCeli, child of the first light. One of the main characters of the story, she is the first to see the vision of Cloudscape and its inhabitants from the Earth's dimension after the great destruction.\n\nDragorion - avatars sung into being by Eneria to bring sleep to the people of Cloudscape. They speak in dreams as lullabies, symphonies, hymns, arias and melodies. ~Legendarium\n\n©️Art of Monztre\n"), RideInt(50)}, false, RideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶?")},
	} {
		r, err := takeString(env, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestDropString(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("abcde"), RideInt(2)}, false, RideString("cde")},
		{[]RideType{RideString("abcde"), RideInt(4)}, false, RideString("e")},
		{[]RideType{RideString("abc"), RideInt(0)}, false, RideString("abc")},
		{[]RideType{RideString("abc"), RideInt(-4)}, false, RideString("abc")},
		{[]RideType{RideString(""), RideInt(0)}, false, RideString("")},
		{[]RideType{RideString(""), RideInt(3)}, false, RideString("")},
		{[]RideType{RideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{RideString("x冬x"), RideInt(2)}, false, RideString("x")},
	} {
		r, err := dropString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSizeString(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("Hello")}, false, RideInt(5)},
		{[]RideType{RideString("Привет")}, false, RideInt(6)},
		{[]RideType{RideString("世界")}, false, RideInt(2)},
		{[]RideType{RideString("")}, false, RideInt(0)},
		{[]RideType{RideString(""), RideInt(3)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{RideString("x冬x")}, false, RideInt(3)},
	} {
		r, err := sizeString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestIndexOfSubstring(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("brown")}, false, RideInt(6)},
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("cafe")}, false, rideUnit{}},
		{[]RideType{RideString("")}, true, nil},
		{[]RideType{RideString(""), RideInt(3)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{RideString("x冬xqweqwe"), RideString("we")}, false, RideInt(4)},          // unicode indexOf
		{[]RideType{takeRideString("世界x冬x", 4), takeRideString("冬", 1)}, false, RideInt(3)}, // unicode indexOf
		{[]RideType{RideString("x冬xqweqwe"), RideString("ww")}, false, rideUnit{}},          // unicode indexOf (not present)
		{[]RideType{RideString(""), RideString("x冬x")}, false, rideUnit{}},                  // unicode indexOf from empty string
	} {
		r, err := indexOfSubstring(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestIndexOfSubstringWithOffset(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("brown"), RideInt(0)}, false, RideInt(6)},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe"), RideInt(10)}, false, RideInt(25)},
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("brown"), RideInt(10)}, false, rideUnit{}},
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("fox"), RideInt(1000)}, false, rideUnit{}},
		{[]RideType{RideString("")}, true, nil},
		{[]RideType{RideString(""), RideInt(3)}, true, nil},
		{[]RideType{RideString(""), RideString(""), RideInt(3), RideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{RideString("x冬xqweqwe"), RideString("x冬xqw"), RideInt(0)}, false, RideInt(0)}, // unicode indexOf with zero offset
		{[]RideType{RideString("冬weqwe"), RideString("we"), RideInt(2)}, false, RideInt(4)},       // unicode indexOf with start offset
		{[]RideType{RideString(""), RideString("x冬x"), RideInt(1)}, false, rideUnit{}},            // unicode indexOf from empty string with offset
	} {
		r, err := indexOfSubstringWithOffset(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestStringToBytes(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("Hello")}, false, RideBytes("Hello")},
		{[]RideType{RideString("Привет")}, false, RideBytes("Привет")},
		{[]RideType{RideString("世界")}, false, RideBytes("世界")},
		{[]RideType{RideString("")}, false, RideBytes{}},
		{[]RideType{RideString(""), RideInt(3)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := stringToBytes(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestDropRightString(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("abcde"), RideInt(2)}, false, RideString("abc")},
		{[]RideType{RideString("abcde"), RideInt(4)}, false, RideString("a")},
		{[]RideType{RideString("abcde"), RideInt(6)}, false, RideString("")},
		{[]RideType{RideString("abc"), RideInt(0)}, false, RideString("abc")},
		{[]RideType{RideString("abc"), RideInt(-4)}, false, RideString("abc")},
		{[]RideType{RideString(""), RideInt(0)}, false, RideString("")},
		{[]RideType{RideString(""), RideInt(3)}, false, RideString("")},
		{[]RideType{RideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{RideString("x冬x"), RideInt(2)}, false, RideString("x")},
	} {
		r, err := dropRightString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestTakeRightString(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("abcde"), RideInt(2)}, false, RideString("de")},
		{[]RideType{RideString("abcde"), RideInt(4)}, false, RideString("bcde")},
		{[]RideType{RideString("abcde"), RideInt(6)}, false, RideString("abcde")},
		{[]RideType{RideString("abc"), RideInt(0)}, false, RideString("")},
		{[]RideType{RideString("abc"), RideInt(-4)}, false, RideString("")},
		{[]RideType{RideString(""), RideInt(0)}, false, RideString("")},
		{[]RideType{RideString(""), RideInt(3)}, false, RideString("")},
		{[]RideType{RideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{RideString("x冬x"), RideInt(2)}, false, RideString("冬x")},
	} {
		r, err := takeRightString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSplitString(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("abcdefg"), RideString("")}, false, RideList{RideString("a"), RideString("b"), RideString("c"), RideString("d"), RideString("e"), RideString("f"), RideString("g")}},
		{[]RideType{RideString("one two three four"), RideString(" ")}, false, RideList{RideString("one"), RideString("two"), RideString("three"), RideString("four")}},
		{[]RideType{RideString(""), RideString(" ")}, false, RideList{RideString("")}},
		{[]RideType{RideString(" "), RideString(" ")}, false, RideList{RideString(""), RideString("")}},
		{[]RideType{RideString(""), RideString("")}, false, RideList{}},
		{[]RideType{RideString(" "), RideString("")}, false, RideList{RideString(" ")}},
		{[]RideType{RideString("abc"), RideInt(0)}, true, nil},
		{[]RideType{RideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{RideString("strx冬x1;🤦;🤦strx冬x2;🤦strx冬x3"), RideString(";🤦")}, false, RideList{RideString("strx冬x1"), RideString(""), RideString("strx冬x2"), RideString("strx冬x3")}},
	} {
		r, err := splitString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestParseInt(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("123345")}, false, RideInt(123345)},
		{[]RideType{RideString("0")}, false, RideInt(0)},
		{[]RideType{RideString(fmt.Sprint(math.MaxInt64))}, false, RideInt(math.MaxInt64)},
		{[]RideType{RideString(fmt.Sprint(math.MinInt64))}, false, RideInt(math.MinInt64)},
		{[]RideType{RideString("")}, false, rideUnit{}},
		{[]RideType{RideString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")}, false, rideUnit{}},
		{[]RideType{RideString("abc")}, false, rideUnit{}},
		{[]RideType{RideString("abc"), RideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := parseInt(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestParseIntValue(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("123345")}, false, RideInt(123345)},
		{[]RideType{RideString("0")}, false, RideInt(0)},
		{[]RideType{RideString(fmt.Sprint(math.MaxInt64))}, false, RideInt(math.MaxInt64)},
		{[]RideType{RideString(fmt.Sprint(math.MinInt64))}, false, RideInt(math.MinInt64)},
		{[]RideType{RideString("")}, false, rideThrow("failed to extract from Unit value")},
		{[]RideType{RideString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")}, false, rideThrow("failed to extract from Unit value")},
		{[]RideType{RideString("abc")}, false, rideThrow("failed to extract from Unit value")},
		{[]RideType{RideString("abc"), RideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := parseIntValue(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestLastIndexOfSubstring(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe")}, false, RideInt(25)},
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("cafe")}, false, rideUnit{}},
		{[]RideType{RideString("")}, true, nil},
		{[]RideType{RideString(""), RideInt(3)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := lastIndexOfSubstring(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestLastIndexOfSubstringWithOffset(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe"), RideInt(30)}, false, RideInt(25)},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe"), RideInt(25)}, false, RideInt(25)},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe"), RideInt(10)}, false, RideInt(5)},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe"), RideInt(5)}, false, RideInt(5)},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe"), RideInt(4)}, false, rideUnit{}},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe"), RideInt(0)}, false, rideUnit{}},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("bebe"), RideInt(-2)}, false, rideUnit{}},
		{[]RideType{RideString("aaa"), RideString("a"), RideInt(0)}, false, RideInt(0)},
		{[]RideType{RideString("aaa"), RideString("b"), RideInt(0)}, false, rideUnit{}},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("dead"), RideInt(11)}, false, RideInt(10)},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("dead"), RideInt(10)}, false, RideInt(10)},
		{[]RideType{RideString("cafe bebe dead beef cafe bebe"), RideString("dead"), RideInt(9)}, false, rideUnit{}},
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("brown"), RideInt(12)}, false, RideInt(6)},
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("fox"), RideInt(14)}, false, RideInt(12)},
		{[]RideType{RideString("quick brown fox jumps over the lazy dog"), RideString("fox"), RideInt(13)}, false, RideInt(12)},
		{[]RideType{RideString("")}, true, nil},
		{[]RideType{RideString(""), RideInt(3)}, true, nil},
		{[]RideType{RideString(""), RideString(""), RideInt(3), RideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := lastIndexOfSubstringWithOffset(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestMakeString(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideList{RideString("1"), RideString("2"), RideString("3")}, RideString(" ")}, false, RideString("1 2 3")},
		{[]RideType{RideList{RideString("one"), RideString("two"), RideString("three")}, RideString(", ")}, false, RideString("one, two, three")},
		{[]RideType{RideList{RideString("")}, RideString("")}, false, RideString("")},
		{[]RideType{RideList{}, RideString(",")}, false, RideString("")},
		{[]RideType{RideList{RideString("one"), RideInt(2), RideString("tree")}, RideString(", ")}, true, nil},
		{[]RideType{RideString("")}, true, nil},
		{[]RideType{RideString(""), RideInt(3)}, true, nil},
		{[]RideType{RideString("1"), RideString("2"), RideString("3")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := makeString(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestContains(t *testing.T) {
	for _, test := range []struct {
		args []RideType
		fail bool
		r    RideType
	}{
		{[]RideType{RideString("ride"), RideString("ide")}, false, RideBoolean(true)},
		{[]RideType{RideString("string"), RideString("substring")}, false, RideBoolean(false)},
		{[]RideType{RideString(""), RideString("")}, false, RideBoolean(true)},
		{[]RideType{RideString("ride"), RideString("")}, false, RideBoolean(true)},
		{[]RideType{RideString(""), RideString("ride")}, false, RideBoolean(false)},
		{[]RideType{RideString(""), RideInt(3)}, true, nil},
		{[]RideType{RideString(""), RideString(""), RideInt(3), RideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{RideInt(1), RideString("x")}, true, nil},
		{[]RideType{RideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
	} {
		r, err := contains(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}
