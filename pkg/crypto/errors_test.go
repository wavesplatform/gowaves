package crypto

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestIncorrectLengthError(t *testing.T) {
	incorrectLenErr := NewIncorrectLengthError("some name", 5, 32)
	require.Equal(t, "incorrect some name length 5, expected 32", incorrectLenErr.Error())

	wrappingErr := errors.Wrap(incorrectLenErr, "message")
	require.Equal(t, "message: incorrect some name length 5, expected 32", wrappingErr.Error())

	var extractedIncorrectLenErr IncorrectLengthError
	require.True(t, errors.As(wrappingErr, &extractedIncorrectLenErr))
	require.Equal(t, "some name", extractedIncorrectLenErr.Name)
	require.Equal(t, 5, extractedIncorrectLenErr.Len)
	require.Equal(t, 32, extractedIncorrectLenErr.ExpectedLen)
}
