package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

// Must create new client or panic
func mustClient(options ...Options) *Client {
	client, err := NewClient(options...)
	if err != nil {
		panic(err)
	}
	return client
}

func TestClient_GetOptions(t *testing.T) {
	client := mustClient()
	assert.Equal(t, "https://nodes.wavesnodes.com", client.options.BaseUrl)

	client = mustClient(Options{BaseUrl: "URL"})
	assert.Equal(t, "URL", client.options.BaseUrl)
}

func TestClient_Do(t *testing.T) {
	client := mustClient()
	bg := context.Background()
	cancel, fn := context.WithCancel(bg)
	fn()

	req, _ := http.NewRequest("GET", "http://google.com", nil)

	resp, err := client.Do(cancel, req, nil)
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
