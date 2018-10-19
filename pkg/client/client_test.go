package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestClient_GetOptions(t *testing.T) {
	client, err := NewClient()
	require.Nil(t, err)
	assert.Equal(t, "https://nodes.wavesnodes.com", client.options.BaseUrl)

	client, err = NewClient(Options{BaseUrl: "URL"})
	require.Nil(t, err)
	assert.Equal(t, "URL", client.options.BaseUrl)
}

func TestClient_Do(t *testing.T) {
	client, err := NewClient()
	require.Nil(t, err)
	bg := context.Background()
	cancel, fn := context.WithCancel(bg)
	fn()

	req, _ := http.NewRequest("GET", "http://google.com", nil)

	resp, err := client.Do(cancel, req, nil)
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
