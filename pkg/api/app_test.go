package api

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAppAuth(t *testing.T) {
	app, _ := NewApp("apiKey", nil)
	require.Error(t, app.checkAuth("bla"))
	require.NoError(t, app.checkAuth("apiKey"))
}
