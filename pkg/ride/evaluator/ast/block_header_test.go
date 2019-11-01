package ast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestNewMapFromBlockHeader(t *testing.T) {
	_, publicKey, _ := crypto.GenerateKeyPair([]byte("test"))
	parent := crypto.MustSignatureFromBase58("4sukfbjbbkBnFevQrGN7VvpBSwvufsuqvq5fmfiMdp1pBDMF5TanbFejRHhsiUQSWPkvWRdagwWD3oxnX3eEqzvM")
	addr := proto.MustAddressFromPublicKey(proto.MainNetScheme, publicKey)
	signa := crypto.MustSignatureFromBase58("5X76YVeG8T6iTxFmD5WNSaR13hxtsgJPQ2oELeZUsrQfZWSXtnUbq1kRqqMjfBngPvaEKVVV2FSujdTXm3hTW172")
	gensig := crypto.MustDigestFromBase58("6a1hWT8QNGw8wnacXQ8vT2YEFLuxRxVpEuaaSf6AbSvU")

	h := proto.BlockHeader{
		Version:       3,
		Timestamp:     1567506205718,
		Parent:        parent,
		FeaturesCount: 2,
		Features:      []int16{7, 99},
		NxtConsensus: proto.NxtConsensus{
			BaseTarget:   1310,
			GenSignature: gensig,
		},
		TransactionCount: 12,
		GenPublicKey:     publicKey,
		BlockSignature:   signa,
		Height:           659687,
	}

	rs, err := newMapFromBlockHeader(proto.MainNetScheme, &h)
	require.NoError(t, err)
	require.Equal(t, "BlockHeader", NewBlockHeader(rs).InstanceOf())
	require.Equal(t, NewLong(1567506205718), rs["timestamp"])
	require.Equal(t, NewLong(3), rs["version"])
	require.Equal(t, NewBytes(parent.Bytes()), rs["reference"])
	require.Equal(t, NewAddressFromProtoAddress(addr), rs["generator"])
	require.Equal(t, NewBytes(publicKey.Bytes()), rs["generatorPublicKey"])
	require.Equal(t, NewBytes(signa.Bytes()), rs["signature"])
	require.Equal(t, NewLong(1310), rs["baseTarget"])
	require.Equal(t, NewBytes(gensig.Bytes()), rs["generationSignature"])
	require.Equal(t, NewLong(12), rs["transactionCount"])
	require.Equal(t, Params(NewLong(7), NewLong(99)), rs["featureVotes"])
}
