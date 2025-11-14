package proto

import (
	"crypto/rand"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func randomDigest() crypto.Digest {
	r := crypto.Digest{}
	_, _ = rand.Read(r[:])
	return r
}

func randomSignature() crypto.Signature {
	r := crypto.Signature{}
	_, _ = rand.Read(r[:])
	return r
}

func randomBlockID() BlockID {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	if b[0] > math.MaxInt8 {
		return NewBlockIDFromSignature(randomSignature())
	}
	return NewBlockIDFromDigest(randomDigest())
}

func randomStateHashV1() StateHashV1 {
	return StateHashV1{
		BlockID: randomBlockID(),
		SumHash: randomDigest(),
		FieldsHashesV1: FieldsHashesV1{
			WavesBalanceHash:  randomDigest(),
			AssetBalanceHash:  randomDigest(),
			DataEntryHash:     randomDigest(),
			AccountScriptHash: randomDigest(),
			AssetScriptHash:   randomDigest(),
			LeaseBalanceHash:  randomDigest(),
			LeaseStatusHash:   randomDigest(),
			SponsorshipHash:   randomDigest(),
			AliasesHash:       randomDigest(),
		},
	}
}

func createStateHashV1() StateHashV1 {
	return StateHashV1{
		BlockID: NewBlockIDFromSignature(crypto.MustSignatureFromBase58(
			"2UwZrKyjx7Bs4RYkEk5SLCdtr9w6GR1EDbpS3TH9DGJKcxSCuQP4nivk4YPFpQTqWmoXXPPUiy6riF3JwhikbSQu",
		)),
		SumHash: crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
		FieldsHashesV1: FieldsHashesV1{
			WavesBalanceHash:  crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			AssetBalanceHash:  crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			DataEntryHash:     crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			AccountScriptHash: crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			AssetScriptHash:   crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			LeaseBalanceHash:  crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			LeaseStatusHash:   crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			SponsorshipHash:   crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			AliasesHash:       crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
		},
	}
}

func TestStateHashJSONRoundTrip(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			sh := randomStateHashV1()
			js, err := sh.MarshalJSON()
			assert.NoError(t, err)
			var sh2 StateHashV1
			err = sh2.UnmarshalJSON(js)
			assert.NoError(t, err)
			assert.Equal(t, sh, sh2)
		})
	}
}

func TestStateHashBinaryRoundTrip(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			sh := randomStateHashV1()
			data := sh.MarshalBinary()
			var sh2 StateHashV1
			err := sh2.UnmarshalBinary(data)
			assert.NoError(t, err)
			assert.Equal(t, sh, sh2)
		})
	}
}

func TestStateHash_GenerateSumHash(t *testing.T) {
	sh := createStateHashV1()
	prevHash := crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	correctSumHash := crypto.MustDigestFromBase58("9ckTqHUsRap8YerHv1EijZMeBRaSFibdTkPqjmK9hoNy")
	err := sh.GenerateSumHash(prevHash[:])
	assert.NoError(t, err)
	assert.Equal(t, correctSumHash, sh.SumHash)
}
