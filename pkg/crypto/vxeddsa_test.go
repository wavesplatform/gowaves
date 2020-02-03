package crypto

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelSetNew(t *testing.T) {
	for i, test := range []struct {
		protocol string
		set      string
	}{
		{"VEdDSA_25519_SHA512_Elligator2", "021E5645644453415F32353531395F5348413531325F456C6C696761746F723200"},
	} {
		exp, err := hex.DecodeString(test.set)
		require.NoError(t, err, i)
		set := newLabelSet(test.protocol)
		assert.ElementsMatch(t, set, exp, i)
	}
}

func TestLabelSetAdd(t *testing.T) {
	for i, test := range []struct {
		original string
		label    string
		expected string
	}{
		{"021E5645644453415F32353531395F5348413531325F456C6C696761746F723200", "1", "031E5645644453415F32353531395F5348413531325F456C6C696761746F7232000131"},
		{"021E5645644453415F32353531395F5348413531325F456C6C696761746F723200", "2", "031E5645644453415F32353531395F5348413531325F456C6C696761746F7232000132"},
		{"021E5645644453415F32353531395F5348413531325F456C6C696761746F723200", "3", "031E5645644453415F32353531395F5348413531325F456C6C696761746F7232000133"},
		{"021E5645644453415F32353531395F5348413531325F456C6C696761746F723204CAFEBEBE", "1", "031E5645644453415F32353531395F5348413531325F456C6C696761746F723204CAFEBEBE0131"},
	} {
		o, err := hex.DecodeString(test.original)
		require.NoError(t, err, i)
		exp, err := hex.DecodeString(test.expected)
		require.NoError(t, err, i)
		set := addLabel(o, test.label)
		assert.ElementsMatch(t, set, exp, i)
	}
}

func TestVRFSign(t *testing.T) {
	msg, err := hex.DecodeString("5468697320697320756E697175652E")
	require.NoError(t, err)
	sk, err := hex.DecodeString("38611D253BEA85A203805343B74A936D3B13B9E3121453E9740B6B827E337E5D")
	require.NoError(t, err)
	sig, err := hex.DecodeString("5D501685D744424DE3EF5CA49ECDDD880FA7421C975CDF94BAE48CA16EC0899737721200EED1A8B0D2D6852826A1EAB78B0DF27F35B3F3E89C96E7AE3DAAA30F037297547886E554AFFC81DE54B575768FB30493C537ECDD5A87577DEB7D8E03")
	require.NoError(t, err)
	actual, err := generateVRFSignature(nil, sk, msg)
	require.NoError(t, err)
	assert.EqualValues(t, sig, actual)
}

func TestVRFVerify(t *testing.T) {
	msg, err := hex.DecodeString("5468697320697320756E697175652E")
	require.NoError(t, err)
	pk, err := hex.DecodeString("21F7345F56D9602F1523298F4F6FCECB14DDE2D5B9A9B48BCA8242681492B920")
	require.NoError(t, err)
	vrf, err := hex.DecodeString("45DC7B816B01B36CFA1645DCAE8AC9BC8E523CD86D007D19953F03E7D54554A0")
	require.NoError(t, err)
	signature, err := hex.DecodeString("5D501685D744424DE3EF5CA49ECDDD880FA7421C975CDF94BAE48CA16EC0899737721200EED1A8B0D2D6852826A1EAB78B0DF27F35B3F3E89C96E7AE3DAAA30F037297547886E554AFFC81DE54B575768FB30493C537ECDD5A87577DEB7D8E03")
	require.NoError(t, err)

	ok, actual, err := verifyVRFSignature(pk, msg, signature)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.ElementsMatch(t, vrf, actual)
}

func TestVRFSignVerify(t *testing.T) {
	msg, err := hex.DecodeString("CE0827E6381654D3FFBE22F546E00199B5761C1E541108E56D5A66213A1569E969A02B1D27D91553B69984010F25331A13EA62BA53B6F5B86DA11F8C22ABBF11D6839E11626D0FEF191BD2D5251D371F57C53240F7CD2B435BE6213C7C8F36D47F3DE23A")
	require.NoError(t, err)
	skb, err := hex.DecodeString("C80827E6381654D3FFBE22F546E00199B5761C1E541108E56D5A66213A156969")
	require.NoError(t, err)
	var sk SecretKey
	copy(sk[:], skb[:SecretKeySize])
	pk := GeneratePublicKey(sk)
	random, err := hex.DecodeString("B33734A591BAB70644D731530B67E8734C157DD72B796B7FA3D7FF6885D4C122")
	require.NoError(t, err)
	vrf, err := hex.DecodeString("5669EC30C0F39E2696BB048B574236DEFA325D307116D6A89612958793192FF5")
	require.NoError(t, err)
	sigOut, err := generateVRFSignature(random, skb, msg)
	require.NoError(t, err)
	ok, calcVrf, err := verifyVRFSignature(pk[:], msg, sigOut)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.ElementsMatch(t, vrf, calcVrf)
}

