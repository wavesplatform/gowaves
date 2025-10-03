package bls_test

import (
	"crypto/rand"
	"fmt"
	"io"
	"slices"
	"testing"

	cbls "github.com/cloudflare/circl/sign/bls"
	"github.com/mr-tron/base58"
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
	sk, err := secretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)

	msg := []byte("single-sign test")
	sig, err := bls.Sign(sk, msg)
	assert.NoError(t, err)
	assert.Len(t, sig, bls.SignatureSize, "compressed G2 signature must be 96 bytes")

	pk, err := sk.PublicKey()
	require.NoError(t, err)

	ok, err := bls.Verify(pk, msg, sig)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify with public key produced from private key.
	lpk, err := sk.PublicKey()
	require.NoError(t, err)
	ok, err = bls.Verify(lpk, msg, sig)
	assert.NoError(t, err)
	assert.True(t, ok)

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
		sk, err := secretKeyFromWavesSecretKey(randWavesSK(t))
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
	sk1, err := secretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)
	sk2, err := secretKeyFromWavesSecretKey(randWavesSK(t))
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
	sk1, err := secretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)
	sk2, err := secretKeyFromWavesSecretKey(randWavesSK(t))
	require.NoError(t, err)

	msg := []byte("same message")

	sig1, err := bls.Sign(sk1, msg)
	require.NoError(t, err)
	sig2, err := bls.Sign(sk2, msg)
	require.NoError(t, err)

	_, err = bls.AggregateSignatures([]bls.Signature{sig1, sig2, sig1})
	require.ErrorIs(t, err, bls.ErrDuplicateSignature)
}

