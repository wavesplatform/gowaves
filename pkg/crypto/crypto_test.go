package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"testing"
)

func testKeccak(t *testing.T, d, h string) {
	data, err := hex.DecodeString(d)
	if assert.NoError(t, err) {
		actual := Keccak256(data)
		expected, err := hex.DecodeString(h)
		if assert.NoError(t, err) {
			assert.Equal(t, expected, actual[:])
		}
	}
}

func TestKeccak1(t *testing.T) {
	const (
		dataString = "0100000000000000000000000000000000000000000000000000000000000000"
		hashString = "48078cfed56339ea54962e72c37c7f588fc4f8e5bc173827ba75cb10a63a96a5"
	)
	testKeccak(t, dataString, hashString)
}

func TestKeccak2(t *testing.T) {
	const (
		dataString = "0000000000"
		hashString = "c41589e7559804ea4a2080dad19d876a024ccb05117835447d72ce08c1d020ec"
	)
	testKeccak(t, dataString, hashString)
}

func TestKeccak3(t *testing.T) {
	const (
		dataString = "64617461"
		hashString = "8f54f1c2d0eb5771cd5bf67a6689fcd6eed9444d91a39e5ef32a9b4ae5ca14ff"
	)
	testKeccak(t, dataString, hashString)
}

func testFastHash(t *testing.T, d, h string) {
	data, err := hex.DecodeString(d)
	if assert.NoError(t, err) {
		expected, err := hex.DecodeString(h)
		if assert.NoError(t, err) {
			actual, err := FastHash(data)
			if assert.NoError(t, err) {
				assert.Equal(t, expected, actual[:])
			}
		}
	}
}

func TestFastHash1(t *testing.T) {
	const (
		dataString = "0100000000000000000000000000000000000000000000000000000000000000"
		hashString = "afbc1c053c2f278e3cbd4409c1c094f184aa459dd2f7fca96d6077730ab9ffe3"
	)
	testFastHash(t, dataString, hashString)
}

func TestFastHash2(t *testing.T) {
	const (
		dataString = "0000000000"
		hashString = "569ed9e4a5463896190447e6ffe37c394c4d77ce470aa29ad762e0286b896832"
	)
	testFastHash(t, dataString, hashString)
}

func TestFastHash3(t *testing.T) {
	const (
		dataString = "64617461"
		hashString = "a035872d6af8639ede962dfe7536b0c150b590f3234a922fb7064cd11971b58e"
	)
	testFastHash(t, dataString, hashString)
}

func testSecureHash(t *testing.T, d, h string) {
	data, err := hex.DecodeString(d)
	if assert.NoError(t, err) {
		expected, err := hex.DecodeString(h)
		if assert.NoError(t, err) {
			actual, err := SecureHash(data)
			if assert.NoError(t, err) {
				assert.Equal(t, expected, actual[:])
			}
		}
	}
}
func TestSecureHash1(t *testing.T) {
	const (
		dataString = "0100000000000000000000000000000000000000000000000000000000000000"
		hashString = "44282d24d307fb66f385e9a814d07b693d17653c5b88d2e9d4e2a3ccc8216e10"
	)
	testSecureHash(t, dataString, hashString)
}

func TestSecureHash2(t *testing.T) {
	const (
		dataString = "0000000000"
		hashString = "c67437bdaf6ed0ce5d3c39eb6dd591d8005fd0c1fb96cb134a6291ab8e1a39ac"
	)
	testSecureHash(t, dataString, hashString)
}

func TestSecureHash3(t *testing.T) {
	const (
		dataString = "64617461"
		hashString = "7a21055775d130cdeb24258834f40cef7d9b0666f9b0f773cdd28ee556551bb0"
	)
	testSecureHash(t, dataString, hashString)
}

func testKeyGeneration(t *testing.T, seed, sk, pk string) {
	accountSeed, err := base58.Decode(seed)
	if assert.NoError(t, err) {
		ask, apk := GenerateKeyPair(accountSeed)
		skBytes, err := base58.Decode(sk)
		if assert.NoError(t, err) {
			var esk SecretKey
			copy(esk[:], skBytes[:SecretKeySize])
			assert.Equal(t, esk, ask)
			pkBytes, err := base58.Decode(pk)
			if assert.NoError(t, err) {
				var epk PublicKey
				copy(epk[:], pkBytes[:PublicKeySize])
				assert.Equal(t, epk, apk)
			}
		}
	}
}

