package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestAppAuth(t *testing.T) {
	app, _ := NewApp("apiKey", nil, services.Services{})
	require.Error(t, app.checkAuth("bla"))
	require.NoError(t, app.checkAuth("apiKey"))
}
