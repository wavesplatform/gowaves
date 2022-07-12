package ride

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcatStrings(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("abc"), rideString("def")}, false, rideString("abcdef")},
		{[]rideType{rideString("abc"), rideString("")}, false, rideString("abc")},
		{[]rideType{rideString(""), rideString("def")}, false, rideString("def")},
		{[]rideType{rideString(""), rideString("")}, false, rideString("")},
		{[]rideType{rideString("abc")}, true, nil},
		{[]rideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]rideType{rideString("abc"), rideString("def"), rideString("ghi")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{}, true, nil},
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
	env := &mockRideEnvironment{
		takeStringFunc: v5takeString,
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("abc"), rideInt(2)}, false, rideString("ab")},
		{[]rideType{rideString("abc"), rideInt(4)}, false, rideString("abc")},
		{[]rideType{rideString("abc"), rideInt(0)}, false, rideString("")},
		{[]rideType{rideString("abc"), rideInt(-4)}, false, rideString("")},
		{[]rideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]rideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]rideType{rideString("abc")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\nCeli, child of the first light. One of the main characters of the story, she is the first to see the vision of Cloudscape and its inhabitants from the Earth's dimension after the great destruction.\n\nDragorion - avatars sung into being by Eneria to bring sleep to the people of Cloudscape. They speak in dreams as lullabies, symphonies, hymns, arias and melodies. ~Legendarium\n\n©️Art of Monztre\n"), rideInt(50)}, false, rideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\n")},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]rideType{rideString("x冬x"), rideInt(2)}, false, rideString("x冬")}, // the result is `x?` but it should be `x冬`
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
	env := &mockRideEnvironment{
		takeStringFunc: takeRideStringWrong,
	}
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("abc"), rideInt(2)}, false, rideString("ab")},
		{[]rideType{rideString("abc"), rideInt(4)}, false, rideString("abc")},
		{[]rideType{rideString("abc"), rideInt(0)}, false, rideString("")},
		{[]rideType{rideString("abc"), rideInt(-4)}, false, rideString("")},
		{[]rideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]rideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]rideType{rideString("abc")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		{[]rideType{rideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶🔶🔶\n\nCeli, child of the first light. One of the main characters of the story, she is the first to see the vision of Cloudscape and its inhabitants from the Earth's dimension after the great destruction.\n\nDragorion - avatars sung into being by Eneria to bring sleep to the people of Cloudscape. They speak in dreams as lullabies, symphonies, hymns, arias and melodies. ~Legendarium\n\n©️Art of Monztre\n"), rideInt(50)}, false, rideString("DRAGORION : Cradle of Many Strings\n[MYTHIC]🔶🔶🔶?")},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("abcde"), rideInt(2)}, false, rideString("cde")},
		{[]rideType{rideString("abcde"), rideInt(4)}, false, rideString("e")},
		{[]rideType{rideString("abc"), rideInt(0)}, false, rideString("abc")},
		{[]rideType{rideString("abc"), rideInt(-4)}, false, rideString("abc")},
		{[]rideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]rideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]rideType{rideString("abc")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]rideType{rideString("x冬x"), rideInt(2)}, false, rideString("x")},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("Hello")}, false, rideInt(5)},
		{[]rideType{rideString("Привет")}, false, rideInt(6)},
		{[]rideType{rideString("世界")}, false, rideInt(2)},
		{[]rideType{rideString("")}, false, rideInt(0)},
		{[]rideType{rideString(""), rideInt(3)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]rideType{rideString("x冬x")}, false, rideInt(3)},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("brown")}, false, rideInt(6)},
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("cafe")}, false, rideUnit{}},
		{[]rideType{rideString("")}, true, nil},
		{[]rideType{rideString(""), rideInt(3)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]rideType{rideString("x冬xqweqwe"), rideString("we")}, false, rideInt(4)},          // unicode indexOf
		{[]rideType{takeRideString("世界x冬x", 4), takeRideString("冬", 1)}, false, rideInt(3)}, // unicode indexOf
		{[]rideType{rideString("x冬xqweqwe"), rideString("ww")}, false, rideUnit{}},          // unicode indexOf (not present)
		{[]rideType{rideString(""), rideString("x冬x")}, false, rideUnit{}},                  // unicode indexOf from empty string
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("brown"), rideInt(0)}, false, rideInt(6)},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(10)}, false, rideInt(25)},
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("brown"), rideInt(10)}, false, rideUnit{}},
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("fox"), rideInt(1000)}, false, rideUnit{}},
		{[]rideType{rideString("")}, true, nil},
		{[]rideType{rideString(""), rideInt(3)}, true, nil},
		{[]rideType{rideString(""), rideString(""), rideInt(3), rideInt(0)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]rideType{rideString("x冬xqweqwe"), rideString("x冬xqw"), rideInt(0)}, false, rideInt(0)}, // unicode indexOf with zero offset
		{[]rideType{rideString("冬weqwe"), rideString("we"), rideInt(2)}, false, rideInt(4)},       // unicode indexOf with start offset
		{[]rideType{rideString(""), rideString("x冬x"), rideInt(1)}, false, rideUnit{}},            // unicode indexOf from empty string with offset
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("Hello")}, false, rideBytes("Hello")},
		{[]rideType{rideString("Привет")}, false, rideBytes("Привет")},
		{[]rideType{rideString("世界")}, false, rideBytes("世界")},
		{[]rideType{rideString("")}, false, rideBytes{}},
		{[]rideType{rideString(""), rideInt(3)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("abcde"), rideInt(2)}, false, rideString("abc")},
		{[]rideType{rideString("abcde"), rideInt(4)}, false, rideString("a")},
		{[]rideType{rideString("abcde"), rideInt(6)}, false, rideString("")},
		{[]rideType{rideString("abc"), rideInt(0)}, false, rideString("abc")},
		{[]rideType{rideString("abc"), rideInt(-4)}, false, rideString("abc")},
		{[]rideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]rideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]rideType{rideString("abc")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]rideType{rideString("x冬x"), rideInt(2)}, false, rideString("x")},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("abcde"), rideInt(2)}, false, rideString("de")},
		{[]rideType{rideString("abcde"), rideInt(4)}, false, rideString("bcde")},
		{[]rideType{rideString("abcde"), rideInt(6)}, false, rideString("abcde")},
		{[]rideType{rideString("abc"), rideInt(0)}, false, rideString("")},
		{[]rideType{rideString("abc"), rideInt(-4)}, false, rideString("")},
		{[]rideType{rideString(""), rideInt(0)}, false, rideString("")},
		{[]rideType{rideString(""), rideInt(3)}, false, rideString("")},
		{[]rideType{rideString("abc")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]rideType{rideString("x冬x"), rideInt(2)}, false, rideString("冬x")},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("abcdefg"), rideString("")}, false, rideList{rideString("a"), rideString("b"), rideString("c"), rideString("d"), rideString("e"), rideString("f"), rideString("g")}},
		{[]rideType{rideString("one two three four"), rideString(" ")}, false, rideList{rideString("one"), rideString("two"), rideString("three"), rideString("four")}},
		{[]rideType{rideString(""), rideString(" ")}, false, rideList{rideString("")}},
		{[]rideType{rideString(" "), rideString(" ")}, false, rideList{rideString(""), rideString("")}},
		{[]rideType{rideString(""), rideString("")}, false, rideList{}},
		{[]rideType{rideString(" "), rideString("")}, false, rideList{rideString(" ")}},
		{[]rideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]rideType{rideString("abc")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
		// scala tests from https://github.com/wavesplatform/Waves/pull/3367
		{[]rideType{rideString("strx冬x1;🤦;🤦strx冬x2;🤦strx冬x3"), rideString(";🤦")}, false, rideList{rideString("strx冬x1"), rideString(""), rideString("strx冬x2"), rideString("strx冬x3")}},
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

func BenchmarkSplitString(b *testing.B) {
	item := strings.Repeat("x", 31)
	list := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		list[i] = item
	}
	s := strings.Join(list, ",")
	args := []rideType{rideString(s), rideString(",")}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := splitString(nil, args...)
		require.NoError(b, err)
		require.NotNil(b, r)
	}
}