func TestKeyGeneration1(t *testing.T) {
	const (
		accountSeedBase58       = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
		expectedSecretKeyBase58 = "YoLY4iripseWvtMt29sc89oJnjxzodDgQ9REmEPFHkK"
		expectedPublicKeyBase58 = "3qTkgmBYFjdSEtib9C4b3yHiEexyJ59A5ZVjSvXsg569"
	)
	testKeyGeneration(t, accountSeedBase58, expectedSecretKeyBase58, expectedPublicKeyBase58)
}

func TestKeyGeneration2(t *testing.T) {
	const (
		accountSeedBase58       = "f8ypmbNfr6ocg8kJ7F1MaC4A89f672ZY6LETRiAEbrb"
		expectedSecretKeyBase58 = "EW8VJkEfqr1nW835vKWBqWGeAZdLm8hN7MWf9ZePKr1y"
		expectedPublicKeyBase58 = "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw"
	)
	testKeyGeneration(t, accountSeedBase58, expectedSecretKeyBase58, expectedPublicKeyBase58)
}

func TestKeyGeneration3(t *testing.T) {
	const (
		accountSeedBase58       = "ALxYCdqyG6rUmWgjUHUKCmLgxPsXboMTkRdnn8M2Z4bh"
		expectedSecretKeyBase58 = "EVLXAcnJgnV1KUJasidEY4myaKwvh2d3p8CPc6srC32A"
		expectedPublicKeyBase58 = "7CPECZ633JRSM39HrB8axeJMZWixBeo2p9bWfwwVAhYj"
	)
	testKeyGeneration(t, accountSeedBase58, expectedSecretKeyBase58, expectedPublicKeyBase58)
}

func TestKeyGeneration4(t *testing.T) {
	const (
		accountSeedBase58       = "GQV9jSWTuEE8R8VK56mpXUKx8Nnr8eGwWtMHSs2CoiAd"
		expectedSecretKeyBase58 = "98JtkrkqGqunaJqtN7J2kvJeUnvrTkpobDVArGXTVFa1"
		expectedPublicKeyBase58 = "FKKmKKWsVBPFWufcTTJjoQZDjMG9jmgzAbFPjQm9DVj8"
	)
	testKeyGeneration(t, accountSeedBase58, expectedSecretKeyBase58, expectedPublicKeyBase58)
}

func testSignVerify(t *testing.T, m, sk, pk string) {
	messageBytes, err := base58.Decode(m)
	if assert.NoError(t, err) {
		skBytes, err := base58.Decode(sk)
		if assert.NoError(t, err) {
			var s SecretKey
			copy(s[:], skBytes[:DigestSize])
			pkBytes, err := base58.Decode(pk)
			if assert.NoError(t, err) {
				var p PublicKey
				copy(p[:], pkBytes[:DigestSize])
				sig := Sign(s, messageBytes)
				assert.True(t, Verify(p, sig, messageBytes))
			}
		}
	}
}

func TestSign1(t *testing.T) {
	const (
		message   = "Psm3kB61bTJnbBZo3eE6fBGg8vAEAG"
		secretKey = "98JtkrkqGqunaJqtN7J2kvJeUnvrTkpobDVArGXTVFa1"
		publicKey = "FKKmKKWsVBPFWufcTTJjoQZDjMG9jmgzAbFPjQm9DVj8"
	)
	testSignVerify(t, message, secretKey, publicKey)
}

func TestSign2(t *testing.T) {
	const (
		message   = "ZWoJ9uCKXVC3m5LCoG9CsoDX8Q4RY4Syyhq6N9Wv2wrVDRPFMgMXsqp49hXa77Cr4UK8ZMzhP7yxs7QUA21fJyH67qkkKaCHMknGDGifBnY1svZmEndokx8PeatJ5upxGYrC8qhM66bpPFpfPxjUwTG9zTjHgHkjUyTLuC23"
		secretKey = "EW8VJkEfqr1nW835vKWBqWGeAZdLm8hN7MWf9ZePKr1y"
		publicKey = "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw"
	)
	testSignVerify(t, message, secretKey, publicKey)
}

