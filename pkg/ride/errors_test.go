package ride_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

func TestIsThrowErr(t *testing.T) {
	require.False(t, ride.IsThrowErr(errors.New("")))
	require.True(t, ride.IsThrowErr(ride.NewThrowError("")))
}