func TestSplit(t *testing.T) {
	for _, test := range []struct {
		s, sep    string
		len, size int
		fail      bool
		r         rideType
	}{
		{"1,2,3", ",", 5, 5, false, rideList{rideString("1"), rideString("2"), rideString("3")}},
		{"1,2,3", ",", 5, 3, false, rideList{rideString("1"), rideString("2"), rideString("3")}},
		{"1,2,3", ",", 3, 3, true, nil},
		{"1,2,3", ",", 5, 2, true, nil},
	} {
		r, err := split(test.s, test.sep, test.len, test.size)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func BenchmarkSplitStringV6(b *testing.B) {
	item := strings.Repeat("x", 24)
	list := make([]string, 20)
	for i := 0; i < 20; i++ {
		list[i] = item
	}
	s := strings.Join(list, ",")
	args := []rideType{rideString(s), rideString(",")}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := splitStringV6(nil, args...)
		require.NoError(b, err)
		require.NotNil(b, r)
	}
}

func BenchmarkSplitString4C(b *testing.B) {
	item := strings.Repeat("x", 59)
	list := make([]string, 100)
	for i := 0; i < 100; i++ {
		list[i] = item
	}
	s := strings.Join(list, ",")
	args := []rideType{rideString(s), rideString(",")}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := splitString4C(nil, args...)
		require.NoError(b, err)
		require.NotNil(b, r)
	}
}

