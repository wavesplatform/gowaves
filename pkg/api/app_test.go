package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppAuth(t *testing.T) {
	app, _ := NewApp("apiKey", nil, nil, nil, nil)
	require.Error(t, app.checkAuth("bla"))
	require.NoError(t, app.checkAuth("apiKey"))
}
