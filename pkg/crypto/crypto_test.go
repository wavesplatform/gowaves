package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKeccak(t *testing.T) {
	tests := []struct {
		data string
		hash string
	}{
		{"0100000000000000000000000000000000000000000000000000000000000000", "48078cfed56339ea54962e72c37c7f588fc4f8e5bc173827ba75cb10a63a96a5"},
		{"0000000000", "c41589e7559804ea4a2080dad19d876a024ccb05117835447d72ce08c1d020ec"},
		{"64617461", "8f54f1c2d0eb5771cd5bf67a6689fcd6eed9444d91a39e5ef32a9b4ae5ca14ff"},
	}
	for _, tc := range tests {
		data, err := hex.DecodeString(tc.data)
		if assert.NoError(t, err) {
			actual := Keccak256(data)
			expected, err := hex.DecodeString(tc.hash)
			if assert.NoError(t, err) {
				assert.Equal(t, expected, actual[:])
			}
		}
	}
}

func TestFastHash(t *testing.T) {
	tests := []struct {
		data string
		hash string
	}{
		{"0100000000000000000000000000000000000000000000000000000000000000", "afbc1c053c2f278e3cbd4409c1c094f184aa459dd2f7fca96d6077730ab9ffe3"},
		{"0000000000", "569ed9e4a5463896190447e6ffe37c394c4d77ce470aa29ad762e0286b896832"},
		{"64617461", "a035872d6af8639ede962dfe7536b0c150b590f3234a922fb7064cd11971b58e"},
	}
	for _, tc := range tests {
		data, err := hex.DecodeString(tc.data)
		if assert.NoError(t, err) {
			expected, err := hex.DecodeString(tc.hash)
			if assert.NoError(t, err) {
				actual, err := FastHash(data)
				if assert.NoError(t, err) {
					assert.Equal(t, expected, actual[:])
				}
			}
		}
	}
}

func TestSecureHash(t *testing.T) {
	tests := []struct {
		data string
		hash string
	}{
		{"0100000000000000000000000000000000000000000000000000000000000000", "44282d24d307fb66f385e9a814d07b693d17653c5b88d2e9d4e2a3ccc8216e10"},
		{"0000000000", "c67437bdaf6ed0ce5d3c39eb6dd591d8005fd0c1fb96cb134a6291ab8e1a39ac"},
		{"64617461", "7a21055775d130cdeb24258834f40cef7d9b0666f9b0f773cdd28ee556551bb0"},
	}
	for _, tc := range tests {
		data, err := hex.DecodeString(tc.data)
		if assert.NoError(t, err) {
			expected, err := hex.DecodeString(tc.hash)
			if assert.NoError(t, err) {
				actual, err := SecureHash(data)
				if assert.NoError(t, err) {
					assert.Equal(t, expected, actual[:])
				}
			}
		}
	}
}

func TestKeyGeneration(t *testing.T) {
	tests := []struct {
		seed       string
		expectedSK string
		expectedPK string
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "YoLY4iripseWvtMt29sc89oJnjxzodDgQ9REmEPFHkK", "3qTkgmBYFjdSEtib9C4b3yHiEexyJ59A5ZVjSvXsg569"},
		{"f8ypmbNfr6ocg8kJ7F1MaC4A89f672ZY6LETRiAEbrb", "EW8VJkEfqr1nW835vKWBqWGeAZdLm8hN7MWf9ZePKr1y", "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw"},
		{"ALxYCdqyG6rUmWgjUHUKCmLgxPsXboMTkRdnn8M2Z4bh", "EVLXAcnJgnV1KUJasidEY4myaKwvh2d3p8CPc6srC32A", "7CPECZ633JRSM39HrB8axeJMZWixBeo2p9bWfwwVAhYj"},
		{"GQV9jSWTuEE8R8VK56mpXUKx8Nnr8eGwWtMHSs2CoiAd", "98JtkrkqGqunaJqtN7J2kvJeUnvrTkpobDVArGXTVFa1", "FKKmKKWsVBPFWufcTTJjoQZDjMG9jmgzAbFPjQm9DVj8"},
	}
	for _, tc := range tests {
		accountSeed, err := base58.Decode(tc.seed)
		if assert.NoError(t, err) {
			ask, apk := GenerateKeyPair(accountSeed)
			skBytes, err := base58.Decode(tc.expectedSK)
			if assert.NoError(t, err) {
				var esk SecretKey
				copy(esk[:], skBytes[:SecretKeySize])
				assert.Equal(t, esk, ask)
				pkBytes, err := base58.Decode(tc.expectedPK)
				if assert.NoError(t, err) {
					var epk PublicKey
					copy(epk[:], pkBytes[:PublicKeySize])
					assert.Equal(t, epk, apk)
				}
			}
		}
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		message string
		sk      string
		pk      string
	}{
		{"Psm3kB61bTJnbBZo3eE6fBGg8vAEAG", "98JtkrkqGqunaJqtN7J2kvJeUnvrTkpobDVArGXTVFa1", "FKKmKKWsVBPFWufcTTJjoQZDjMG9jmgzAbFPjQm9DVj8"},
		{"ZWoJ9uCKXVC3m5LCoG9CsoDX8Q4RY4Syyhq6N9Wv2wrVDRPFMgMXsqp49hXa77Cr4UK8ZMzhP7yxs7QUA21fJyH67qkkKaCHMknGDGifBnY1svZmEndokx8PeatJ5upxGYrC8qhM66bpPFpfPxjUwTG9zTjHgHkjUyTLuC23", "EW8VJkEfqr1nW835vKWBqWGeAZdLm8hN7MWf9ZePKr1y", "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw"},
	}
	for _, tc := range tests {
		messageBytes, err := base58.Decode(tc.message)
		if assert.NoError(t, err) {
			skBytes, err := base58.Decode(tc.sk)
			if assert.NoError(t, err) {
				var s SecretKey
				copy(s[:], skBytes[:DigestSize])
				pkBytes, err := base58.Decode(tc.pk)
				if assert.NoError(t, err) {
					var p PublicKey
					copy(p[:], pkBytes[:DigestSize])
					sig := Sign(s, messageBytes)
					assert.True(t, Verify(p, sig, messageBytes))
				}
			}
		}
	}
}