func testVerify(t *testing.T, pk, m, s string) {
	pkBytes, err := base58.Decode(pk)
	if assert.NoError(t, err) {
		var p PublicKey
		copy(p[:], pkBytes[:DigestSize])
		messageBytes, err1 := base58.Decode(m)
		signatureBytes, err2 := base58.Decode(s)
		if assert.NoError(t, err1) && assert.NoError(t, err2) {
			var sig Signature
			copy(sig[:], signatureBytes[:SignatureSize])
			assert.Nil(t, err)
			assert.True(t, Verify(p, sig, messageBytes))
		}
	}
}

func TestVerify1(t *testing.T) {
	const (
		publicKey = "DZUxn4pC7QdYrRqacmaAJghatvnn1Kh1mkE2scZoLuGJ"
		message   = "cC4wvhC1MiVxpHBudaVmVaQCgL8HwoNhdFFxziArV5bbQt9PXaUPxcLuUS9FLsrV1jX5d3927usPwkuGKcvjyCDKB87Gs8wHiSeyMo1Vcx7uU9ExAThA6vxH9FL8JB6ygi86KDMpHsAGAe4HMHzJzBSY6vuTXiRZDq"
		signature = "5M9jF4TyXKALsZbRKWvqjLjphnMxCZ2eymz8HEV1QXYkyNejKUSfeCQJ4JZpSgxKge9pMTwCp6bWXXpWf9tGrp7N"
	)
	testVerify(t, publicKey, message, signature)
}

func TestVerify2(t *testing.T) {
	const (
		publicKey = "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw"
		message   = "31Y6R7pHocjBqfz6zFEt2VSDoBWwCTcMChjEEpkhNAp4Kp67WQ2DZpA2YmcKCMtzvYRRfbPkRw9QiYuSpwxj6NVdrHt9nVm1EUN8kFSuYVqGjDtpSxE6mH1CmNvsUmMgMkovEVa5Z"
		signature = "2MxM5vTBQcEw9TR53CQqF2WpbgshA8PojsGtY2BxqpGtWhDLhHjzYAei8qcKpotamhZR752v8Be3TSoQEYikJ5Wp"
	)
	testVerify(t, publicKey, message, signature)
}

func TestVerify3(t *testing.T) {
	const (
		publicKey = "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw"
		message   = "ZWoJ9uCKXVC3m5LCoG9CsoDX8Q4RY4Syyhq6N9Wv2wrVDRPFMgMXsqp49hXa77Cr4UK8ZMzhP7yxs7QUA21fJyH67qkkKaCHMknGDGifBnY1svZmEndokx8PeatJ5upxGYrC8qhM66bpPFpfPxjUwTG9zTjHgHkjUyTLuC23"
		signature = "kshMdg9J9iP9esY2oKpgqVWY1Ju2g7LAtkVRQnJX8DiaPgaebRL2fzJ9KvZf5gZg5qLJaFS27frhKvWn5AGQmp6"
	)
	testVerify(t, publicKey, message, signature)
}

func TestVerify4(t *testing.T) {
	const (
		publicKey = "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw"
		message   = "ZWoJ9uCKXVC3m5LCoG9CsoDX8Q4RY4Syyhq6N9Wv2wrVDRPFMgMXsqp49hXa77Cr4UK8ZMzhP7yxs7QUA21fJyH67qkkKaCHMknGDGifBnY1svZmEndokx8PeatJ5upxGYrC8qhM66bpPFpfPxjUwTG9zTjHgHkjUyTLuC23"
		signature = "4NskSm9LqD4c5oqUH6S6D7Xwq1oiEa39KTiBdMBQ4GNEDtfwWt6T7kV6Zf99wfB6nboUwBCuATKj2dzPWZUL94hv"
	)
	testVerify(t, publicKey, message, signature)
}

func benchmarkBase58Encode(b *testing.B, size int) {
	bytes := make([]byte, size)
	rand.Read(bytes)
	for n := 0; n < b.N; n++ {
		base58.Encode(bytes)
	}
}

func benchmarkBase58Decode(b *testing.B, size int) {
	bytes := make([]byte, size)
	rand.Read(bytes)
	s := base58.Encode(bytes)
	for n := 0; n < b.N; n++ {
		base58.Decode(s)
	}
}

