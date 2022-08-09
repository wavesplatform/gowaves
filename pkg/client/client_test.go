package client

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockHttpRequest struct {
	Body       io.ReadCloser
	StatusCode int
}

func NewMockHttpRequestFromString(s string, statusCode int) *MockHttpRequest {
	return &MockHttpRequest{
		Body:       io.NopCloser(strings.NewReader(s)),
		StatusCode: statusCode,
	}
}

func (a *MockHttpRequest) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Request:    req,
		StatusCode: a.StatusCode,
		Body:       a.Body,
	}, nil
}

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

	req, _ := http.NewRequest("GET", "https://google.com", nil)

	resp, err := client.Do(cancel, req, nil)
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestMockHttpRequest(t *testing.T) {
	url := "https://github.com/wavesplatform/gowaves"
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.Header.Set("ApiKey", "123456")

	rs := NewMockHttpRequestFromString("", 200)
	resp, err := rs.Do(req)
	require.Nil(t, err)
	assert.Equal(t, url, resp.Request.URL.String())
	assert.Equal(t, "123456", resp.Request.Header.Get("ApiKey"))
}

func TestJoinUrl(t *testing.T) {
	url, err := joinUrl("https://wavesplatform.com", "path")
	require.NoError(t, err)
	assert.Equal(t, "https://wavesplatform.com/path", url.String())

	url, err = joinUrl("https://clinton.vostokservices.com/node-0", "/consensus/basetarget")
	require.NoError(t, err)
	assert.Equal(t, "https://clinton.vostokservices.com/node-0/consensus/basetarget", url.String())
}

func client(t *testing.T, doer Doer) *Client {
	url, _ := os.LookupEnv("GOWAVES_CLIENT_URL")
	if len(url) > 0 {
		client, err := NewClient(Options{
			BaseUrl: url,
		})
		if err != nil {
			t.Fatal(err)
		}
		return client
	}
	client, err := NewClient(Options{
		BaseUrl: "https://nodes-stagenet.wavesnodes.com",
		Client:  doer,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}
