package ride //nolint:testpackage // White-box tests of the unexported string helpers.

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// referenceRunesDrop is the original implementation of runesDrop.
func referenceRunesDrop(s string, n int) string {
	runes := []rune(s)
	out := make([]rune, len(runes)-n)
	copy(out, runes[n:])
	return string(out)
}

// referenceRunesTake is the original implementation of runesTake.
func referenceRunesTake(s string, n int) string {
	out := make([]rune, n)
	copy(out, []rune(s)[:n])
	return string(out)
}

// referenceSplit is the original implementation of split.
func referenceSplit(s, sep string, stringLengthLimit, resultListSizeLimit int) (rideList, error) {
	if sl := len(s); sl > stringLengthLimit {
		return nil, errors.Errorf("string length %d exceeds string length limit %d", sl, stringLengthLimit)
	}
	res := strings.Split(s, sep)
	rl := len(res)
	if rl > resultListSizeLimit {
		return nil, errors.Errorf("list size %d exceeds list size limit %d", rl, resultListSizeLimit)
	}
	r := make(rideList, rl)
	for i, s := range res {
		r[i] = rideString(s)
	}
	return r, nil
}

func testStrings() []string {
	return []string{
		"",
		"a",
		"abc",
		"quick brown fox jumps over the lazy dog",
		"привет мир",
		"x😀y😀z",
		"日本語のテキスト",
		"mixed ascii и кириллица 😀",
		"\xff",
		"ab\xffcd",
		"\xe2\x82",          // Truncated 3-byte sequence.
		"ok\xe2\x82",        // Truncated sequence at the end.
		"\xed\xa0\x80 tail", // Encoded surrogate, invalid UTF-8.
		"при\xc0вет",        // Overlong encoding in the middle.
		"😀\xf0\x9f\x98 end", // Truncated 4-byte sequence.
		"a\uFFFDb",          // Valid replacement character.
	}
}

func TestRunesTakeDropMatchReference(t *testing.T) {
	for _, s := range testStrings() {
		l := utf8.RuneCountInString(s)
		for n := 0; n <= l; n++ {
			assert.Equal(t, referenceRunesTake(s, n), runesTake(s, n), "take %q %d", s, n)
			assert.Equal(t, referenceRunesDrop(s, n), runesDrop(s, n), "drop %q %d", s, n)
		}
		for _, n := range []int{-100, -1, 0, 1, l - 1, l, l + 1, 1000} {
			assert.Equal(t, rideString(referenceRunesTake(s, max(min(n, l), 0))), takeRideString(s, n))
			assert.Equal(t, rideString(referenceRunesDrop(s, max(min(n, l), 0))), dropRideString(s, n))
		}
	}
}

func TestRunesOffset(t *testing.T) {
	for _, test := range []struct {
		s   string
		n   int
		off int
	}{
		{"", 0, 0}, {"", 5, 0}, {"abc", -1, 0}, {"abc", 0, 0}, {"abc", 2, 2}, {"abc", 3, 3}, {"abc", 4, 3},
		{"привет", 1, 2}, {"привет", 6, 12}, {"x😀y", 2, 5}, {"x😀y", 3, 6},
	} {
		assert.Equal(t, test.off, runesOffset(test.s, test.n), "%q %d", test.s, test.n)
	}
}

func TestStringFunctionsMultibyteAndInvalidUTF8(t *testing.T) {
	for _, s := range testStrings() {
		l := utf8.RuneCountInString(s)
		for _, n := range []int{-1, 0, 1, l / 2, l - 1, l, l + 1} {
			args := []rideType{rideString(s), rideInt(n)}
			c := max(min(n, l), 0)
			r, err := takeString(newTestEnv(t).withTakeStringV5().toEnv(), args...)
			require.NoError(t, err)
			assert.Equal(t, rideString(referenceRunesTake(s, c)), r)
			r, err = dropString(nil, args...)
			require.NoError(t, err)
			assert.Equal(t, rideString(referenceRunesDrop(s, c)), r)
			r, err = takeRightString(nil, args...)
			require.NoError(t, err)
			assert.Equal(t, rideString(referenceRunesDrop(s, max(min(l-n, l), 0))), r)
			r, err = dropRightString(nil, args...)
			require.NoError(t, err)
			assert.Equal(t, rideString(referenceRunesTake(s, max(min(l-n, l), 0))), r)
		}
	}
}

func TestIndexOfSubstringWithOffsetMultibyte(t *testing.T) {
	reference := func(s1, s2 string, n int) rideType {
		if n < 0 || n > utf8.RuneCountInString(s1) {
			return rideUnit{}
		}
		i := runesIndex(referenceRunesDrop(s1, n), s2)
		if i == -1 {
			return rideUnit{}
		}
		return rideInt(i + n)
	}
	subs := []string{"", "a", "b", "мир", "😀", "�", "z", "cd", "end", "тек"}
	for _, s := range testStrings() {
		l := utf8.RuneCountInString(s)
		for _, sub := range subs {
			for n := -1; n <= l+1; n++ {
				r, err := indexOfSubstringWithOffset(nil, rideString(s), rideString(sub), rideInt(n))
				require.NoError(t, err)
				assert.Equal(t, reference(s, sub, n), r, "%q %q %d", s, sub, n)
			}
		}
	}
	r, err := indexOfSubstringWithOffset(nil, rideString("x😀y😀z"), rideString("😀"), rideInt(2))
	require.NoError(t, err)
	assert.Equal(t, rideInt(3), r)
}

