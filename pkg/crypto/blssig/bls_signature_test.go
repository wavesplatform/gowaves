package blssig_test

import (
	"crypto/rand"
	"io"
	"testing"

	cbls "github.com/cloudflare/circl/sign/bls"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/blssig"
)

func randWavesSK(t *testing.T) crypto.SecretKey {
	var sk crypto.SecretKey
	_, err := io.ReadFull(rand.Reader, sk[:])
	require.NoError(t, err)
	return sk
}

func TestSignAndVerifySingle(t *testing.T) {
	wavesSK := randWavesSK(t)
	sk, err := blssig.SecretKeyFromWaves(wavesSK)
	require.NoError(t, err)

	msg := []byte("single-sign test")
	sig := blssig.Sign(sk, msg)
	require.Len(t, sig, 96, "compressed G2 signature must be 96 bytes")

	// Verify with CIRCL directly
	pk := sk.PublicKey()
	ok := cbls.Verify[cbls.G1](pk, msg, sig)
	require.True(t, ok, "single signature should verify")

	// Negative: wrong message
	ok = cbls.Verify[cbls.G1](pk, []byte("other"), sig)
	require.False(t, ok, "signature must fail on different message")
}

func TestAggregateFromWavesSecrets_SameMessage(t *testing.T) {
	const n = 4
	msg := []byte("aggregate same msg test")

	// Make n random Waves secrets
	waves := make([]crypto.SecretKey, n)
	for i := range waves {
		waves[i] = randWavesSK(t)
	}

	aggSig, pubs, err := blssig.AggregateFromWavesSecrets(waves, msg)
	require.NoError(t, err)
	require.Len(t, pubs, n)
	require.Len(t, aggSig, 96)

	ok := blssig.VerifyAggregate(pubs, msg, aggSig)
	require.True(t, ok, "aggregate verify should pass")

	ok = blssig.VerifyAggregate(pubs, []byte("wrong"), aggSig)
	require.False(t, ok, "aggregate must fail on different message")
}

func TestAggregateSignatures_DirectAndHelper(t *testing.T) {
	// Build two keys
	w1, w2 := randWavesSK(t), randWavesSK(t)
	k1, err := blssig.SecretKeyFromWaves(w1)
	require.NoError(t, err)
	k2, err := blssig.SecretKeyFromWaves(w2)
	require.NoError(t, err)

	msg := []byte("same msg")

	sig1 := blssig.Sign(k1, msg)
	sig2 := blssig.Sign(k2, msg)

	// Aggregate using the helper.
	agg1, pubs, err := blssig.AggregateFromWavesSecrets([]crypto.SecretKey{w1, w2}, msg)
	require.NoError(t, err)
	require.Len(t, pubs, 2)

	// Aggregate directly.
	agg2, err := blssig.AggregateSignatures([]cbls.Signature{sig1, sig2})
	require.NoError(t, err)

	require.Equal(t, agg1, agg2, "aggregates via helper vs direct must match")
	require.True(t, blssig.VerifyAggregate(pubs, msg, agg1))
	require.True(t, blssig.VerifyAggregate(pubs, msg, agg2))
}

func TestVerifyAggregate_RejectsDuplicatePublicKeys(t *testing.T) {
	w1, w2 := randWavesSK(t), randWavesSK(t)
	msg := []byte("same message")

	aggSig, pubs, err := blssig.AggregateFromWavesSecrets([]crypto.SecretKey{w1, w2}, msg)
	require.NoError(t, err)
	require.Len(t, pubs, 2)

	pubsDup := []*cbls.PublicKey[cbls.G1]{pubs[0], pubs[0]}
	ok := blssig.VerifyAggregate(pubsDup, msg, aggSig)
	require.False(t, ok, "VerifyAggregate must fail on duplicate public keys")
}

func TestAggregateSignatures_RejectsDuplicateSignatures(t *testing.T) {
	w := randWavesSK(t)
	sk, err := blssig.SecretKeyFromWaves(w)
	require.NoError(t, err)

	msg := []byte("m")
	s := blssig.Sign(sk, msg)

	_, err = blssig.AggregateSignatures([]cbls.Signature{s, s})
	require.ErrorIs(t, err, blssig.ErrDuplicateSignature)
}
