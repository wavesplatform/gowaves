package proto

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func randomBool() bool {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return b[0]%2 == 0
}

func randomHeight() uint64 {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return binary.BigEndian.Uint64(b[:8])
}

func randomByte() byte {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return b[0]
}

func randomVersion() string {
	return fmt.Sprintf("v%d.%d.%d", randomByte(), randomByte(), randomByte())
}

func randomFieldsHashesV1() FieldsHashesV1 {
	return FieldsHashesV1{
		WavesBalanceHash:  randomDigest(),
		AssetBalanceHash:  randomDigest(),
		DataEntryHash:     randomDigest(),
		AccountScriptHash: randomDigest(),
		AssetScriptHash:   randomDigest(),
		LeaseBalanceHash:  randomDigest(),
		LeaseStatusHash:   randomDigest(),
		SponsorshipHash:   randomDigest(),
		AliasesHash:       randomDigest(),
	}
}

func randomStateHashV1() StateHashV1 {
	return StateHashV1{
		BlockID:        randomBlockID(),
		SumHash:        randomDigest(),
		FieldsHashesV1: randomFieldsHashesV1(),
	}
}

func randomStateHashV2() StateHashV2 {
	return StateHashV2{
		BlockID: randomBlockID(),
		SumHash: randomDigest(),
		FieldsHashesV2: FieldsHashesV2{
			FieldsHashesV1: randomFieldsHashesV1(),
			GeneratorsHash: randomDigest(),
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

func TestStateHashV1JSONRoundTrip(t *testing.T) {
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

func TestStateHashV2JSONRoundTrip(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			sh := randomStateHashV2()
			js, err := sh.MarshalJSON()
			assert.NoError(t, err)
			var sh2 StateHashV2
			err = sh2.UnmarshalJSON(js)
			assert.NoError(t, err)
			assert.Equal(t, sh, sh2)
		})
	}
}

func TestStateHashV1BinaryRoundTrip(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			sh := randomStateHashV1()
			data, err := sh.MarshalBinary()
			require.NoError(t, err)
			var sh2 StateHashV1
			err = sh2.UnmarshalBinary(data)
			assert.NoError(t, err)
			assert.Equal(t, sh, sh2)
		})
	}
}

func TestStateHashV2BinaryRoundTrip(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			sh := randomStateHashV2()
			data, err := sh.MarshalBinary()
			require.NoError(t, err)
			var sh2 StateHashV2
			err = sh2.UnmarshalBinary(data)
			assert.NoError(t, err)
			assert.Equal(t, sh, sh2)
		})
	}
}

func TestStateHashBinaryRoundTrip(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			activated := randomBool()
			sh, err := NewLegacyStateHash(randomBlockID(), randomFieldsHashesV1(),
				LegacyStateHashFeatureActivated{
					FinalityActivated: activated,
				},
				LegacyStateHashV2Opt(randomDigest(), randomDigest()),
			)
			require.NoError(t, err)
			data, err := sh.MarshalBinary()
			require.NoError(t, err)
			sh2 := EmptyLegacyStateHash(activated)
			err = sh2.UnmarshalBinary(data)
			assert.NoError(t, err)
			assert.Equal(t, sh, sh2)
		})
	}
}

