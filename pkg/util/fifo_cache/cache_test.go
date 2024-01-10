package fifo_cache_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

type impl struct {
	val int
	key string
}

func (a impl) Key() string {
	return a.key
}

func (a impl) Value() int {
	return a.val
}

func TestFIFOCache(t *testing.T) {

	a := fifo_cache.New[string, int](1)
	require.Equal(t, 0, a.Len())

	a.Add(impl{5, "5"})
	require.Equal(t, 1, a.Len())

	_, ok := a.Get("H")
	require.False(t, ok)

	val, ok := a.Get("5")
	require.True(t, ok)
	require.Equal(t, 5, val)

	require.True(t, a.Exists("5"))

	a.Add(impl{7, "7"})
	require.False(t, a.Exists("5"))
	require.True(t, a.Exists("7"))

	require.Equal(t, 1, a.Cap())

}

func TestFIFOCache_Add2(t *testing.T) {
	a := fifo_cache.New[string, int](2)
	a.Add2("5", 5)
	require.Equal(t, 1, a.Len())
	a.Add2("5", 6)
	require.Equal(t, 1, a.Len())
}
