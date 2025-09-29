package bls_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
)

func TestPoPRoundTrip(t *testing.T) {
	for i, test := range []struct {
		height uint32
	}{
		{height: 0},
		{height: 1},
		{height: 123456},
		{height: 4294967295},
		{height: math.MaxInt32},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			sk, err := secretKeyFromWavesSecretKey(randWavesSK(t))
			require.NoError(t, err)
			pk, err := sk.PublicKey()
			require.NoError(t, err)
			msg, sig, err := bls.ProvePoP(sk, pk, test.height)
			assert.NoError(t, err)
			assert.Len(t, msg, bls.PoPMessageSize)
			ok, err := bls.VerifyPoP(pk, test.height, sig)
			assert.NoError(t, err)
			assert.True(t, ok)
			ok, err = bls.VerifyPoP(pk, 13, sig)
			assert.NoError(t, err)
			assert.False(t, ok)
		})
	}
}

func TestPoPVerifyScalaCompatibility(t *testing.T) {
	for i, test := range []struct {
		pk     string
		msg    string
		height uint32
		sig    string
	}{
		{
			pk:     "7QtCEETGT76GHP7gR3Qc9DQzNjJYbxn4UJ7Bz7RofMQx5RJY7mZNveuFNfgJYg2kLn",
			msg:    "ixUCXhhDbpRXVM3Cnaog2MNLRVt3R9oRgnNnrtCtxv35Lac2KQYMQkKNmHW9wt35dDA6vfU",
			height: 3,
			sig: "my5jyvoghjn94fQU1HQ5EN4WLdxhVzMZJJVY2F8nQ9kDJDr1wCoPrnLvY3xF6FiDJ2wWK8C" +
				"EeWd2NTKhFMB4chDSwRLRw2xPT45kMC726watbDx8cuF3omkwsZpRDKyX4x4",
		},
		{
			pk:     "7QtCEETGT76GHP7gR3Qc9DQzNjJYbxn4UJ7Bz7RofMQx5RJY7mZNveuFNfgJYg2kLn",
			msg:    "ixUCXhhDbpRXVM3Cnaog2MNLRVt3R9oRgnNnrtCtxv35Lac2KQYMQkKNmHW9wt35dDA6vfX",
			height: 6,
			sig: "ud4JBLaM8oqK5BmRAo1eXrTQPqD6fJ6yP1f6YovRV2P2ykKxgcgsv12wMvDNzLhd7KD96Nq" +
				"bUU88Ffqzsn47c1vBGrH6jR1NCbs9snf2FPiBTsX46eL95rgysCmZLiXsN29",
		},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			pk, err := bls.NewPublicKeyFromBase58(test.pk)
			require.NoError(t, err)
			sig, err := bls.NewSignatureFromBase58(test.sig)
			require.NoError(t, err)
			// Check message itself.
			ok, err := bls.VerifyPoP(pk, test.height, sig)
			assert.NoError(t, err)
			assert.True(t, ok)
			// Reconstruct message and check again.
			ok, err = bls.VerifyPoP(pk, test.height, sig)
			assert.NoError(t, err)
			assert.True(t, ok)
		})
	}
}
