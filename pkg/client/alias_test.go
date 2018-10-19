package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestNewAlias(t *testing.T) {
	assert.NotNil(t, NewAlias(defaultOptions))
}

func TestAlias_Get(t *testing.T) {
	client, err := NewClient(Options{BaseUrl: "https://testnode1.wavesnodes.com"})
	require.Nil(t, err)
	body, resp, err :=
		client.Alias.Get(context.Background(), "frozen")

	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.NotEqual(t, "", body)
}

func TestAlias_GetByAddress(t *testing.T) {
	address, err := proto.NewAddressFromString("3MvxxSiCqCA5gvtp3EgjEukpWg4HccdLAMh")
	assert.Nil(t, err)

	client, err := NewClient(Options{BaseUrl: "https://testnode1.wavesnodes.com"})
	require.Nil(t, err)
	body, resp, err :=
		client.Alias.GetByAddress(context.Background(), address)

	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
}