func BenchmarkBase58Encode64(b *testing.B)   { benchmarkBase58Encode(b, 64) }
func BenchmarkBase58Encode128(b *testing.B)  { benchmarkBase58Encode(b, 128) }
func BenchmarkBase58Encode256(b *testing.B)  { benchmarkBase58Encode(b, 256) }
func BenchmarkBase58Encode512(b *testing.B)  { benchmarkBase58Encode(b, 512) }
func BenchmarkBase58Encode1024(b *testing.B) { benchmarkBase58Encode(b, 1024) }
func BenchmarkBase58Encode2048(b *testing.B) { benchmarkBase58Encode(b, 2048) }

func BenchmarkBase58Decode64(b *testing.B)   { benchmarkBase58Decode(b, 64) }
func BenchmarkBase58Decode128(b *testing.B)  { benchmarkBase58Decode(b, 128) }
func BenchmarkBase58Decode256(b *testing.B)  { benchmarkBase58Decode(b, 256) }
func BenchmarkBase58Decode512(b *testing.B)  { benchmarkBase58Decode(b, 512) }
func BenchmarkBase58Decode1024(b *testing.B) { benchmarkBase58Decode(b, 1024) }
func BenchmarkBase58Decode2048(b *testing.B) { benchmarkBase58Decode(b, 2048) }

func benchmarkSign(b *testing.B, size int) {
	data := make([]byte, size)
	rand.Read(data)
	seed := make([]byte, 32)
	rand.Read(seed)
	sk := GenerateSecretKey(seed)
	for n := 0; n < b.N; n++ {
		Sign(sk, data)
	}
}

func BenchmarkSign64(b *testing.B)   { benchmarkSign(b, 64) }
func BenchmarkSign128(b *testing.B)  { benchmarkSign(b, 128) }
func BenchmarkSign256(b *testing.B)  { benchmarkSign(b, 256) }
func BenchmarkSign512(b *testing.B)  { benchmarkSign(b, 512) }
func BenchmarkSign1024(b *testing.B) { benchmarkSign(b, 1024) }
func BenchmarkSign2048(b *testing.B) { benchmarkSign(b, 2048) }

func benchmarkVerify(b *testing.B, size int) {
	data := make([]byte, size)
	rand.Read(data)
	seed := make([]byte, 32)
	rand.Read(seed)
	sk, pk := GenerateKeyPair(seed)
	s := Sign(sk, data)
	for n := 0; n < b.N; n++ {
		Verify(pk, s, data)
	}
}

func BenchmarkVerify64(b *testing.B)   { benchmarkVerify(b, 64) }
func BenchmarkVerify128(b *testing.B)  { benchmarkVerify(b, 128) }
func BenchmarkVerify256(b *testing.B)  { benchmarkVerify(b, 256) }
func BenchmarkVerify512(b *testing.B)  { benchmarkVerify(b, 512) }
func BenchmarkVerify1024(b *testing.B) { benchmarkVerify(b, 1024) }
func BenchmarkVerify2048(b *testing.B) { benchmarkVerify(b, 2048) }

func benchmarkFastHash(b *testing.B, size int) {
	data := make([]byte, size)
	rand.Read(data)
	for n := 0; n < b.N; n++ {
		FastHash(data)
	}
}

func BenchmarkFastHash64(b *testing.B)   { benchmarkFastHash(b, 64) }
func BenchmarkFastHash128(b *testing.B)  { benchmarkFastHash(b, 128) }
func BenchmarkFastHash256(b *testing.B)  { benchmarkFastHash(b, 256) }
func BenchmarkFastHash512(b *testing.B)  { benchmarkFastHash(b, 512) }
func BenchmarkFastHash1024(b *testing.B) { benchmarkFastHash(b, 1024) }
func BenchmarkFastHash2048(b *testing.B) { benchmarkFastHash(b, 2048) }

func benchmarkSecureHash(b *testing.B, size int) {
	data := make([]byte, size)
	rand.Read(data)
	for n := 0; n < b.N; n++ {
		SecureHash(data)
	}
}

func BenchmarkSecureHash64(b *testing.B)   { benchmarkSecureHash(b, 64) }
func BenchmarkSecureHash128(b *testing.B)  { benchmarkSecureHash(b, 128) }
func BenchmarkSecureHash256(b *testing.B)  { benchmarkSecureHash(b, 256) }
func BenchmarkSecureHash512(b *testing.B)  { benchmarkSecureHash(b, 512) }
func BenchmarkSecureHash1024(b *testing.B) { benchmarkSecureHash(b, 1024) }
func BenchmarkSecureHash2048(b *testing.B) { benchmarkSecureHash(b, 2048) }
