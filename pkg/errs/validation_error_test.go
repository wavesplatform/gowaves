package errs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockValidationError(t *testing.T) {
	require.True(t, IsValidationError(NewBlockValidationError("")))
}

func TestIsValidationError(t *testing.T) {
	require.False(t, IsValidationError(nil))
}
