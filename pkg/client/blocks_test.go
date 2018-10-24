package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlocks_HeightBySignature(t *testing.T) {
	client, err := NewClient(Options{BaseUrl: "https://testnode1.wavesnodes.com"})
	require.Nil(t, err)
	body, resp, err :=
		client.Blocks.HeightBySignature(
			context.Background(),
			"2TXfMcQNHJVmkbNoznZrFRLaQHiBayFV9mzxt4VJkyXmxe9aGNn5A2unzUX4M2tqiHEfaWdfCBBo8zJQQpFrCKUY")

	if err != nil {
		t.Fatalf("expected nil, found %+v", err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(379627), body.Height)
}

func TestBlocks_HeadersAt(t *testing.T) {
	client, err := NewClient(Options{BaseUrl: "https://testnode1.wavesnodes.com"})
	require.Nil(t, err)
	body, resp, err :=
		client.Blocks.HeadersAt(context.Background(), 379627)

	if err != nil {
		t.Fatalf("expected nil, found %+v", err)
	}

	headers := &Headers{
		Version:   3,
		Timestamp: 1539774945278,
		Reference: "EVd5YGijfQc9h8UFgzgJGpTqqAsekuE7GSEg6AobFVoPQ9eMkRcWSjLuHCCmemfZQCm64T7vHVtT7n5ud1KGvK8",
		NxtConsensus: NxtConsensus{
			BaseTarget:          727,
			GenerationSignature: "HCE5bG5AdSj7C5czZB8DGMou9AS1bC9853MWBKUMLipg",
		},
		Features:         []uint64{},
		Generator:        "3My3KZgFQ3CrVHgz6vGRt8687sH4oAA1qp8",
		Signature:        "2TXfMcQNHJVmkbNoznZrFRLaQHiBayFV9mzxt4VJkyXmxe9aGNn5A2unzUX4M2tqiHEfaWdfCBBo8zJQQpFrCKUY",
		Blocksize:        387,
		TransactionCount: 1,
		Height:           379627,
	}

	assert.NotNil(t, resp)
	assert.Equal(t, headers, body)
}

func TestBlocks_HeadersLast(t *testing.T) {
	client, err := NewClient(Options{BaseUrl: "https://testnode1.wavesnodes.com"})
	require.Nil(t, err)
	body, resp, err :=
		client.Blocks.HeadersLast(context.Background())

	if err != nil {
		t.Fatalf("expected nil, found %+v", err)
	}
	assert.NotNil(t, resp)
	assert.IsType(t, &Headers{}, body)
}

func TestBlocks_HeadersSeq(t *testing.T) {
	client, err := NewClient(Options{BaseUrl: "https://testnode1.wavesnodes.com"})
	require.Nil(t, err)
	body, resp, err :=
		client.Blocks.HeadersSeq(context.Background(), 375500, 375500)

	if err != nil {
		t.Fatalf("expected nil, found %+v", err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
}