func TestParseInt(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("123345")}, false, rideInt(123345)},
		{[]rideType{rideString("0")}, false, rideInt(0)},
		{[]rideType{rideString(fmt.Sprint(math.MaxInt64))}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideString(fmt.Sprint(math.MinInt64))}, false, rideInt(math.MinInt64)},
		{[]rideType{rideString("")}, false, rideUnit{}},
		{[]rideType{rideString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")}, false, rideUnit{}},
		{[]rideType{rideString("abc")}, false, rideUnit{}},
		{[]rideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("123345")}, false, rideInt(123345)},
		{[]rideType{rideString("0")}, false, rideInt(0)},
		{[]rideType{rideString(fmt.Sprint(math.MaxInt64))}, false, rideInt(math.MaxInt64)},
		{[]rideType{rideString(fmt.Sprint(math.MinInt64))}, false, rideInt(math.MinInt64)},
		{[]rideType{rideString("")}, true, nil},
		{[]rideType{rideString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")}, true, nil},
		{[]rideType{rideString("abc")}, true, nil},
		{[]rideType{rideString("abc"), rideInt(0)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe")}, false, rideInt(25)},
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("cafe")}, false, rideUnit{}},
		{[]rideType{rideString("")}, true, nil},
		{[]rideType{rideString(""), rideInt(3)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(30)}, false, rideInt(25)},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(25)}, false, rideInt(25)},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(10)}, false, rideInt(5)},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(5)}, false, rideInt(5)},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(4)}, false, rideUnit{}},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(0)}, false, rideUnit{}},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("bebe"), rideInt(-2)}, false, rideUnit{}},
		{[]rideType{rideString("aaa"), rideString("a"), rideInt(0)}, false, rideInt(0)},
		{[]rideType{rideString("aaa"), rideString("b"), rideInt(0)}, false, rideUnit{}},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("dead"), rideInt(11)}, false, rideInt(10)},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("dead"), rideInt(10)}, false, rideInt(10)},
		{[]rideType{rideString("cafe bebe dead beef cafe bebe"), rideString("dead"), rideInt(9)}, false, rideUnit{}},
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("brown"), rideInt(12)}, false, rideInt(6)},
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("fox"), rideInt(14)}, false, rideInt(12)},
		{[]rideType{rideString("quick brown fox jumps over the lazy dog"), rideString("fox"), rideInt(13)}, false, rideInt(12)},
		{[]rideType{rideString("")}, true, nil},
		{[]rideType{rideString(""), rideInt(3)}, true, nil},
		{[]rideType{rideString(""), rideString(""), rideInt(3), rideInt(0)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
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

func TestMkStringLoose(t *testing.T) {
	for _, test := range []struct {
		list      []rideType
		sep       string
		size, len int
		fail      bool
		r         string
	}{
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, ",", 5, 5, false, "1,2,3"},
		{[]rideType{rideString("1"), rideInt(2), rideString("3")}, ",", 5, 5, false, "1,2,3"},
		{[]rideType{rideString("1"), rideString("2"), rideBoolean(true)}, ",", 5, 5, true, ""},
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, ",", 3, 5, false, "1,2,3"},
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, ",", 2, 5, true, ""},
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, ",", 3, 3, true, ""},
	} {
		r, err := mkString(test.list, test.sep, test.size, test.len, looseStringList)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}

	}
}

