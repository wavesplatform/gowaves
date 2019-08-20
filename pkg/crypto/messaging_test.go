package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTSSharedKey(t *testing.T) {
	seed1 := []byte("1f98af466da54014bdc08bfbaaaf3c67")
	skA, pkA, err := GenerateKeyPair(generateAccountSeed(t, seed1))
	assert.NoError(t, err)
	seed2 := append(seed1, seed1...)
	skB, pkB, err := GenerateKeyPair(generateAccountSeed(t, seed2))
	assert.NoError(t, err)
	assert.Equal(t, "881b10c3835c4fcd7ec47be7ab4210e233949f2b2a4d9a924c0d26087748374e", hex.EncodeToString(skA[:]))
	assert.Equal(t, "007f416f254cd057850e06290baa7e2d9324261b34c124344e086b79762fa346", hex.EncodeToString(pkA[:]))
	assert.Equal(t, "a07ec01cbe12a902714e7c22e1625eb5b85cf4eef292b760d1c908ab38200965", hex.EncodeToString(skB[:]))
	assert.Equal(t, "ee11c84c2e8796ce751b88838f3f18f09109a8ed857e83da7af965779e061a21", hex.EncodeToString(pkB[:]))
	shared1, err := SharedKey(skA, pkB, []byte("waves"))
	assert.NoError(t, err)
	shared2, err := SharedKey(skB, pkA, []byte("waves"))
	assert.NoError(t, err)
	assert.ElementsMatch(t, shared1, shared2)
	assert.Equal(t, "3e5333596b876d46f1693d305086b981df86632b064f6e6ea789071cff51b9ef", hex.EncodeToString(shared1))
}

func TestSharedKey(t *testing.T) {
	tests := []struct {
		seedA  string
		seedB  string
		prefix string
		skA    string
		pkA    string
		skB    string
		pkB    string
		sk     string
	}{
		{
			"immense learn clever six organ spare squirrel burden unaware fly deputy taste rural cost loyal",
			"seminar similar organ talent matter risk wise furnace museum hedgehog unit avocado marine chalk rug",
			"71f30630fcecb01dfc65",
			"e867f916ecdb888b4275982a3509bcbb33425965f28cf06f71527373a9e5234b",
			"cc9bb1de85c0b0d0b9c587da4f172445778aff6150306ed0a091195ed7435b3e",
			"38545fe060566ada10301a801f0d1832299a50da7abf7d66ec873670913ab248",
			"49c865e1cfebbde04c50e22d72d9c0f386e6cbdb2b4d9284dc33f1ed5104402f",
			"a2c4f8ecdadf346377826602856e7c4aba50b8d129e7be09ea4384b6047c86a3",
		},
		{
			"feed wife situate gasp okay zebra print clean loop early foam green modify giant marine",
			"glove learn bitter hood luggage weekend mammal garbage struggle project wage gym this problem enlist",
			"473426cbb757ffbfb98d",
			"58d6e7b17ba4b5410592a82f2ccebfa2af0b3d53fa342e46ff1695b828d82e4f",
			"8ec0b0745b1b789f3700fc830a058ab01d88f46ccc98ab43a14e9263bbb47d2e",
			"60ecc281c85bf280496282df621399850066b406560260ba957d54ff3c62d140",
			"8ad7966ccb0d9203d6fc5b092f99f4a2474a60184446b33379ee8139f413c618",
			"160402a95f2c35d533d03e75d93347409489b97259a6ebfb1c6f6c53677611b5",
		},
	}
	for _, tc := range tests {
		seedA := generateAccountSeed(t, []byte(tc.seedA))
		skA, pkA, err := GenerateKeyPair(seedA)
		assert.NoError(t, err)
		assert.Equal(t, tc.skA, hex.EncodeToString(skA[:]))
		assert.Equal(t, tc.pkA, hex.EncodeToString(pkA[:]))
		seedB := generateAccountSeed(t, []byte(tc.seedB))
		skB, pkB, err := GenerateKeyPair(seedB)
		assert.NoError(t, err)
		assert.Equal(t, tc.skB, hex.EncodeToString(skB[:]))
		assert.Equal(t, tc.pkB, hex.EncodeToString(pkB[:]))
		prefix, err := hex.DecodeString(tc.prefix)
		require.NoError(t, err)
		shared1, err := SharedKey(skA, pkB, prefix)
		require.NoError(t, err)
		shared2, err := SharedKey(skB, pkA, prefix)
		require.NoError(t, err)
		assert.ElementsMatch(t, shared1, shared2)
		assert.Equal(t, tc.sk, hex.EncodeToString(shared1))
	}
}

