package errs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtend(t *testing.T) {
	require.EqualError(t, Extend(errors.New("a"), "b"), "b: a")
}
