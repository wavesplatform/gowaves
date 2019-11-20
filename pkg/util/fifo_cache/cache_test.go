package fifo_cache_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

type impl struct {
	val int
	key []byte
}

func (a impl) Key() []byte {
	return a.key
}

func (a impl) Value() interface{} {
	return a.val
}

func TestFIFOCache(t *testing.T) {

	a := fifo_cache.New(1)
	require.Equal(t, 0, a.Len())

	a.Add(impl{5, []byte("5")})
	require.Equal(t, 1, a.Len())

	_, ok := a.Get([]byte{'H'})
	require.False(t, ok)

	val, ok := a.Get([]byte{'5'})
	require.True(t, ok)
	require.Equal(t, 5, val)

	require.True(t, a.Exists([]byte{'5'}))

	a.Add(impl{7, []byte("7")})
	require.False(t, a.Exists([]byte{'5'}))
	require.True(t, a.Exists([]byte{'7'}))

	require.Equal(t, 1, a.Cap())

}

func TestFIFOCache_Add2(t *testing.T) {
	a := fifo_cache.New(2)
	a.Add2([]byte("5"), 5)
	require.Equal(t, 1, a.Len())
	a.Add2([]byte("5"), 6)
	require.Equal(t, 1, a.Len())
}
