package api

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestUnixMillis(t *testing.T) {
	expected := time.Now()
	ts := unixMillis(expected)

	require.Equal(t, expected.UnixNano()/1_000_000, ts)
}

func TestFromUnixMillis(t *testing.T) {
	now := time.Now()

	expected := now.Truncate(time.Millisecond)
	millis := expected.UnixNano() / 1_000_000

	actual := fromUnixMillis(millis)

	require.True(t, expected.Equal(actual))
	require.Equal(t, expected.String(), actual.String())
}