func TestVerify(t *testing.T) {
	tests := []struct {
		pk      string
		message string
		sig     string
	}{
		{"DZUxn4pC7QdYrRqacmaAJghatvnn1Kh1mkE2scZoLuGJ", "cC4wvhC1MiVxpHBudaVmVaQCgL8HwoNhdFFxziArV5bbQt9PXaUPxcLuUS9FLsrV1jX5d3927usPwkuGKcvjyCDKB87Gs8wHiSeyMo1Vcx7uU9ExAThA6vxH9FL8JB6ygi86KDMpHsAGAe4HMHzJzBSY6vuTXiRZDq", "5M9jF4TyXKALsZbRKWvqjLjphnMxCZ2eymz8HEV1QXYkyNejKUSfeCQJ4JZpSgxKge9pMTwCp6bWXXpWf9tGrp7N"},
		{"CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw", "31Y6R7pHocjBqfz6zFEt2VSDoBWwCTcMChjEEpkhNAp4Kp67WQ2DZpA2YmcKCMtzvYRRfbPkRw9QiYuSpwxj6NVdrHt9nVm1EUN8kFSuYVqGjDtpSxE6mH1CmNvsUmMgMkovEVa5Z", "2MxM5vTBQcEw9TR53CQqF2WpbgshA8PojsGtY2BxqpGtWhDLhHjzYAei8qcKpotamhZR752v8Be3TSoQEYikJ5Wp"},
		{"CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw", "ZWoJ9uCKXVC3m5LCoG9CsoDX8Q4RY4Syyhq6N9Wv2wrVDRPFMgMXsqp49hXa77Cr4UK8ZMzhP7yxs7QUA21fJyH67qkkKaCHMknGDGifBnY1svZmEndokx8PeatJ5upxGYrC8qhM66bpPFpfPxjUwTG9zTjHgHkjUyTLuC23", "kshMdg9J9iP9esY2oKpgqVWY1Ju2g7LAtkVRQnJX8DiaPgaebRL2fzJ9KvZf5gZg5qLJaFS27frhKvWn5AGQmp6"},
		{"CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw", "ZWoJ9uCKXVC3m5LCoG9CsoDX8Q4RY4Syyhq6N9Wv2wrVDRPFMgMXsqp49hXa77Cr4UK8ZMzhP7yxs7QUA21fJyH67qkkKaCHMknGDGifBnY1svZmEndokx8PeatJ5upxGYrC8qhM66bpPFpfPxjUwTG9zTjHgHkjUyTLuC23", "4NskSm9LqD4c5oqUH6S6D7Xwq1oiEa39KTiBdMBQ4GNEDtfwWt6T7kV6Zf99wfB6nboUwBCuATKj2dzPWZUL94hv"},
	}
	for _, tc := range tests {
		pkBytes, err := base58.Decode(tc.pk)
		if assert.NoError(t, err) {
			var p PublicKey
			copy(p[:], pkBytes[:DigestSize])
			messageBytes, err1 := base58.Decode(tc.message)
			signatureBytes, err2 := base58.Decode(tc.sig)
			if assert.NoError(t, err1) && assert.NoError(t, err2) {
				var sig Signature
				copy(sig[:], signatureBytes[:SignatureSize])
				assert.Nil(t, err)
				assert.True(t, Verify(p, sig, messageBytes))
			}
		}
	}
}

func BenchmarkBase58Decode(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			bytes := make([]byte, size)
			rand.Read(bytes)
			s := base58.Encode(bytes)
			for n := 0; n < b.N; n++ {
				base58.Decode(s)
			}
		})
	}
}

func BenchmarkBase58Encode(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			bytes := make([]byte, size)
			rand.Read(bytes)
			for n := 0; n < b.N; n++ {
				base58.Encode(bytes)
			}
		})
	}
}

func BenchmarkSign(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)
			seed := make([]byte, 32)
			rand.Read(seed)
			sk := GenerateSecretKey(seed)
			for n := 0; n < b.N; n++ {
				Sign(sk, data)
			}
		})
	}
}

func BenchmarkVerify(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)
			seed := make([]byte, 32)
			rand.Read(seed)
			sk, pk := GenerateKeyPair(seed)
			s := Sign(sk, data)
			for n := 0; n < b.N; n++ {
				Verify(pk, s, data)
			}
		})
	}
}

func BenchmarkFastHash(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)
			for n := 0; n < b.N; n++ {
				FastHash(data)
			}
		})
	}
}

func BenchmarkSecureHash(b *testing.B) {
	for size := 64; size <= 2048; size *= 2 {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)
			for n := 0; n < b.N; n++ {
				SecureHash(data)
			}
		})
	}
}

func TestSecretKey_String(t *testing.T) {
	k := "YoLY4iripseWvtMt29sc89oJnjxzodDgQ9REmEPFHkK"
	secretKey, err := NewSecretKeyFromBase58(k)
	require.Nil(t, err)
	assert.Equal(t, k, secretKey.String())
}
