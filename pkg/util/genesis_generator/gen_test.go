package genesis_generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestGenerate(t *testing.T) {
	a, err := proto.MustKeyPair([]byte("test")).Addr(proto.MainNetScheme)
	require.NoError(t, err)
	block, err := GenerateGenesisBlock(proto.MainNetScheme, []GenesisTransactionInfo{{Address: a, Amount: 9000000000000000, Timestamp: 1558516864282}}, 153722867, 1558516864282)
	require.NoError(t, err)
	require.Equal(t, 1, block.TransactionCount)
	ok, err := block.VerifySignature(proto.MainNetScheme)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestGenerateMainNet(t *testing.T) {
	txs := []GenesisTransactionInfo{
		{Address: proto.MustAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ"), Amount: 9999999500000000, Timestamp: 1465742577614},
		{Address: proto.MustAddressFromString("3P8JdJGYc7vaLu4UXUZc1iRLdzrkGtdCyJM"), Amount: 100000000, Timestamp: 1465742577614},
		{Address: proto.MustAddressFromString("3PAGPDPqnGkyhcihyjMHe9v36Y4hkAh9yDy"), Amount: 100000000, Timestamp: 1465742577614},
		{Address: proto.MustAddressFromString("3P9o3ZYwtHkaU1KxsKkFjJqJKS3dLHLC9oF"), Amount: 100000000, Timestamp: 1465742577614},
		{Address: proto.MustAddressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3"), Amount: 100000000, Timestamp: 1465742577614},
		{Address: proto.MustAddressFromString("3PBWXDFUc86N2EQxKJmW8eFco65xTyMZx6J"), Amount: 100000000, Timestamp: 1465742577614},
	}
	sig := crypto.MustSignatureFromBase58("FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2")
	block, err := RecreateGenesisBlock(proto.MainNetScheme, txs, 153722867, 1460678400000, sig)
	require.NoError(t, err)
	bb, err := block.MarshalBinary(proto.MainNetScheme)
	require.NoError(t, err)
	assert.Equal(t, 500, len(bb))
	assert.Equal(t, 6, block.TransactionCount)
	assert.Equal(t, 283, int(block.TransactionBlockLength))
	assert.Equal(t, sig.Bytes(), block.BlockSignature.Bytes())
	ok, err := block.VerifySignature(proto.MainNetScheme)
	require.NoError(t, err)
	assert.True(t, ok)

	txID1, err := block.Transactions[0].GetID(proto.MainNetScheme)
	require.NoError(t, err)
	assert.Equal(t,
		crypto.MustSignatureFromBase58("2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8").Bytes(), txID1)

	txID2, err := block.Transactions[1].GetID(proto.MainNetScheme)
	require.NoError(t, err)
	assert.Equal(t,
		crypto.MustSignatureFromBase58("2TsxPS216SsZJAiep7HrjZ3stHERVkeZWjMPFcvMotrdGpFa6UCCmoFiBGNizx83Ks8DnP3qdwtJ8WFcN9J4exa3").Bytes(), txID2)

	txID3, err := block.Transactions[2].GetID(proto.MainNetScheme)
	require.NoError(t, err)
	assert.Equal(t,
		crypto.MustSignatureFromBase58("3gF8LFjhnZdgEVjP7P6o1rvwapqdgxn7GCykCo8boEQRwxCufhrgqXwdYKEg29jyPWthLF5cFyYcKbAeFvhtRNTc").Bytes(), txID3)

	txID4, err := block.Transactions[3].GetID(proto.MainNetScheme)
	require.NoError(t, err)
	assert.Equal(t,
		crypto.MustSignatureFromBase58("5hjSPLDyqic7otvtTJgVv73H3o6GxgTBqFMTY2PqAFzw2GHAnoQddC4EgWWFrAiYrtPadMBUkoepnwFHV1yR6u6g").Bytes(), txID4)

	txID5, err := block.Transactions[4].GetID(proto.MainNetScheme)
	require.NoError(t, err)
	assert.Equal(t,
		crypto.MustSignatureFromBase58("ivP1MzTd28yuhJPkJsiurn2rH2hovXqxr7ybHZWoRGUYKazkfaL9MYoTUym4sFgwW7WB5V252QfeFTsM6Uiz3DM").Bytes(), txID5)

	txID6, err := block.Transactions[5].GetID(proto.MainNetScheme)
	require.NoError(t, err)
	assert.Equal(t,
		crypto.MustSignatureFromBase58("29gnRjk8urzqc9kvqaxAfr6niQTuTZnq7LXDAbd77nydHkvrTA4oepoMLsiPkJ8wj2SeFB5KXASSPmbScvBbfLiV").Bytes(), txID6)
}

func TestGenerateDevNet(t *testing.T) {
	txs := []GenesisTransactionInfo{
		{Address: proto.MustAddressFromString("3FgScYB6MNdnN8m4xXddQe1Bjkwmd3U7YtM"), Amount: 6130000000000000, Timestamp: 1597073607702},
		{Address: proto.MustAddressFromString("3FWXhvWq2r8m54MmCEZ3YZkLg2qUdGWbU3V"), Amount: 15000000000000, Timestamp: 1597073607702},
		{Address: proto.MustAddressFromString("3FcSgww3tKZ7feQVmcnPFmRxsjqBodYz63x"), Amount: 25000000000000, Timestamp: 1597073607702},
		{Address: proto.MustAddressFromString("3FS5TnwA7xEXQ8LFRBdNk1MwqFR5SGz8vPn"), Amount: 25000000000000, Timestamp: 1597073607702},
		{Address: proto.MustAddressFromString("3FPzy3a12ccLUXTVTz5vhvkmVYXTXdVTKqF"), Amount: 40000000000000, Timestamp: 1597073607702},
		{Address: proto.MustAddressFromString("3FdEAz6F8xj37XVSUVTiqu8YfKBvtzWZZtn"), Amount: 45000000000000, Timestamp: 1597073607702},
		{Address: proto.MustAddressFromString("3FWMHWBXf5qzDenTFhUhT2tuqaoGnYHr6PM"), Amount: 50000000000000, Timestamp: 1597073607702},
		{Address: proto.MustAddressFromString("3FQntwq5KiXxEb8k2xLM6VGcZbBoTEroCsB"), Amount: 70000000000000, Timestamp: 1597073607702},
	}
	sig := crypto.MustSignatureFromBase58("5rDxRRzc9CM21j8XuAE1qp39svEr1BeLLF38HnchZd579ATdAPHqWxkt42AtoAV52GkVLU6F3TC2CWp2nzRKHpj8")
	block, err := RecreateGenesisBlock('D', txs, 5000, 1597073607702, sig)
	require.NoError(t, err)
	assert.Equal(t, 8, block.TransactionCount)
	ok, err := block.VerifySignature('D')
	require.NoError(t, err)
	assert.True(t, ok)
}
