package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

var blocksHeightBySignatureJson = `
{
  "height": 372306
}
`

func TestBlocks_HeightBySignature(t *testing.T) {
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(blocksHeightBySignatureJson, 200),
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Blocks.HeightBySignature(
			context.Background(),
			"2TXfMcQNHJVmkbNoznZrFRLaQHiBayFV9mzxt4VJkyXmxe9aGNn5A2unzUX4M2tqiHEfaWdfCBBo8zJQQpFrCKUY")
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, 372306, body.Height)
	assert.Equal(t, "https://testnode1.wavesnodes.com/blocks/height/2TXfMcQNHJVmkbNoznZrFRLaQHiBayFV9mzxt4VJkyXmxe9aGNn5A2unzUX4M2tqiHEfaWdfCBBo8zJQQpFrCKUY", resp.Request.URL.String())
}

var blocksHeadersAtJson = `
{
  "version": 2,
  "timestamp": 1485532386874,
  "reference": "2DqubQMMBt4ot7y8F37JKWLV9J1Fvn35b4TBLGc3A9gzRvL4DweknWghxYJLYf8edDtDZujCbu1Cwqr19kC8jy12",
  "nxt-consensus": {
    "base-target": 279,
    "generation-signature": "GdXMcQzP99TJMsKX37v6BqVDcbC1xd26fgk5LRjhQUhR"
  },
  "generator": "3N5GRqzDBhjVXnCn44baHcz2GoZy5qLxtTh",
  "signature": "4rnYtWNE8WV4heso4q86Uwbcf1XZR5ShfszaKzyRg7aELP2Su3sFUhcCrQCyBA9SbE4T8pkd2AnLKnwBHwhUKaDq",
  "blocksize": 29882,
  "transactionCount": 100,
  "height": 370
}`

func TestBlocks_HeadersAt(t *testing.T) {
	reference, _ := crypto.NewSignatureFromBase58("2DqubQMMBt4ot7y8F37JKWLV9J1Fvn35b4TBLGc3A9gzRvL4DweknWghxYJLYf8edDtDZujCbu1Cwqr19kC8jy12")
	generator, _ := proto.NewAddressFromString("3N5GRqzDBhjVXnCn44baHcz2GoZy5qLxtTh")
	signature, _ := crypto.NewSignatureFromBase58("4rnYtWNE8WV4heso4q86Uwbcf1XZR5ShfszaKzyRg7aELP2Su3sFUhcCrQCyBA9SbE4T8pkd2AnLKnwBHwhUKaDq")

	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(blocksHeadersAtJson, 200),
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Blocks.HeadersAt(context.Background(), 370)
	require.NoError(t, err)
	headers := &Headers{
		Version:   2,
		Timestamp: 1485532386874,
		Reference: reference,
		NxtConsensus: NxtConsensus{
			BaseTarget:          279,
			GenerationSignature: "GdXMcQzP99TJMsKX37v6BqVDcbC1xd26fgk5LRjhQUhR",
		},
		Features:         nil,
		Generator:        generator,
		Signature:        signature,
		Blocksize:        29882,
		TransactionCount: 100,
		Height:           370,
	}

	assert.NotNil(t, resp)
	assert.Equal(t, headers, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/blocks/headers/at/370", resp.Request.URL.String())
}

var blocksHeadersLastJson = `
{
  "version": 3,
  "timestamp": 1542018438521,
  "reference": "5AP9TZaUXmK2M5dL2oLZVAF4wGdpB7fPoiQ6N7gPaZ6K6yc1uUpuqhJDNAGX5oYUjq8DUXh54h8vswBu5kye6Fb4",
  "nxt-consensus": {
    "base-target": 941,
    "generation-signature": "6fZ2coD4arq6jezTNHQXHLnUQqVvKwCTxQL8BzpqbYGZ"
  },
  "features": [],
  "generator": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "signature": "3gTwUyQ995T1zmjsSZM6s6zCrWQh9JcFotfCetAaEMh6QYGFXPVgQsmudDtyLFsgrBfaS7GjPzAmpY2CkbLxEG5j",
  "blocksize": 225,
  "transactionCount": 0,
  "height": 372323
}
`

func TestBlocks_HeadersLast(t *testing.T) {
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(blocksHeadersLastJson, 200),
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Blocks.HeadersLast(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.IsType(t, &Headers{}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/blocks/headers/last", resp.Request.URL.String())
}

var blocksHeadersSetJson = `
[
  {
    "version": 2,
    "timestamp": 1485532386874,
    "reference": "2DqubQMMBt4ot7y8F37JKWLV9J1Fvn35b4TBLGc3A9gzRvL4DweknWghxYJLYf8edDtDZujCbu1Cwqr19kC8jy12",
    "nxt-consensus": {
      "base-target": 279,
      "generation-signature": "GdXMcQzP99TJMsKX37v6BqVDcbC1xd26fgk5LRjhQUhR"
    },
    "generator": "3N5GRqzDBhjVXnCn44baHcz2GoZy5qLxtTh",
    "signature": "4rnYtWNE8WV4heso4q86Uwbcf1XZR5ShfszaKzyRg7aELP2Su3sFUhcCrQCyBA9SbE4T8pkd2AnLKnwBHwhUKaDq",
    "blocksize": 29882,
    "transactionCount": 100,
    "height": 370
  }
]`

func TestBlocks_HeadersSeq(t *testing.T) {
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(blocksHeadersSetJson, 200),
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Blocks.HeadersSeq(context.Background(), 375500, 375500)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.Equal(t, "https://testnode1.wavesnodes.com/blocks/headers/seq/375500/375500", resp.Request.URL.String())
}
