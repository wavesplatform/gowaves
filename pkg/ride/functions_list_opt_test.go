package ride //nolint:testpackage // White-box tests of the unexported list and tuple helpers.

import (
	"fmt"
	stdmath "math"
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/math"
)

// referenceMedian is the median algorithm used before the optimization.
func referenceMedian(values []int) rideInt {
	items := slices.Clone(values)
	sort.Ints(items)
	half := len(items) / 2
	if len(items)%2 == 1 {
		return rideInt(items[half])
	}
	return rideInt(math.FloorDiv(int64(items[half-1])+int64(items[half]), 2))
}

func referenceMinMax(values []int) (rideInt, rideInt) {
	return rideInt(slices.Min(values)), rideInt(slices.Max(values))
}

func TestMedianMinMaxMatchReference(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	sizes := []int{2, 3, 4, 5, medianStackBufferSize - 1, medianStackBufferSize, medianStackBufferSize + 1, 100, 999,
		maxListSize}
	gens := map[string]func() rideInt{
		"small":    func() rideInt { return rideInt(rnd.IntN(10) - 5) },
		"large":    func() rideInt { return rideInt(rnd.Int64()) },
		"negative": func() rideInt { return rideInt(-rnd.Int64()) },
		"extremes": func() rideInt {
			return []rideInt{stdmath.MinInt64, stdmath.MaxInt64, -1, 0, 1}[rnd.IntN(5)]
		},
	}
	for name, gen := range gens {
		for _, size := range sizes {
			values := make([]int, size)
			list := make(rideList, size)
			for i := range list {
				v := gen()
				values[i] = int(v)
				list[i] = v
			}
			orig := make(rideList, size)
			copy(orig, list)
			r, err := median(nil, list)
			require.NoError(t, err)
			assert.Equal(t, referenceMedian(values), r, "median %s/%d", name, size)
			mn, mx := referenceMinMax(values)
			r, err = listMin(nil, list)
			require.NoError(t, err)
			assert.Equal(t, mn, r, "min %s/%d", name, size)
			r, err = listMax(nil, list)
			require.NoError(t, err)
			assert.Equal(t, mx, r, "max %s/%d", name, size)
			assert.Equal(t, orig, list, "argument list must stay unchanged")
		}
	}
	r, err := listMin(nil, rideList{rideInt(7)})
	require.NoError(t, err)
	assert.Equal(t, rideInt(7), r)
	r, err = listMax(nil, rideList{rideInt(-7)})
	require.NoError(t, err)
	assert.Equal(t, rideInt(-7), r)
}

func TestMedianMinMaxErrors(t *testing.T) {
	large := make(rideList, maxListSize+1)
	for i := range large {
		large[i] = rideInt(i)
	}
	mixed := rideList{rideInt(1), rideString("2"), rideBoolean(true)}
	mixedLarge := make(rideList, medianStackBufferSize+1)
	for i := range mixedLarge {
		mixedLarge[i] = rideInt(i)
	}
	mixedLarge[medianStackBufferSize] = rideUnit{}
	for _, test := range []struct {
		fn   rideFunction
		args []rideType
		err  string
	}{
		{median, []rideType{rideList{}}, "median: invalid list size 0"},
		{median, []rideType{rideList{rideInt(1)}}, "median: invalid list size 1"},
		{median, []rideType{large}, "median: invalid list size 1001"},
		{median, []rideType{mixed}, "median: unexpected type of list element 'String'"},
		{median, []rideType{mixedLarge}, "median: unexpected type of list element 'Unit'"},
		{median, []rideType{rideInt(1)}, "median: unexpected type of argument 'Int'"},
		{listMax, []rideType{rideList{}}, "listMax: invalid list size 0"},
		{listMax, []rideType{large}, "listMax: invalid list size 1001"},
		{listMax, []rideType{mixed}, "listMax: unexpected type of list element 'String'"},
		{listMax, []rideType{rideList{rideString("1")}}, "listMax: unexpected type of list element 'String'"},
		{listMin, []rideType{rideList{}}, "listMin: invalid list size 0"},
		{listMin, []rideType{large}, "listMin: invalid list size 1001"},
		{listMin, []rideType{mixed}, "listMin: unexpected type of list element 'String'"},
		{listMin, []rideType{mixedLarge}, "listMin: unexpected type of list element 'Unit'"},
	} {
		_, err := test.fn(nil, test.args...)
		assert.EqualError(t, err, test.err)
	}
}

