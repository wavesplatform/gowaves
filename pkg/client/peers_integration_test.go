// +build integration

package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeersIntegration_All(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	client, err := NewClient(Options{
		BaseUrl: "https://testnodes.wavesnodes.com",
		ApiKey:  apiKey,
	})
	require.Nil(t, err)
	_, resp, err :=
		client.Peers.All(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPeersIntegration_Connected(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	client, err := NewClient(Options{
		BaseUrl: "https://testnodes.wavesnodes.com",
		ApiKey:  apiKey,
	})
	require.Nil(t, err)
	_, resp, err :=
		client.Peers.Connected(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPeersIntegration_Blacklisted(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	client, err := NewClient(Options{
		BaseUrl: "https://testnodes.wavesnodes.com",
		ApiKey:  apiKey,
	})
	require.Nil(t, err)
	_, resp, err :=
		client.Peers.Blacklisted(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPeersIntegration_Suspended(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	client, err := NewClient(Options{
		BaseUrl: "https://testnodes.wavesnodes.com",
		ApiKey:  apiKey,
	})
	require.Nil(t, err)
	_, resp, err :=
		client.Peers.Blacklisted(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPeersIntegration_Connect(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	client, err := NewClient(Options{
		BaseUrl: "https://testnodes.wavesnodes.com",
		ApiKey:  apiKey,
	})
	require.Nil(t, err)
	_, resp, err :=
		client.Peers.Connect(context.Background(), "localhost", 6868)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPeersIntegration_ClearBlacklist(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	client, err := NewClient(Options{
		BaseUrl: "https://testnodes.wavesnodes.com",
		ApiKey:  apiKey,
	})
	require.Nil(t, err)
	_, resp, err :=
		client.Peers.ClearBlacklist(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}
