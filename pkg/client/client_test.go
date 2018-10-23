package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type MockHttpRequest struct {
	Body       io.ReadCloser
	StatusCode int
}

func NewMockHttpRequestFromString(s string, statusCode int) *MockHttpRequest {
	return &MockHttpRequest{
		Body:       ioutil.NopCloser(strings.NewReader(s)),
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

	req, _ := http.NewRequest("GET", "http://google.com", nil)

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