func TestComputeVRF(t *testing.T) {
	msg, err := hex.DecodeString("CE0827E6381654D3FFBE22F546E00199B5761C1E541108E56D5A66213A1569E969A02B1D27D91553B69984010F25331A13EA62BA53B6F5B86DA11F8C22ABBF11D6839E11626D0FEF191BD2D5251D371F57C53240F7CD2B435BE6213C7C8F36D47F3DE23A")
	require.NoError(t, err)
	skb, err := hex.DecodeString("C80827E6381654D3FFBE22F546E00199B5761C1E541108E56D5A66213A156969")
	require.NoError(t, err)
	var sk SecretKey
	copy(sk[:], skb[:SecretKeySize])
	pk := GeneratePublicKey(sk)
	random, err := hex.DecodeString("B33734A591BAB70644D731530B67E8734C157DD72B796B7FA3D7FF6885D4C122")
	require.NoError(t, err)
	vrf, err := hex.DecodeString("5669EC30C0F39E2696BB048B574236DEFA325D307116D6A89612958793192FF5")
	require.NoError(t, err)
	sigOut, err := generateVRFSignature(random, skb, msg)
	require.NoError(t, err)
	ok, calcVrf, err := verifyVRFSignature(pk[:], msg, sigOut)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.ElementsMatch(t, vrf, calcVrf)
	compVRF := ComputeVRF(sk, msg)
	assert.ElementsMatch(t, vrf, compVRF)
}

func TestVRFVerificationFailureByMessage(t *testing.T) {
	msg, err := hex.DecodeString("5468697320697320756E697175652E")
	require.NoError(t, err)
	pk, err := hex.DecodeString("21F7345F56D9602F1523298F4F6FCECB14DDE2D5B9A9B48BCA8242681492B920")
	require.NoError(t, err)
	signature, err := hex.DecodeString("5D501685D744424DE3EF5CA49ECDDD880FA7421C975CDF94BAE48CA16EC0899737721200EED1A8B0D2D6852826A1EAB78B0DF27F35B3F3E89C96E7AE3DAAA30F037297547886E554AFFC81DE54B575768FB30493C537ECDD5A87577DEB7D8E03")
	require.NoError(t, err)
	msg[4] ^= 0xff

	ok, vrf, err := verifyVRFSignature(pk, msg, signature)
	assert.False(t, ok)
	assert.Nil(t, vrf)
	assert.NoError(t, err)
}

func TestVRFVerificationFailureByPublicKey(t *testing.T) {
	msg, err := hex.DecodeString("5468697320697320756E697175652E")
	require.NoError(t, err)
	pk, err := hex.DecodeString("21F7345F56D9602F1523298F4F6FCECB14DDE2D5B9A9B48BCA8242681492B920")
	require.NoError(t, err)
	signature, err := hex.DecodeString("5D501685D744424DE3EF5CA49ECDDD880FA7421C975CDF94BAE48CA16EC0899737721200EED1A8B0D2D6852826A1EAB78B0DF27F35B3F3E89C96E7AE3DAAA30F037297547886E554AFFC81DE54B575768FB30493C537ECDD5A87577DEB7D8E03")
	require.NoError(t, err)
	pk[4] ^= 0xff

	ok, vrf, err := verifyVRFSignature(pk, msg, signature)
	assert.False(t, ok)
	assert.Nil(t, vrf)
	assert.NoError(t, err)
}