func TestPadPKCS7Padding(t *testing.T) {
	tests := []struct {
		msg string
		exp string
	}{
		{"ac", "ac0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f"},
		{"ac60", "ac600e0e0e0e0e0e0e0e0e0e0e0e0e0e"},
		{"ac6051", "ac60510d0d0d0d0d0d0d0d0d0d0d0d0d"},
		{"ac60510e", "ac60510e0c0c0c0c0c0c0c0c0c0c0c0c"},
		{"ac60510e83", "ac60510e830b0b0b0b0b0b0b0b0b0b0b"},
		{"ac60510e8310", "ac60510e83100a0a0a0a0a0a0a0a0a0a"},
		{"ac60510e831051", "ac60510e831051090909090909090909"},
		{"ac60510e83105179", "ac60510e831051790808080808080808"},
		{"ac60510e8310517983", "ac60510e831051798307070707070707"},
		{"ac60510e831051798306", "ac60510e831051798306060606060606"},
		{"ac60510e831051798306a5", "ac60510e831051798306a50505050505"},
		{"ac60510e831051798306a5c0", "ac60510e831051798306a5c004040404"},
		{"ac60510e831051798306a5c034", "ac60510e831051798306a5c034030303"},
		{"ac60510e831051798306a5c034be", "ac60510e831051798306a5c034be0202"},
		{"ac60510e831051798306a5c034bed1", "ac60510e831051798306a5c034bed101"},
		{"ac60510e831051798306a5c034bed1a5", "ac60510e831051798306a5c034bed1a510101010101010101010101010101010"},
		{"ac60510e831051798306a5c034bed1a5fb", "ac60510e831051798306a5c034bed1a5fb0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f"},
	}
	for _, tc := range tests {
		msg, err := hex.DecodeString(tc.msg)
		require.NoError(t, err)
		act, err := padPKCS7Padding(msg)
		require.NoError(t, err)
		assert.Equal(t, tc.exp, hex.EncodeToString(act))
	}
}

func TestTrimPKCS7Padding(t *testing.T) {
	tests := []struct {
		exp string
		msg string
	}{
		{"ac", "ac0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f"},
		{"ac60", "ac600e0e0e0e0e0e0e0e0e0e0e0e0e0e"},
		{"ac6051", "ac60510d0d0d0d0d0d0d0d0d0d0d0d0d"},
		{"ac60510e", "ac60510e0c0c0c0c0c0c0c0c0c0c0c0c"},
		{"ac60510e83", "ac60510e830b0b0b0b0b0b0b0b0b0b0b"},
		{"ac60510e8310", "ac60510e83100a0a0a0a0a0a0a0a0a0a"},
		{"ac60510e831051", "ac60510e831051090909090909090909"},
		{"ac60510e83105179", "ac60510e831051790808080808080808"},
		{"ac60510e8310517983", "ac60510e831051798307070707070707"},
		{"ac60510e831051798306", "ac60510e831051798306060606060606"},
		{"ac60510e831051798306a5", "ac60510e831051798306a50505050505"},
		{"ac60510e831051798306a5c0", "ac60510e831051798306a5c004040404"},
		{"ac60510e831051798306a5c034", "ac60510e831051798306a5c034030303"},
		{"ac60510e831051798306a5c034be", "ac60510e831051798306a5c034be0202"},
		{"ac60510e831051798306a5c034bed1", "ac60510e831051798306a5c034bed101"},
		{"ac60510e831051798306a5c034bed1a5", "ac60510e831051798306a5c034bed1a510101010101010101010101010101010"},
		{"ac60510e831051798306a5c034bed1a5fb", "ac60510e831051798306a5c034bed1a5fb0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f"},
	}
	for _, tc := range tests {
		msg, err := hex.DecodeString(tc.msg)
		require.NoError(t, err)
		act, err := trimPKCS7Padding(msg)
		require.NoError(t, err)
		assert.Equal(t, tc.exp, hex.EncodeToString(act))
	}
}

func TestEncryptAESECB(t *testing.T) {
	tests := []struct {
		msg string
		key string
		enc string
	}{
		{"ac60510e831051798306a5c034bed1a5fb3d90483af6a469a6cbe0b4590d6d35", "7d1d6cf7993d529ae0cca883af6b6cd45e3bff3b3b3af5189ce08b25bd2707c3", "74c2d4c188f8476637bac66c0fd079f77177f690eaf58e894952a7e5e2a43a4bca1e1a0e30f31b7f362f401c8e8bc5a9"},
		{"610a40ba4d047dc589356e6f9e5ba2deac46107fe05883469ec7ed8ac190a90f", "0a95f11412b827aafbc92810d9a55848bd00d55508dcaa942ecf3beeff787ced", "c3e4c674e48ec36894e4001ce4b04775f8916630c556c8a386e81bfb5b776b1f40f39e1a963a3ad63a3b4f0275877c0f"},
	}
	for _, tc := range tests {
		msg, err := hex.DecodeString(tc.msg)
		require.NoError(t, err)
		key, err := hex.DecodeString(tc.key)
		require.NoError(t, err)
		enc, err := encryptAESECB(msg, key)
		require.NoError(t, err)
		assert.Equal(t, tc.enc, hex.EncodeToString(enc))
	}
}