func TestSplitMatchesReference(t *testing.T) {
	for _, s := range append(testStrings(), ",", ",,", "a,b,,c,", ", a, b, ", "a, b, c", "😀,😀") {
		for _, sep := range []string{"", ",", ", ", "a", "😀", "\xff", "не", "long separator"} {
			for _, lim := range []int{0, 1, 2, 3, 100} {
				exp, expErr := referenceSplit(s, sep, 1000, lim)
				r, err := split(s, sep, 1000, lim)
				if expErr != nil {
					require.EqualError(t, err, expErr.Error())
					continue
				}
				require.NoError(t, err)
				assert.Equal(t, exp, r, "%q %q", s, sep)
			}
		}
	}
	_, err := split(strings.Repeat("a", 11), ",", 10, 100)
	assert.EqualError(t, err, "string length 11 exceeds string length limit 10")
}

func TestMkStringStrictMatchesMkString(t *testing.T) {
	lists := []rideList{
		{},
		{rideString("")},
		{rideString("one")},
		{rideString(strings.Repeat("x", 20))},
		{rideString("a"), rideString("b")},
		{rideString("при"), rideString(""), rideString("вет"), rideString("😀")},
		{rideString("a"), rideInt(1)},
		{rideInt(1), rideString("a")},
		{rideString("a"), rideByteVector{1, 2}, rideString("c")},
	}
	for _, list := range lists {
		for _, sep := range []string{"", ",", ", ", "разделитель"} {
			for _, sizeLimit := range []int{0, 1, 2, 10} {
				for _, lengthLimit := range []int{0, 1, 5, 10, 100} {
					exp, expErr := mkString(list, sep, sizeLimit, lengthLimit, strictStringList)
					r, err := mkStringStrict(list, sep, sizeLimit, lengthLimit)
					if expErr != nil {
						require.EqualError(t, err, expErr.Error())
						continue
					}
					require.NoError(t, err)
					assert.Equal(t, exp, r)
				}
			}
		}
	}
}

func TestMakeStringV6Limits(t *testing.T) {
	list := make(rideList, 70)
	for i := range list {
		list[i] = rideString("abcdef")
	}
	r, err := makeStringV6(nil, list, rideString(","))
	require.NoError(t, err)
	assert.Equal(t, rideString(strings.Repeat("abcdef,", 69)+"abcdef"), r)
	longer := append(slices.Clone(list), rideString("x"))
	_, err = makeStringV6(nil, longer, rideString(","))
	assert.EqualError(t, err, "makeStringV6: list size 71 exceeds size limit 70")
	list[0] = rideString(strings.Repeat("y", 100))
	_, err = makeStringV6(nil, list, rideString(","))
	assert.EqualError(t, err, "makeStringV6: result length 583 exceeds maximum allowed 500")
	list[1] = rideInt(1)
	_, err = makeStringV6(nil, list, rideString(","))
	assert.EqualError(t, err, "makeStringV6: unexpected list item type 'Int'")
}

func TestMakeString2CAnd11CLimits(t *testing.T) {
	for _, tc := range []struct {
		name        string
		fn          rideFunction
		sizeLimit   int
		lengthLimit int
	}{
		{"makeString2C", makeString2C, 100, 6000},
		{"makeString11C", makeString11C, 1000, 32767},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list := make(rideList, tc.sizeLimit)
			for i := range list {
				list[i] = rideString("a")
			}
			r, err := tc.fn(nil, list, rideString(","))
			require.NoError(t, err)
			assert.Equal(t, rideString(strings.Repeat("a,", tc.sizeLimit-1)+"a"), r)
			_, err = tc.fn(nil, append(slices.Clone(list), rideString("a")), rideString(","))
			assert.EqualError(t, err, fmt.Sprintf("makeString: list size %d exceeds size limit %d",
				tc.sizeLimit+1, tc.sizeLimit))
			// The result of exactly the length limit is allowed, one byte more is not.
			fill := tc.lengthLimit - (2*tc.sizeLimit - 1)
			list[0] = rideString(strings.Repeat("a", fill+1))
			r, err = tc.fn(nil, list, rideString(","))
			require.NoError(t, err)
			rs, ok := r.(rideString)
			require.True(t, ok)
			assert.Len(t, string(rs), tc.lengthLimit)
			list[0] = rideString(strings.Repeat("a", fill+2))
			_, err = tc.fn(nil, list, rideString(","))
			assert.EqualError(t, err, fmt.Sprintf("makeString: result length %d exceeds maximum allowed %d",
				tc.lengthLimit+1, tc.lengthLimit))
		})
	}
}

func TestNormalizeDigitsFastPath(t *testing.T) {
	for _, test := range []struct {
		s   string
		r   string
		err bool
	}{
		{"", "", false},
		{"0", "0", false},
		{"+123", "+123", false},
		{"-9223372036854775808", "-9223372036854775808", false},
		{"12-3", "", true},
		{"--1", "", true},
		{"1 2", "", true},
		{"١٢٣", "123", false},
		{"-١٢٣", "-123", false},
		{"1٢3", "123", false},
		{"𝟏", "", true},
	} {
		r, err := normalizeDigits(test.s)
		if test.err {
			assert.Error(t, err, test.s)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.r, r)
	}
	for _, test := range []struct {
		s string
		r rideType
	}{
		{"+0", rideInt(0)}, {"-1", rideInt(-1)}, {"9223372036854775807", rideInt(9223372036854775807)},
		{"9223372036854775808", rideUnit{}}, {"", rideUnit{}}, {"+", rideUnit{}}, {"1-", rideUnit{}},
	} {
		r, err := parseInt(nil, rideString(test.s))
		require.NoError(t, err)
		assert.Equal(t, test.r, r, test.s)
	}
}
