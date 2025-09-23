package bls_test

import (
	"crypto/rand"
	"io"
	"testing"

	cbls "github.com/cloudflare/circl/sign/bls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
)

func randWavesSK(t *testing.T) crypto.SecretKey {
	var sk crypto.SecretKey
	_, err := io.ReadFull(rand.Reader, sk[:])
	require.NoError(t, err)
	return sk
}

func TestSignAndVerifySingle(t *testing.T) {
	sk, err := bls.NewSecretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)

	msg := []byte("single-sign test")
	sig, err := bls.Sign(sk, msg)
	assert.NoError(t, err)
	require.Len(t, sig, bls.SignatureSize, "compressed G2 signature must be 96 bytes")

	pk, err := sk.PublicKey()
	require.NoError(t, err)

	ok, err := bls.Verify(pk, msg, sig)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify with CIRCL directly
	csk, err := sk.ToCIRCLSecretKey()
	require.NoError(t, err)

	cpk := csk.PublicKey()
	ok = cbls.Verify[cbls.G1](cpk, msg, sig.Bytes())
	require.True(t, ok, "single signature should verify")

	// Negative: wrong message
	ok, err = bls.Verify(pk, []byte("other"), sig)
	assert.NoError(t, err)
	assert.False(t, ok, "signature must fail on different message")
}

func TestAggregateFromWavesSecrets_SameMessage(t *testing.T) {
	const n = 4
	msg := []byte("aggregate same msg test")

	// Make n secrete keys.
	sks := make([]bls.SecretKey, n)
	pks := make([]bls.PublicKey, n)
	for i := range sks {
		sk, err := bls.NewSecretKeyFromWavesSecretKey(randWavesSK(t))
		require.NoError(t, err)
		sks[i] = sk
		pk, err := sk.PublicKey()
		require.NoError(t, err)
		pks[i] = pk
	}
	// Make n signatures.
	sigs := make([]bls.Signature, n)
	for i, sk := range sks {
		sig, err := bls.Sign(sk, msg)
		require.NoError(t, err)
		sigs[i] = sig
	}
	// Aggregate signatures.
	aggSig, err := bls.AggregateSignatures(sigs)
	require.NoError(t, err)
	require.Len(t, aggSig, bls.SignatureSize)

	ok := bls.VerifyAggregate(pks, msg, aggSig)
	require.True(t, ok, "aggregate verify should pass")

	ok = bls.VerifyAggregate(pks, []byte("wrong"), aggSig)
	require.False(t, ok, "aggregate must fail on different message")
}

func TestVerifyAggregate_RejectsDuplicatePublicKeys(t *testing.T) {
	sk1, err := bls.NewSecretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)
	sk2, err := bls.NewSecretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)

	pk1, err := sk1.PublicKey()
	require.NoError(t, err)
	pk2, err := sk2.PublicKey()
	require.NoError(t, err)

	msg := []byte("same message")

	sig1, err := bls.Sign(sk1, msg)
	require.NoError(t, err)
	sig2, err := bls.Sign(sk2, msg)
	require.NoError(t, err)

	aggSig, err := bls.AggregateSignatures([]bls.Signature{sig1, sig2})
	require.NoError(t, err)

	pubs := []bls.PublicKey{pk1, pk2, pk1}
	ok := bls.VerifyAggregate(pubs, msg, aggSig)
	require.False(t, ok, "VerifyAggregate must fail on duplicate public keys")
}

func TestAggregateSignatures_RejectsDuplicateSignatures(t *testing.T) {
	sk1, err := bls.NewSecretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)
	sk2, err := bls.NewSecretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)

	msg := []byte("same message")

	sig1, err := bls.Sign(sk1, msg)
	require.NoError(t, err)
	sig2, err := bls.Sign(sk2, msg)
	require.NoError(t, err)

	_, err = bls.AggregateSignatures([]bls.Signature{sig1, sig2, sig1})
	require.ErrorIs(t, err, bls.ErrDuplicateSignature)
}
