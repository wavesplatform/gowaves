package lock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMutex_Lock1(t *testing.T) {
	l := NewMutex()
	require.NoError(t, l.Lock(1*time.Millisecond))
	require.Error(t, l.Lock(1*time.Millisecond))
}

func TestMutex_Lock2(t *testing.T) {
	l := NewMutex()
	require.NoError(t, l.Lock(1*time.Millisecond))
	l.Unlock()
	require.NoError(t, l.Lock(1*time.Millisecond))
}

func TestMutex_Lock3(t *testing.T) {
	l := NewMutex()
	require.NoError(t, l.Lock(1*time.Millisecond))
	require.Error(t, l.Lock(1*time.Millisecond))

	l.Unlock()
	require.NoError(t, l.Lock(1*time.Millisecond))
}