func TestMkStringStrict(t *testing.T) {
	for _, test := range []struct {
		list      []rideType
		sep       string
		size, len int
		fail      bool
		r         string
	}{
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, ",", 5, 5, false, "1,2,3"},
		{[]rideType{rideString("1"), rideInt(2), rideString("3")}, ",", 5, 5, true, ""},
		{[]rideType{rideString("1"), rideString("2"), rideBoolean(true)}, ",", 5, 5, true, ""},
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, ",", 3, 5, false, "1,2,3"},
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, ",", 2, 5, true, ""},
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, ",", 3, 3, true, ""},
	} {
		r, err := mkString(test.list, test.sep, test.size, test.len, strictStringList)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}

	}
}

func BenchmarkMakeStringV6(b *testing.B) {
	item := "123456"
	list := make([]rideType, 70)
	for i := 0; i < 70; i++ {
		list[i] = rideString(item)
	}
	b.ResetTimer()
	args := []rideType{rideList(list), rideString(",")}
	for i := 0; i < b.N; i++ {
		r, err := makeStringV6(nil, args...)
		require.NoError(b, err)
		require.NotEmpty(b, r)
	}
}

func BenchmarkMakeString2C(b *testing.B) {
	item := strings.Repeat("x", 59)
	list := make([]rideType, 100)
	for i := 0; i < 100; i++ {
		list[i] = rideString(item)
	}
	args := []rideType{rideList(list), rideString(",")}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := makeString2C(nil, args...)
		require.NoError(b, err)
		require.NotEmpty(b, r)
	}
}

func TestMakeString(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideList{rideString("1"), rideString("2"), rideString("3")}, rideString(" ")}, false, rideString("1 2 3")},
		{[]rideType{rideList{rideString("one"), rideString("two"), rideString("three")}, rideString(", ")}, false, rideString("one, two, three")},
		{[]rideType{rideList{rideString("")}, rideString("")}, false, rideString("")},
		{[]rideType{rideList{}, rideString(",")}, false, rideString("")},
		{[]rideType{rideList{rideString("one"), rideInt(2), rideString("tree")}, rideString(", ")}, false, rideString("one, 2, tree")},
		{[]rideType{rideList{rideString("one"), rideBoolean(true), rideString("tree")}, rideString(", ")}, false, rideString("one, true, tree")},
		{[]rideType{rideString("")}, true, nil},
		{[]rideType{rideString(""), rideInt(3)}, true, nil},
		{[]rideType{rideString("1"), rideString("2"), rideString("3")}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
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
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideString("ride"), rideString("ide")}, false, rideBoolean(true)},
		{[]rideType{rideString("string"), rideString("substring")}, false, rideBoolean(false)},
		{[]rideType{rideString(""), rideString("")}, false, rideBoolean(true)},
		{[]rideType{rideString("ride"), rideString("")}, false, rideBoolean(true)},
		{[]rideType{rideString(""), rideString("ride")}, false, rideBoolean(false)},
		{[]rideType{rideString(""), rideInt(3)}, true, nil},
		{[]rideType{rideString(""), rideString(""), rideInt(3), rideInt(0)}, true, nil},
		{[]rideType{rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideString("x")}, true, nil},
		{[]rideType{rideInt(1)}, true, nil},
		{[]rideType{}, true, nil},
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
