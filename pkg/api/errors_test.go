package api

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBadRequest(t *testing.T) {
	err := BadRequestError{errors.New("bad request")}
	require.Error(t, err)
}

func TestAuthError(t *testing.T) {
	err := AuthError{errors.New("bad auth")}
	require.Error(t, err)
}