func TestListSearchFunctions(t *testing.T) {
	tuple := tuple2{el1: rideInt(1), el2: rideString("1")}
	list := rideList{rideInt(1), rideString("1"), rideBoolean(true), rideInt(1), tuple, rideUnit{}, rideString("1")}
	for _, test := range []struct {
		list    rideList
		e       rideType
		first   rideType
		last    rideType
		contain rideBoolean
	}{
		{rideList{}, rideInt(1), rideUnit{}, rideUnit{}, false},
		{list, rideInt(1), rideInt(0), rideInt(3), true},
		{list, rideString("1"), rideInt(1), rideInt(6), true},
		{list, rideBoolean(true), rideInt(2), rideInt(2), true},
		{list, rideBoolean(false), rideUnit{}, rideUnit{}, false},
		{list, rideUnit{}, rideInt(5), rideInt(5), true},
		{list, tuple2{el1: rideInt(1), el2: rideString("1")}, rideInt(4), rideInt(4), true},
		{list, tuple2{el1: rideString("1"), el2: rideInt(1)}, rideUnit{}, rideUnit{}, false},
		{list, rideByteVector("1"), rideUnit{}, rideUnit{}, false},
		{list, rideInt(2), rideUnit{}, rideUnit{}, false},
	} {
		r, err := indexOfList(nil, test.list, test.e)
		require.NoError(t, err)
		assert.Equal(t, test.first, r, "indexOf %s", test.e)
		r, err = lastIndexOfList(nil, test.list, test.e)
		require.NoError(t, err)
		assert.Equal(t, test.last, r, "lastIndexOf %s", test.e)
		r, err = containsElement(nil, test.list, test.e)
		require.NoError(t, err)
		assert.Equal(t, test.contain, r, "contains %s", test.e)
	}
	large := make(rideList, maxListSize+1)
	for i := range large {
		large[i] = rideInt(i)
	}
	_, err := indexOfList(nil, large, rideInt(1))
	assert.EqualError(t, err, "indexOfList: list size exceeds 1000 elements")
	_, err = lastIndexOfList(nil, large, rideInt(1))
	assert.EqualError(t, err, "lastIndexOfList: list size exceeds 1000 elements")
	r, err := containsElement(nil, large, rideInt(maxListSize))
	require.NoError(t, err)
	assert.Equal(t, rideBoolean(true), r)
	_, err = containsElement(nil, rideInt(1), rideInt(1))
	assert.EqualError(t, err, "containsElement: unexpected type of argument 1 'Int'")
}

// referenceTupleInstanceOf builds the tuple type name the way it was built before the optimization.
func referenceTupleInstanceOf(elements ...rideType) string {
	names := make([]string, len(elements))
	for i, e := range elements {
		names[i] = e.instanceOf()
	}
	return fmt.Sprintf("(%s)", strings.Join(names, ", "))
}

func TestTupleInstanceOfAndEq(t *testing.T) {
	inner := tuple2{el1: rideInt(1), el2: rideString("a")}
	innerOther := tuple2{el1: rideString("a"), el2: rideInt(1)}
	t3 := tuple3{el1: inner, el2: rideList{rideInt(1)}, el3: rideUnit{}}
	assert.Equal(t, "(Int, String)", inner.instanceOf())
	assert.Equal(t, referenceTupleInstanceOf(inner, rideList{rideInt(1)}, rideUnit{}), t3.instanceOf())
	assert.Equal(t, "((Int, String), List[Any], Unit)", t3.instanceOf())

	elements := make([]rideType, 22)
	for i := range elements {
		elements[i] = rideInt(i)
	}
	elements[21] = inner
	t22, err := newTuple22(nil, elements...)
	require.NoError(t, err)
	assert.Equal(t, referenceTupleInstanceOf(elements...), t22.instanceOf())

	for _, test := range []struct {
		a, b rideType
		eq   bool
	}{
		{inner, tuple2{el1: rideInt(1), el2: rideString("a")}, true},
		{inner, tuple2{el1: rideInt(2), el2: rideString("a")}, false},
		{inner, innerOther, false},
		{inner, tuple3{el1: rideInt(1), el2: rideString("a"), el3: rideUnit{}}, false},
		{inner, rideString("(Int, String)"), false},
		{inner, rideList{rideInt(1), rideString("a")}, false},
		{t3, tuple3{el1: tuple2{el1: rideInt(1), el2: rideString("a")}, el2: rideList{rideInt(1)}, el3: rideUnit{}}, true},
		{t3, tuple3{el1: innerOther, el2: rideList{rideInt(1)}, el3: rideUnit{}}, false},
		{t3, tuple3{el1: inner, el2: rideList{rideInt(2)}, el3: rideUnit{}}, false},
		{t3, tuple3{el1: inner, el2: rideList{rideInt(1)}, el3: rideInt(0)}, false},
		{tuple2{el1: rideUnit{}, el2: rideInt(0)}, tuple2{el1: rideInt(0), el2: rideUnit{}}, false},
	} {
		// The result must be the same as comparing the tuple type names first, as it was done before.
		expected := test.a.instanceOf() == test.b.instanceOf() && test.eq
		assert.Equal(t, test.eq, expected)
		assert.Equal(t, test.eq, test.a.eq(test.b), "%s == %s", test.a, test.b)
		assert.Equal(t, test.eq, test.b.eq(test.a), "%s == %s", test.b, test.a)
		r, eqErr := eq(nil, test.a, test.b)
		require.NoError(t, eqErr)
		assert.Equal(t, rideBoolean(test.eq), r)
	}
	r, err := getType(nil, t3)
	require.NoError(t, err)
	assert.Equal(t, rideString("((Int, String), List[Any], Unit)"), r)
}
