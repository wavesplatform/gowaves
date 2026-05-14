package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func TestAppAuth(t *testing.T) {
	cfg := &settings.BlockchainSettings{
		FunctionalitySettings: settings.FunctionalitySettings{
			GenerationPeriod: 0,
		},
	}
	app, _ := NewApp("apiKey", nil, services.Services{}, cfg)
	require.Error(t, app.checkAuth("bla"))
	require.NoError(t, app.checkAuth("apiKey"))
}