func TestStateHashJSONRoundTrip(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			activated := randomBool()
			sh, err := NewLegacyStateHash(randomBlockID(), randomFieldsHashesV1(),
				LegacyStateHashFeatureActivated{
					FinalityActivated: activated,
				},
				LegacyStateHashV2Opt(randomDigest(), randomDigest()),
			)
			require.NoError(t, err)
			js, err := sh.MarshalJSON()
			require.NoError(t, err)
			sh2 := EmptyLegacyStateHash(activated)
			err = sh2.UnmarshalJSON(js)
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

func TestStateHashV2_GenerateSumHashScalaCompatibility(t *testing.T) {
	/* Output from Scala test com/wavesplatform/state/StateHashSpec.scala:138
	PrevHash: 46e2hSbVy6YNqx4GH2ZwJW66jMD6FgXzirAUHDD6mVGi
	StateHash: StateHash(3jiGZ5Wiyhm2tubLEgWgnh5eSSjJQqRnTXtMXE2y5HL8,
		HashMap(
			WavesBalance -> 3PhZ3CqdvDR58QGE62gVJFm5pZ6Q5CMpSLWV3KxVkAT7,
			LeaseBalance -> 59QG6ZmcCkLmNuuPLxp2ifNZcr4BzMCahtKQ5iqyM1kJ,
			AssetBalance -> 6CbFygrWrb31bRy3M9BrFry4DZoxN1FCCRBGk5vdwf4S,
			LeaseStatus -> AGLak7NRU4Q6dPWch4nsNbc7iBJMpPUy5agxUW55aLja,
			NextCommittedGenerators -> Gni1oXsHrtK8wSEuRDeZ9qpF8UpKj41HGEWaYSj9bCyC,
			DataEntry -> DcBnRPoAFXhM5nXKmEWTMPrMWhyceWt9FHypcJCJ6UKx,
			Sponsorship -> 3mYNS5c9pEJ6LbwSQh9eevfjDsZAU78KoHX6ct22qBK8,
			AccountScript -> AMrxWar34wJdGWjDj2peT2c1itiPaPwY81hU32hyrB88,
			Alias -> 46e2hSbVy6YNqx4GH2ZwJW66jMD6FgXzirAUHDD6mVGi,
			AssetScript -> H8V5TrNNmwCU1erqVXmQbLoi9b4kd5iJSpMmvJ7CXeyf
		)
	)
	TotalHash: 3jiGZ5Wiyhm2tubLEgWgnh5eSSjJQqRnTXtMXE2y5HL8
	*/
	sh := StateHashV2{
		FieldsHashesV2: FieldsHashesV2{
			FieldsHashesV1: FieldsHashesV1{
				WavesBalanceHash:  crypto.MustDigestFromBase58("3PhZ3CqdvDR58QGE62gVJFm5pZ6Q5CMpSLWV3KxVkAT7"),
				AssetBalanceHash:  crypto.MustDigestFromBase58("6CbFygrWrb31bRy3M9BrFry4DZoxN1FCCRBGk5vdwf4S"),
				DataEntryHash:     crypto.MustDigestFromBase58("DcBnRPoAFXhM5nXKmEWTMPrMWhyceWt9FHypcJCJ6UKx"),
				AccountScriptHash: crypto.MustDigestFromBase58("AMrxWar34wJdGWjDj2peT2c1itiPaPwY81hU32hyrB88"),
				AssetScriptHash:   crypto.MustDigestFromBase58("H8V5TrNNmwCU1erqVXmQbLoi9b4kd5iJSpMmvJ7CXeyf"),
				LeaseBalanceHash:  crypto.MustDigestFromBase58("59QG6ZmcCkLmNuuPLxp2ifNZcr4BzMCahtKQ5iqyM1kJ"),
				LeaseStatusHash:   crypto.MustDigestFromBase58("AGLak7NRU4Q6dPWch4nsNbc7iBJMpPUy5agxUW55aLja"),
				SponsorshipHash:   crypto.MustDigestFromBase58("3mYNS5c9pEJ6LbwSQh9eevfjDsZAU78KoHX6ct22qBK8"),
				AliasesHash:       crypto.MustDigestFromBase58("46e2hSbVy6YNqx4GH2ZwJW66jMD6FgXzirAUHDD6mVGi"),
			},
			GeneratorsHash: crypto.MustDigestFromBase58("Gni1oXsHrtK8wSEuRDeZ9qpF8UpKj41HGEWaYSj9bCyC"),
		},
	}
	prevHash := crypto.MustDigestFromBase58("46e2hSbVy6YNqx4GH2ZwJW66jMD6FgXzirAUHDD6mVGi")
	correctSumHash := crypto.MustDigestFromBase58("3jiGZ5Wiyhm2tubLEgWgnh5eSSjJQqRnTXtMXE2y5HL8")
	err := sh.GenerateSumHash(prevHash.Bytes())
	require.NoError(t, err)
	assert.Equal(t, correctSumHash, sh.GetSumHash())
}

func TestStateHashDebug(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			activated := randomBool()
			sh, err := NewLegacyStateHash(randomBlockID(), randomFieldsHashesV1(),
				LegacyStateHashFeatureActivated{
					FinalityActivated: activated,
				},
				LegacyStateHashV2Opt(randomDigest(), randomDigest()),
			)
			require.NoError(t, err)
			h := randomHeight()
			v := randomVersion()
			ss := randomDigest()
			bt := uint64(randomByte())
			dsh, err := NewStateHashDebug(activated, sh, h, v, ss, bt)
			require.NoError(t, err)
			if activated {
				ash, ok := dsh.(*StateHashDebugV2)
				require.True(t, ok)
				assert.Equal(t, h, ash.Height)
				assert.Equal(t, v, ash.Version)
				assert.Equal(t, ss, ash.SnapshotHash)
				assert.Equal(t, bt, ash.BaseTarget)
			} else {
				ash, ok := dsh.(*StateHashDebugV1)
				require.True(t, ok)
				assert.Equal(t, h, ash.Height)
				assert.Equal(t, v, ash.Version)
				assert.Equal(t, ss, ash.SnapshotHash)
			}
			assert.Equal(t, sh, dsh.GetStateHash())
		})
	}
}