func TestVRFVerificationFailureBySignature(t *testing.T) {
	msg, err := hex.DecodeString("5468697320697320756E697175652E")
	require.NoError(t, err)
	pk, err := hex.DecodeString("21F7345F56D9602F1523298F4F6FCECB14DDE2D5B9A9B48BCA8242681492B920")
	require.NoError(t, err)
	signature, err := hex.DecodeString("5D501685D744424DE3EF5CA49ECDDD880FA7421C975CDF94BAE48CA16EC0899737721200EED1A8B0D2D6852826A1EAB78B0DF27F35B3F3E89C96E7AE3DAAA30F037297547886E554AFFC81DE54B575768FB30493C537ECDD5A87577DEB7D8E03")
	require.NoError(t, err)

	sig := make([]byte, len(signature))
	copy(sig, signature)
	sig[4] ^= 0xff
	ok, vrf, err := verifyVRFSignature(pk, msg, sig)
	assert.False(t, ok)
	assert.Nil(t, vrf)
	assert.NoError(t, err)

	copy(sig, signature)
	sig[32+4] ^= 0xff
	ok, vrf, err = verifyVRFSignature(pk, msg, sig)
	assert.False(t, ok)
	assert.Nil(t, vrf)
	assert.NoError(t, err)

	copy(sig, signature)
	sig[64+4] ^= 0xff
	ok, vrf, err = verifyVRFSignature(pk, msg, sig)
	assert.False(t, ok)
	assert.Nil(t, vrf)
	assert.NoError(t, err)
}

func TestVRFMultipleRoundTrips(t *testing.T) {
	for i := 0; i < 100; i++ {
		rand.Seed(time.Now().UnixNano())
		ml := rand.Intn(2048)
		msg := make([]byte, ml)
		seed := make([]byte, 256)
		rand.Read(msg)
		rand.Read(seed)
		sk, pk, err := GenerateKeyPair(seed)
		require.NoError(t, err)
		sig1, err := SignVRF(sk, msg)
		require.NoError(t, err)
		assert.NotEmpty(t, sig1)
		sig2, err := SignVRF(sk, msg)
		require.NoError(t, err)
		assert.NotEmpty(t, sig2)
		vrf0 := ComputeVRF(sk, msg)
		ok, vrf1, err := VerifyVRF(pk, msg, sig1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.NotEmpty(t, vrf1)
		ok, vrf2, err := VerifyVRF(pk, msg, sig2)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.NotEmpty(t, vrf2)
		assert.ElementsMatch(t, vrf1, vrf2)
		assert.ElementsMatch(t, vrf0, vrf1)
	}
}

func BenchmarkSignVRF(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			msg := make([]byte, size)
			if _, err := rand.Read(msg); err != nil {
				b.Fatalf("rand.Read(): %v\n", err)
			}
			seed := make([]byte, 32)
			if _, err := rand.Read(seed); err != nil {
				b.Fatalf("rand.Read(): %v\n", err)
			}
			sk := GenerateSecretKey(seed)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				if _, err := SignVRF(sk, msg); err != nil {
					b.Fatalf("SignVRF() failed: %v\n", err)
				}
			}
		})
	}
}

func BenchmarkVerifyVRF(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			msg := make([]byte, size)
			if _, err := rand.Read(msg); err != nil {
				b.Fatalf("rand.Read(): %v\n", err)
			}
			seed := make([]byte, 32)
			if _, err := rand.Read(seed); err != nil {
				b.Fatalf("rand.Read(): %v\n", err)
			}
			sk, pk, err := GenerateKeyPair(seed)
			if err != nil {
				b.Fatalf("GenerateKeyPair() failed: %v\n", err)
			}
			s, err := SignVRF(sk, msg)
			if err != nil {
				b.Fatalf("SignVRF() failed: %v\n", err)
			}
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				ok, _, err := VerifyVRF(pk, msg, s)
				if err != nil {
					b.Fatalf("VerifyVRF() failed: %v\n", err)
				}
				if !ok {
					b.Fatal("VerifyVRF() returned False")
				}
			}
		})
	}
}

func BenchmarkComputeVRF(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			msg := make([]byte, size)
			if _, err := rand.Read(msg); err != nil {
				b.Fatalf("rand.Read(): %v\n", err)
			}
			seed := make([]byte, 32)
			if _, err := rand.Read(seed); err != nil {
				b.Fatalf("rand.Read(): %v\n", err)
			}
			sk, _, err := GenerateKeyPair(seed)
			if err != nil {
				b.Fatalf("GenerateKeyPair() failed: %v\n", err)
			}
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				vrf := ComputeVRF(sk, msg)
				_ = vrf
			}
		})
	}
}