func TestDecryptAESECB(t *testing.T) {
	tests := []struct {
		msg string
		key string
		enc string
	}{
		{"ac60510e831051798306a5c034bed1a5fb3d90483af6a469a6cbe0b4590d6d35", "7d1d6cf7993d529ae0cca883af6b6cd45e3bff3b3b3af5189ce08b25bd2707c3", "74c2d4c188f8476637bac66c0fd079f77177f690eaf58e894952a7e5e2a43a4bca1e1a0e30f31b7f362f401c8e8bc5a9"},
		{"610a40ba4d047dc589356e6f9e5ba2deac46107fe05883469ec7ed8ac190a90f", "0a95f11412b827aafbc92810d9a55848bd00d55508dcaa942ecf3beeff787ced", "c3e4c674e48ec36894e4001ce4b04775f8916630c556c8a386e81bfb5b776b1f40f39e1a963a3ad63a3b4f0275877c0f"},
	}
	for _, tc := range tests {
		enc, err := hex.DecodeString(tc.enc)
		require.NoError(t, err)
		key, err := hex.DecodeString(tc.key)
		require.NoError(t, err)
		msg, err := decryptAESECB(enc, key)
		require.NoError(t, err)
		assert.Equal(t, tc.msg, hex.EncodeToString(msg))
	}
}

func TestDecrypt(t *testing.T) {
	tests := []struct {
		msg string
		sk  string
		enc string
	}{
		{"message", "a2c4f8ecdadf346377826602856e7c4aba50b8d129e7be09ea4384b6047c86a3", "0159443db40c951e43a2770560e5ce3970f81679167c7906fed9647c0ad2e3a33b386b110eb2aaea2e4be1db6097922ecfda9bc34f2f7f77f92dcde0f6d64fb130aeda126a426b8628ca57e3e4f04e676b670a55a37c675e8cadcdd533e8fd76eadc5e617032d3e85e7a7ec923f50cf610f9c4b8b83d0b6c4d02944951da6d388dbe4244c50072f1"},
		{"message", "160402a95f2c35d533d03e75d93347409489b97259a6ebfb1c6f6c53677611b5", "015b66d9b0ce12026b448d6fe88b28e6b602389e1c18ec4b014f32eeb5dc0d6ed2140ee79e722084e3bebbd3570cc3cea5516706d0048473e1b6ee849b2a158ad3bc5326a6a536a7263dbcf7aaa409d248341181c8cc3f1b74f06794e3e88dfb85432b413f49e7771b20f26f84901962a0ca7a8e2f03f91d6891037b3737352b29927aa8a906f8b4"},
	}
	for _, tc := range tests {
		key, err := hex.DecodeString(tc.sk)
		require.NoError(t, err)
		enc, err := hex.DecodeString(tc.enc)
		require.NoError(t, err)
		msg, err := Decrypt(key, enc)
		require.NoError(t, err)
		assert.Equal(t, tc.msg, string(msg))
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	tests := []struct {
		msg string
		sk  string
	}{
		{"message", "a2c4f8ecdadf346377826602856e7c4aba50b8d129e7be09ea4384b6047c86a3"},
		{"The quick brown fox jumps over the lazy dog", "160402a95f2c35d533d03e75d93347409489b97259a6ebfb1c6f6c53677611b5"},
		{"🏳️‍🌈🇩🇪🏳️‍🌈🇩🇪", "160402a95f2c35d533d03e75d93347409489b97259a6ebfb1c6f6c53677611b5"},
	}
	for _, tc := range tests {
		key, err := hex.DecodeString(tc.sk)
		require.NoError(t, err)
		enc, err := Encrypt(key, []byte(tc.msg))
		require.NoError(t, err)
		act, err := Decrypt(key, enc)
		require.NoError(t, err)
		assert.Equal(t, tc.msg, string(act))
	}
}

func generateAccountSeed(t *testing.T, seed []byte) []byte {
	n := make([]byte, 4)
	s := append(n, []byte(seed)...)
	accA, err := SecureHash(s)
	require.NoError(t, err)
	return accA[:]
}
