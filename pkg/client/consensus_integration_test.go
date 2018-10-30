// +build integration

package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestConsensusIntegration_GeneratingBalance(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	addr, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, _ := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  apiKey,
	})
	_, resp, err :=
		client.Consensus.GeneratingBalance(context.Background(), addr)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestConsensusIntegration_GenerationSignatureByBlock(t *testing.T) {
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
		client.Consensus.GenerationSignatureByBlock(context.Background(), "3Z9W6dX3iAqyhv2gsE1WRRd5yLYdtjojLzNSXEFZNuVs21hkuNUmhqTNLqrcGnERJMaPtrfvag4AjQpjykvQM13a")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestConsensusIntegration_BaseTargetByBlock(t *testing.T) {
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
		client.Consensus.BaseTargetByBlock(context.Background(), "3Z9W6dX3iAqyhv2gsE1WRRd5yLYdtjojLzNSXEFZNuVs21hkuNUmhqTNLqrcGnERJMaPtrfvag4AjQpjykvQM13a")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestConsensusIntegration_BaseTarget(t *testing.T) {
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
		client.Consensus.BaseTarget(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestConsensusIntegration_Algo(t *testing.T) {
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
		client.Consensus.Algo(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestConsensusIntegration_GenerationSignature(t *testing.T) {
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
		client.Consensus.GenerationSignature(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}