func TestScalaCompatibilityIndividualKeys(t *testing.T) {
	for i, test := range []struct {
		sk  string
		pk  string
		msg string
		sig string
	}{
		{
			sk:  "7QrCkjViFu6YdgCJ5CpSYJhvhpX2vmxgqBTv3Sbbwdu8",
			pk:  "66aU1fahh7JqwNtX2Fs1xVsFfocGEAo6J4Jf1SfyMBnBCKvVuVJrV4v5jDwtkTL4aB",
			msg: "2EwnXKysVvyFT",
			sig: "tjNe4uFJAJKPEEJjzBmzqsVnJXPSczcDSgS8ynZK27x2Go5uogJWPbUbfZJhhkjfeM" +
				"ow3BvdiHJCFBPGGF3kUGia7NFDZpkcxXNsWJQ1fCxfoTiPtMN2S9hGBwKVi8jFXq1",
		},
		{
			sk:  "8TG6n9rkMuqshDGUTwJt122ioe98ZTmjW2b77BtSYcPw",
			pk:  "6UiH1DdvMTvFzD3jVqyqz89s3MaMCfSCVF3ZY3wysdHxzzWq6j8fuk17HJCsoKUjVJ",
			msg: "2EwnXKysVvyFT",
			sig: "pKn7sCRLzRMEKaQBahPpf8JvUebEknGd2KbB2rWf11R6JthBxo8tJhU8PU9qfPYgTx" +
				"xYkRkeBENQyJSfxMfhkXkstmxk3KjwzVCqPYPqXm8sW85wFp7hVpMbH4jgauhoHMY",
		},
		{
			sk:  "8cMQE1tMwY81rzx2gRXJs1wZ7CEW28nmWqfoPyBWTV6Z",
			pk:  "6PakMfx4Lm5tfWuUzTbq1ASpXQ2KHBoWGWVZcLii7wfE3gRENdWpYNnQni7YsGNdBX",
			msg: "2EwnXKysVvyFT",
			sig: "22tm5aapLs1q9nyUxftSpiCM9CSbXadYm1W1jR3JByohUHi5PKTnJbUwBR9qzqJmw5" +
				"KGMz27ZCZz2LUHLxWKMj6Wf3QZmJeLKJtwhZQrLHYnxzskPu4h3QJ3JVsjF1TqUu5i",
		},
		{
			sk:  "8LpMSWhBCMhtFzcpw8BG2uvF33XFgDiZFy8tnwrm9VqC",
			pk:  "7dJyNxhN8K9AwcvSLyuM9TBQKAaNtLAH8dbhpkJ9hNWyCPMvCuPtRfFpRouC66xrcA",
			msg: "2EwnXKysVvyFT",
			sig: "osva5g4KVWWepsi9SpqkP3htqb1G27fnA8AdTUqwhPovUMCMu8H8VYCXzY1iFYNexb" +
				"ZW23DgDFExvQu4gQU9EdroR1WSC8aBhTAi9n2yi4BmuLcUNFTzjcopbyA2W6cfyGW",
		},
		{
			sk:  "5WN9f4FtY8vYDEzHzPHM37XJ6nPK7SXDiFcfCfH9GM3c",
			pk:  "6JK5t1ejidrSLfK8CBeY6TEjGk6RgR3FnFHbTBBtRLzRdHquSLem1Cn3F1AADTZRXq",
			msg: "2EwnXKysVvyFT",
			sig: "oSee1LsYjVX9LCicSj8R81ky1ytty3ysvKopP219MZW2hjBA3R36qhd5db7BscRUc1" +
				"2T8ZrzYWbQ6ShamTP1UJXP6w4DVgAM81ycyJQ3pc8ZcxQvyJ4HaHuD5zTwPz6wZuu",
		},
		{
			sk:  "2iZv8rNmHLozEvZruhxS1NsmP1vFKVh4v7pLVjE5rZZr",
			pk:  "7mr8XSaegfQTZwg7ZDKHBJGPXiKfFETRrveCNPuCK1AGrLDn5JAFfqTQEe1aGcpcM6",
			msg: "2EwnXKysVvyFT",
			sig: "pN99iSsodRotMuR2hJXTL23AFLXM9Yav4v5WpBT6a1JNBGch675pEFRsfievbpCQoV" +
				"eCZCFLqcdxsWeDohqcqJBimxRwT9FQ46SWSqQssQBYdSbYn9FPHP4snMYmryXM87z",
		},
		{
			sk:  "c2DfuL5FgrX18yocP9P6t64gVDVKQCPanKGWWbChb29",
			pk:  "7pTYbUjbZypyoq9KKTCLDqp84XPMJHfswMkvK6eej93k2nStDXmz16PVCzgmoQJSxF",
			msg: "2EwnXKysVvyFT",
			sig: "yrD7AcHtF94T5CCwDaH5NVYo1foAQTmV4NXvmNmXGKcfEbtJV1PQVezxZ51fbAZuBX" +
				"iU9u9HDMtirPJZt2mB6xRsgcy4cQ7P7FKFEx8JMgUu9bYP7UUYXdxsrHbcxmEY3pT",
		},
		{
			sk:  "7kk2WThvUJGrV6Y8KXCdMewDvW4N4UcHWXNAzLZvwuWt",
			pk:  "7o2qfELmjNEDcVCkGxE3CRtQXXwYfvaqZmsEUF3tqz6x6dnyT7DDYLbHJjaFFMhgRs",
			msg: "2EwnXKysVvyFT",
			sig: "zLAJCKM2MCACAXTx5qcSTu3kRUCdAhyookbRLNVkFf2G2Mpn7LAEdnMAjVSTankEhQ" +
				"35EtSRVS4JhLWiCzuRU1rPfAjkHwBJsyZLsnXobACRi6wMnM4NUXDXGP9Jnq5Bmcs",
		},
		{
			sk:  "6o6ay7SKnjwKq1n2Y87kVKQLjZC4TAg7yM8RLNJei7Kj",
			pk:  "6wfVJ2PwU6Fk3t6iFZL37WcAoCgU9XCBvRnuPR4Sd7bo14f2BSeNwnGgvCNnDmgxTU",
			msg: "2EwnXKysVvyFT",
			sig: "24v3nCPjMvAJgZKHfxv7FcXtZPpzHaSfaCa3L7TQ93Cu1yTfdYXPm2WVEy6qCbBBcr" +
				"vRBQ1v3tUzK8bRURJyYSbHCegbj19R9RV2Wf5gRnaumvhae7PXURsyUWrPJx3vzjxk",
		},
		{
			sk:  "5SKq1X6yAb8jReZBpdqjeLczf3ZRRAskqXhuR9NbX5V7",
			pk:  "6aj65eLuBTpTVczhTCGHFdMgnVDbfx96ThBVVqhVNUPirddmtMeNQiC6h8oqvhpvFa",
			msg: "2EwnXKysVvyFT",
			sig: "qESWK96Bpmoi7BvBKSbd695ymGV1hBPRTHJDysZ9ocvF81cwJvvqcCFAn7SKELCDda" +
				"2H7655bUTAPy6fvgnEsYnCPDkJ7cEpMqLMtoXL7ssMsx1fPH2cWHmuxopsAvYvhkm",
		},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			sk, err := bls.NewSecretKeyFromBase58(test.sk)
			require.NoError(t, err)
			pk, err := bls.NewPublicKeyFromBase58(test.pk)
			require.NoError(t, err)

			// Check that local generated public keys match the test PKs.
			apk, err := sk.PublicKey()
			require.NoError(t, err)
			assert.Equal(t, pk, apk)

			msg, err := base58.Decode(test.msg)
			require.NoError(t, err)

			sig, err := bls.NewSignatureFromBase58(test.sig)
			require.NoError(t, err)

			// Verify individual signatures.
			ok, err := bls.Verify(pk, msg, sig)
			require.NoError(t, err)
			require.True(t, ok, "sig1 must verify")
		})
	}
}

