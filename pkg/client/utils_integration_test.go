//go:build integration
// +build integration

package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUtilsIntegration_Seed(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}
	client, _ := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  apiKey,
	})
	_, resp, err :=
		client.Utils.Seed(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUtilsIntegration_HashSecure(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}
	client, _ := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  apiKey,
	})
	_, resp, err :=
		client.Utils.HashSecure(context.Background(), "xxx")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUtilsIntegration_HashFast(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}
	client, _ := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  apiKey,
	})
	_, resp, err :=
		client.Utils.HashFast(context.Background(), "xxx")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUtilsIntegration_Time(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}
	client, _ := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  apiKey,
	})
	_, resp, err :=
		client.Utils.Time(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUtils_SeedByLength2(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}
	client, _ := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  apiKey,
	})
	_, resp, err :=
		client.Utils.SeedByLength(context.Background(), 44)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUtilsIntegration_ScriptCompile(t *testing.T) {
	client, _ := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	_, resp, err :=
		client.Utils.ScriptCompile(context.Background(), "1 == 1")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUtilsIntegration_ScriptEstimate(t *testing.T) {
	client, _ := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	_, resp, err :=
		client.Utils.ScriptEstimate(context.Background(), "base64:AQa3b8tH")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}
