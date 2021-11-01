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
		{[]RideType{rideString("abc"), rideString("def")}, false, rideString("abcdef")},
		{[]RideType{rideString("abc"), rideString("")}, false, rideString("abc")},
		{[]RideType{rideString(""), rideString("def")}, false, rideString("def")},
		{[]RideType{rideString(""), rideString("")}, false, rideString("")},
		{[]RideType{rideString("abc")}, true, nil},
		{[]RideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]RideType{rideString("abc"), rideString("def"), rideString("ghi")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
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
		{[]RideType{rideString("abc"), rideInt(2)}, false, rideString("ab")},
		{[]RideType{rideString("abc"), rideInt(4)}, false, rideString("abc")},
		{[]RideType{rideString("abc"), rideInt(0)}, false, rideString("")},
		{[]RideType{rideString("abc"), rideInt(-4)}, false, rideString("")},
		{[]RideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]RideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]RideType{rideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		{[]RideType{rideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\nCeli, child of the first light. One of the main characters of the story, she is the first to see the vision of Cloudscape and its inhabitants from the Earth's dimension after the great destruction.\n\nDragorion - avatars sung into being by Eneria to bring sleep to the people of Cloudscape. They speak in dreams as lullabies, symphonies, hymns, arias and melodies. ~Legendarium\n\n©️Art of Monztre\n"), rideInt(50)}, false, rideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\n")},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{rideString("x冬x"), rideInt(2)}, false, rideString("x冬")}, // the result is `x?` but it should be `x冬`
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
		{[]RideType{rideString("abc"), rideInt(2)}, false, rideString("ab")},
		{[]RideType{rideString("abc"), rideInt(4)}, false, rideString("abc")},
		{[]RideType{rideString("abc"), rideInt(0)}, false, rideString("")},
		{[]RideType{rideString("abc"), rideInt(-4)}, false, rideString("")},
		{[]RideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]RideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]RideType{rideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		{[]RideType{rideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\nCeli, child of the first light. One of the main characters of the story, she is the first to see the vision of Cloudscape and its inhabitants from the Earth's dimension after the great destruction.\n\nDragorion - avatars sung into being by Eneria to bring sleep to the people of Cloudscape. They speak in dreams as lullabies, symphonies, hymns, arias and melodies. ~Legendarium\n\n©️Art of Monztre\n"), rideInt(50)}, false, rideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶?")},
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
		{[]RideType{rideString("abcde"), rideInt(2)}, false, rideString("cde")},
		{[]RideType{rideString("abcde"), rideInt(4)}, false, rideString("e")},
		{[]RideType{rideString("abc"), rideInt(0)}, false, rideString("abc")},
		{[]RideType{rideString("abc"), rideInt(-4)}, false, rideString("abc")},
		{[]RideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]RideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]RideType{rideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{rideString("x冬x"), rideInt(2)}, false, rideString("x")},
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
		{[]RideType{rideString("Hello")}, false, rideInt(5)},
		{[]RideType{rideString("Привет")}, false, rideInt(6)},
		{[]RideType{rideString("世界")}, false, rideInt(2)},
		{[]RideType{rideString("")}, false, rideInt(0)},
		{[]RideType{rideString(""), rideInt(3)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{rideString("x冬x")}, false, rideInt(3)},
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
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("brown")}, false, rideInt(6)},
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("cafe")}, false, rideUnit{}},
		{[]RideType{rideString("")}, true, nil},
		{[]RideType{rideString(""), rideInt(3)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{rideString("x冬xqweqwe"), rideString("we")}, false, rideInt(4)},          // unicode indexOf
		{[]RideType{takeRideString("世界x冬x", 4), takeRideString("冬", 1)}, false, rideInt(3)}, // unicode indexOf
		{[]RideType{rideString("x冬xqweqwe"), rideString("ww")}, false, rideUnit{}},          // unicode indexOf (not present)
		{[]RideType{rideString(""), rideString("x冬x")}, false, rideUnit{}},                  // unicode indexOf from empty string
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
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("brown"), rideInt(0)}, false, rideInt(6)},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(10)}, false, rideInt(25)},
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("brown"), rideInt(10)}, false, rideUnit{}},
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("fox"), rideInt(1000)}, false, rideUnit{}},
		{[]RideType{rideString("")}, true, nil},
		{[]RideType{rideString(""), rideInt(3)}, true, nil},
		{[]RideType{rideString(""), rideString(""), rideInt(3), rideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{rideString("x冬xqweqwe"), rideString("x冬xqw"), rideInt(0)}, false, rideInt(0)}, // unicode indexOf with zero offset
		{[]RideType{rideString("冬weqwe"), rideString("we"), rideInt(2)}, false, rideInt(4)},       // unicode indexOf with start offset
		{[]RideType{rideString(""), rideString("x冬x"), rideInt(1)}, false, rideUnit{}},            // unicode indexOf from empty string with offset
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
		{[]RideType{rideString("Hello")}, false, rideBytes("Hello")},
		{[]RideType{rideString("Привет")}, false, rideBytes("Привет")},
		{[]RideType{rideString("世界")}, false, rideBytes("世界")},
		{[]RideType{rideString("")}, false, rideBytes{}},
		{[]RideType{rideString(""), rideInt(3)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
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
		{[]RideType{rideString("abcde"), rideInt(2)}, false, rideString("abc")},
		{[]RideType{rideString("abcde"), rideInt(4)}, false, rideString("a")},
		{[]RideType{rideString("abcde"), rideInt(6)}, false, rideString("")},
		{[]RideType{rideString("abc"), rideInt(0)}, false, rideString("abc")},
		{[]RideType{rideString("abc"), rideInt(-4)}, false, rideString("abc")},
		{[]RideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]RideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]RideType{rideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{rideString("x冬x"), rideInt(2)}, false, rideString("x")},
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
		{[]RideType{rideString("abcde"), rideInt(2)}, false, rideString("de")},
		{[]RideType{rideString("abcde"), rideInt(4)}, false, rideString("bcde")},
		{[]RideType{rideString("abcde"), rideInt(6)}, false, rideString("abcde")},
		{[]RideType{rideString("abc"), rideInt(0)}, false, rideString("")},
		{[]RideType{rideString("abc"), rideInt(-4)}, false, rideString("")},
		{[]RideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]RideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]RideType{rideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{rideString("x冬x"), rideInt(2)}, false, rideString("冬x")},
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
		{[]RideType{rideString("abcdefg"), rideString("")}, false, rideList{rideString("a"), rideString("b"), rideString("c"), rideString("d"), rideString("e"), rideString("f"), rideString("g")}},
		{[]RideType{rideString("one two three four"), rideString(" ")}, false, rideList{rideString("one"), rideString("two"), rideString("three"), rideString("four")}},
		{[]RideType{rideString(""), rideString(" ")}, false, rideList{rideString("")}},
		{[]RideType{rideString(" "), rideString(" ")}, false, rideList{rideString(""), rideString("")}},
		{[]RideType{rideString(""), rideString("")}, false, rideList{}},
		{[]RideType{rideString(" "), rideString("")}, false, rideList{rideString(" ")}},
		{[]RideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]RideType{rideString("abc")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
		{[]RideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]RideType{rideString("strx冬x1;🤦;🤦strx冬x2;🤦strx冬x3"), rideString(";🤦")}, false, rideList{rideString("strx冬x1"), rideString(""), rideString("strx冬x2"), rideString("strx冬x3")}},
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
		{[]RideType{rideString("123345")}, false, rideInt(123345)},
		{[]RideType{rideString("0")}, false, rideInt(0)},
		{[]RideType{rideString(fmt.Sprint(math.MaxInt64))}, false, rideInt(math.MaxInt64)},
		{[]RideType{rideString(fmt.Sprint(math.MinInt64))}, false, rideInt(math.MinInt64)},
		{[]RideType{rideString("")}, false, rideUnit{}},
		{[]RideType{rideString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")}, false, rideUnit{}},
		{[]RideType{rideString("abc")}, false, rideUnit{}},
		{[]RideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
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
		{[]RideType{rideString("123345")}, false, rideInt(123345)},
		{[]RideType{rideString("0")}, false, rideInt(0)},
		{[]RideType{rideString(fmt.Sprint(math.MaxInt64))}, false, rideInt(math.MaxInt64)},
		{[]RideType{rideString(fmt.Sprint(math.MinInt64))}, false, rideInt(math.MinInt64)},
		{[]RideType{rideString("")}, false, rideThrow("failed to extract from Unit value")},
		{[]RideType{rideString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")}, false, rideThrow("failed to extract from Unit value")},
		{[]RideType{rideString("abc")}, false, rideThrow("failed to extract from Unit value")},
		{[]RideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
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
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe")}, false, rideInt(25)},
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("cafe")}, false, rideUnit{}},
		{[]RideType{rideString("")}, true, nil},
		{[]RideType{rideString(""), rideInt(3)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
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
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(30)}, false, rideInt(25)},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(25)}, false, rideInt(25)},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(10)}, false, rideInt(5)},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(5)}, false, rideInt(5)},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(4)}, false, rideUnit{}},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(0)}, false, rideUnit{}},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(-2)}, false, rideUnit{}},
		{[]RideType{rideString("aaa"), rideString("a"), rideInt(0)}, false, rideInt(0)},
		{[]RideType{rideString("aaa"), rideString("b"), rideInt(0)}, false, rideUnit{}},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("dead"), rideInt(11)}, false, rideInt(10)},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("dead"), rideInt(10)}, false, rideInt(10)},
		{[]RideType{rideString("cafe bebe dead beef cafe bebe"), rideString("dead"), rideInt(9)}, false, rideUnit{}},
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("brown"), rideInt(12)}, false, rideInt(6)},
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("fox"), rideInt(14)}, false, rideInt(12)},
		{[]RideType{rideString("quick brown fox jumps over the lazy dog"), rideString("fox"), rideInt(13)}, false, rideInt(12)},
		{[]RideType{rideString("")}, true, nil},
		{[]RideType{rideString(""), rideInt(3)}, true, nil},
		{[]RideType{rideString(""), rideString(""), rideInt(3), rideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
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
		{[]RideType{rideList{rideString("1"), rideString("2"), rideString("3")}, rideString(" ")}, false, rideString("1 2 3")},
		{[]RideType{rideList{rideString("one"), rideString("two"), rideString("three")}, rideString(", ")}, false, rideString("one, two, three")},
		{[]RideType{rideList{rideString("")}, rideString("")}, false, rideString("")},
		{[]RideType{rideList{}, rideString(",")}, false, rideString("")},
		{[]RideType{rideList{rideString("one"), rideInt(2), rideString("tree")}, rideString(", ")}, true, nil},
		{[]RideType{rideString("")}, true, nil},
		{[]RideType{rideString(""), rideInt(3)}, true, nil},
		{[]RideType{rideString("1"), rideString("2"), rideString("3")}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
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
		{[]RideType{rideString("ride"), rideString("ide")}, false, rideBoolean(true)},
		{[]RideType{rideString("string"), rideString("substring")}, false, rideBoolean(false)},
		{[]RideType{rideString(""), rideString("")}, false, rideBoolean(true)},
		{[]RideType{rideString("ride"), rideString("")}, false, rideBoolean(true)},
		{[]RideType{rideString(""), rideString("ride")}, false, rideBoolean(false)},
		{[]RideType{rideString(""), rideInt(3)}, true, nil},
		{[]RideType{rideString(""), rideString(""), rideInt(3), rideInt(0)}, true, nil},
		{[]RideType{rideUnit{}}, true, nil},
		{[]RideType{rideInt(1), rideString("x")}, true, nil},
		{[]RideType{rideInt(1)}, true, nil},
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