func TestScalaCompatibilityAggregatedSignatures(t *testing.T) {
	for i, test := range []struct {
		pks    []string
		msg    string
		sigs   []string
		aggSig string
	}{
		{
			pks: []string{
				"66aU1fahh7JqwNtX2Fs1xVsFfocGEAo6J4Jf1SfyMBnBCKvVuVJrV4v5jDwtkTL4aB",
				"6UiH1DdvMTvFzD3jVqyqz89s3MaMCfSCVF3ZY3wysdHxzzWq6j8fuk17HJCsoKUjVJ",
			},
			msg: "2EwnXKysVvyFT",
			sigs: []string{
				"tjNe4uFJAJKPEEJjzBmzqsVnJXPSczcDSgS8ynZK27x2Go5uogJWPbUbfZJhhkjfeMo" +
					"w3BvdiHJCFBPGGF3kUGia7NFDZpkcxXNsWJQ1fCxfoTiPtMN2S9hGBwKVi8jFXq1",
				"pKn7sCRLzRMEKaQBahPpf8JvUebEknGd2KbB2rWf11R6JthBxo8tJhU8PU9qfPYgTxx" +
					"YkRkeBENQyJSfxMfhkXkstmxk3KjwzVCqPYPqXm8sW85wFp7hVpMbH4jgauhoHMY",
			},
			aggSig: "r3oSeCJ75HbzgZhRJmw9mQMiGdbA2zhK7nF1wy7Ajju2jyUXS69eqB477RJN8wm" +
				"2vzZgTpciwKV6dbWzmEKZkDARkoR6mbJTjzaWPDweKweBVfDBoyNsYvwrzXg8xrgeLDu",
		},
		{
			pks: []string{
				"66aU1fahh7JqwNtX2Fs1xVsFfocGEAo6J4Jf1SfyMBnBCKvVuVJrV4v5jDwtkTL4aB",
				"6UiH1DdvMTvFzD3jVqyqz89s3MaMCfSCVF3ZY3wysdHxzzWq6j8fuk17HJCsoKUjVJ",
			},
			msg: "2EwnXKysVvyFT",
			sigs: []string{
				"tjNe4uFJAJKPEEJjzBmzqsVnJXPSczcDSgS8ynZK27x2Go5uogJWPbUbfZJhhkjfeMo" +
					"w3BvdiHJCFBPGGF3kUGia7NFDZpkcxXNsWJQ1fCxfoTiPtMN2S9hGBwKVi8jFXq1",
				"pKn7sCRLzRMEKaQBahPpf8JvUebEknGd2KbB2rWf11R6JthBxo8tJhU8PU9qfPYgTxx" +
					"YkRkeBENQyJSfxMfhkXkstmxk3KjwzVCqPYPqXm8sW85wFp7hVpMbH4jgauhoHMY",
			},
			aggSig: "r3oSeCJ75HbzgZhRJmw9mQMiGdbA2zhK7nF1wy7Ajju2jyUXS69eqB477RJN8wm2" +
				"vzZgTpciwKV6dbWzmEKZkDARkoR6mbJTjzaWPDweKweBVfDBoyNsYvwrzXg8xrgeLDu",
		},
		{
			pks: []string{
				"6PakMfx4Lm5tfWuUzTbq1ASpXQ2KHBoWGWVZcLii7wfE3gRENdWpYNnQni7YsGNdBX",
				"7dJyNxhN8K9AwcvSLyuM9TBQKAaNtLAH8dbhpkJ9hNWyCPMvCuPtRfFpRouC66xrcA",
			},
			msg: "2EwnXKysVvyFT",
			sigs: []string{
				"22tm5aapLs1q9nyUxftSpiCM9CSbXadYm1W1jR3JByohUHi5PKTnJbUwBR9qzqJmw5K" +
					"GMz27ZCZz2LUHLxWKMj6Wf3QZmJeLKJtwhZQrLHYnxzskPu4h3QJ3JVsjF1TqUu5i",
				"osva5g4KVWWepsi9SpqkP3htqb1G27fnA8AdTUqwhPovUMCMu8H8VYCXzY1iFYNexbZ" +
					"W23DgDFExvQu4gQU9EdroR1WSC8aBhTAi9n2yi4BmuLcUNFTzjcopbyA2W6cfyGW",
			},
			aggSig: "qaEQnqFuiPNCufM7VfEUwJiHK4ox6iLDt7XEYMHgkkRfnvt7rHM6GS1kUxsfV4Yu" +
				"9WPFmn59ffF3WdsuDjH29weRsaicjwdqRG4Cg5fw5s7qgjXVXjQ9n9Yuc99aM1D56qb",
		},
		{
			pks: []string{
				"6JK5t1ejidrSLfK8CBeY6TEjGk6RgR3FnFHbTBBtRLzRdHquSLem1Cn3F1AADTZRXq",
				"7mr8XSaegfQTZwg7ZDKHBJGPXiKfFETRrveCNPuCK1AGrLDn5JAFfqTQEe1aGcpcM6",
				"7pTYbUjbZypyoq9KKTCLDqp84XPMJHfswMkvK6eej93k2nStDXmz16PVCzgmoQJSxF",
			},
			msg: "2EwnXKysVvyFT",
			sigs: []string{
				"oSee1LsYjVX9LCicSj8R81ky1ytty3ysvKopP219MZW2hjBA3R36qhd5db7BscRUc12" +
					"T8ZrzYWbQ6ShamTP1UJXP6w4DVgAM81ycyJQ3pc8ZcxQvyJ4HaHuD5zTwPz6wZuu",
				"pN99iSsodRotMuR2hJXTL23AFLXM9Yav4v5WpBT6a1JNBGch675pEFRsfievbpCQoVe" +
					"CZCFLqcdxsWeDohqcqJBimxRwT9FQ46SWSqQssQBYdSbYn9FPHP4snMYmryXM87z",
				"yrD7AcHtF94T5CCwDaH5NVYo1foAQTmV4NXvmNmXGKcfEbtJV1PQVezxZ51fbAZuBXiU" +
					"9u9HDMtirPJZt2mB6xRsgcy4cQ7P7FKFEx8JMgUu9bYP7UUYXdxsrHbcxmEY3pT",
			},
			aggSig: "swRUtaCA6Q77Gi1E29qQPXTPmhYLUSQwMtxcesDSh88NvCwWATVUUb17VmcVkZe6x" +
				"Myp5yu3ps5VoBbxh4QZiBPS97EuyXyGtpHLNQdHZQ3NR835QWWgUDa8qYP7CVKDk6P",
		},
		{
			pks: []string{
				"7o2qfELmjNEDcVCkGxE3CRtQXXwYfvaqZmsEUF3tqz6x6dnyT7DDYLbHJjaFFMhgRs",
				"6wfVJ2PwU6Fk3t6iFZL37WcAoCgU9XCBvRnuPR4Sd7bo14f2BSeNwnGgvCNnDmgxTU",
				"6aj65eLuBTpTVczhTCGHFdMgnVDbfx96ThBVVqhVNUPirddmtMeNQiC6h8oqvhpvFa",
			},
			msg: "2EwnXKysVvyFT",
			sigs: []string{
				"zLAJCKM2MCACAXTx5qcSTu3kRUCdAhyookbRLNVkFf2G2Mpn7LAEdnMAjVSTankEhQ3" +
					"5EtSRVS4JhLWiCzuRU1rPfAjkHwBJsyZLsnXobACRi6wMnM4NUXDXGP9Jnq5Bmcs",
				"24v3nCPjMvAJgZKHfxv7FcXtZPpzHaSfaCa3L7TQ93Cu1yTfdYXPm2WVEy6qCbBBcrv" +
					"RBQ1v3tUzK8bRURJyYSbHCegbj19R9RV2Wf5gRnaumvhae7PXURsyUWrPJx3vzjxk",
				"qESWK96Bpmoi7BvBKSbd695ymGV1hBPRTHJDysZ9ocvF81cwJvvqcCFAn7SKELCDda2" +
					"H7655bUTAPy6fvgnEsYnCPDkJ7cEpMqLMtoXL7ssMsx1fPH2cWHmuxopsAvYvhkm",
			},
			aggSig: "u8UZtNBGoDoicpMsoA4ZqUTDWRTPqnx36UqZ7AZk3VE9SCLk2VbJmAoyWTmp9Ch" +
				"bA4YhPrKkfoqhkgCPKfcCybjU4TXTH2eufaL5g994Cr8bWEe8EuXFx2hQ4pM13FsRhEw",
		},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			pks := make([]bls.PublicKey, 0, len(test.pks))
			for _, s := range test.pks {
				pk, err := bls.NewPublicKeyFromBase58(s)
				require.NoError(t, err)
				pks = append(pks, pk)
			}
			sigs := make([]bls.Signature, 0, len(test.sigs))
			for _, s := range test.sigs {
				sig, err := bls.NewSignatureFromBase58(s)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
			msg, err := base58.Decode(test.msg)
			require.NoError(t, err)
			aggSig, err := base58.Decode(test.aggSig)
			require.NoError(t, err)

			// Check that local aggregated signature matches the test aggSig.
			ags, err := bls.AggregateSignatures(sigs)
			require.NoError(t, err)
			assert.Equal(t, aggSig, ags)

			// Verify aggregate signature.
			ok := bls.VerifyAggregate(pks, msg, aggSig)
			require.True(t, ok, "aggregate must verify")

			// Verify that order of public keys does not matter.
			slices.Reverse(pks)
			ok = bls.VerifyAggregate(pks, msg, aggSig)
			require.True(t, ok, "aggregate must verify")
		})
	}
}

// secretKeyFromWavesSecretKey generates BLS secret key from Waves secret key.
func secretKeyFromWavesSecretKey(wavesSK crypto.SecretKey) (bls.SecretKey, error) {
	k, err := cbls.KeyGen[cbls.G1](wavesSK.Bytes(), nil, nil)
	if err != nil {
		return bls.SecretKey{}, fmt.Errorf("failed to create BLS secret key from Waves secret key: %w", err)
	}
	b, err := k.MarshalBinary()
	if err != nil {
		return bls.SecretKey{}, fmt.Errorf("failed to create BLS secret key from Waves secret key: %w", err)
	}
	sk, err := bls.NewSecretKeyFromBytes(b)
	if err != nil {
		return bls.SecretKey{}, fmt.Errorf("failed to create BLS secret key from Waves secret key: %w", err)
	}
	return sk, nil
}
