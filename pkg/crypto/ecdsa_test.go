package crypto

import (
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deHex(t require.TestingT, s string) []byte {
	md, err := hex.DecodeString(s)
	require.NoError(t, err)
	return md
}

func digestMessage(t require.TestingT, s string) []byte {
	m := "\u0019Ethereum Signed Message:\n" + strconv.Itoa(len(s)) + s
	d, err := Keccak256([]byte(m))
	require.NoError(t, err)
	return d.Bytes()
}

func TestECRecoverCompressed(t *testing.T) {
	for _, test := range []struct {
		sig    []byte
		digest []byte
		pk     []byte
	}{
		{
			deHex(t, "e6ca6508de09cbb639216743721076bc8beb7bb45e796e0e3422872f9f0fcd362e693be7ca40e2123dd1efaf71ebb94d38052458281ad3b69ec8977c8294928400"),
			deHex(t, "2b350a58f723b94ef3992ad0d3046f2398aef2fe117dc3a36737fb29df4a706a"),
			deHex(t, "038e369e2984373ad623e10960bf6ed54c80aaa019e7b4134153b3f1657e082ca8"),
		},
		{
			deHex(t, "7cbe1dec2d86b86dce325ab1826d2578a5d050ff3e72cfc381255e41179bd71e467a3d3a2f7adf489a658975fbec1b4a83a6b94351519fe3747396bb3b306da000"),
			deHex(t, "2b350a58f723b94ef3992ad0d3046f2398aef2fe117dc3a36737fb29df4a706a"),
			deHex(t, "02a0a8a6f2571ad2424b3a2539ff8203d20b66cd0133d331b7995f7d99cc6844a6"),
		},
		{
			deHex(t, "242480df7d99877aa803fb3b47b522c7db4f287ff1a2374bf3c9ad6ea4b3d85c7262e2a46c098a6fa722c50b025f991f1dcbd03e4159030eb33f58632245c27d00"),
			deHex(t, "caebd49bd1b95797ef2ae6d900772c374552bbffa460cd250cbed4ddbab5b984"),
			deHex(t, "03b03bfe2cd9496596b1551c8e6fa7c7a161818665a26eea9997766e0622fc1b2b"),
		},
		{
			deHex(t, "13214df23498f2276cdae703f2e12bdea3569ef02590d87a97b79d95826e213505911709dc5cab73a388be3ff6642148a16935230be1ceaca0205389b523b4b201"),
			deHex(t, "caebd49bd1b95797ef2ae6d900772c374552bbffa460cd250cbed4ddbab5b984"),
			deHex(t, "02e793cd797b750d87e4fc9163621a469143b7f0d28ebf03278e0cfd58630a9d19"),
		},
		{
			deHex(t, "789a80053e4927d0a898db8e065e948f5cf086e32f9ccaa54c1908e22ac430c62621578113ddbb62d509bf6049b8fb544ab06d36f916685a2eb8e57ffadde02301"),
			deHex(t, "1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8"),
			deHex(t, "039a7df67f79246283fdc93af76d4f8cdd62c4886e8cd870944e817dd0b97934fd"),
		},
	} {
		pk, err := ECDSARecoverPublicKey(test.digest, test.sig)
		require.NoError(t, err)
		assert.ElementsMatch(t, test.pk, pk.SerializeCompressed())
	}
}

func TestECRecoverUncompressed(t *testing.T) {
	for _, test := range []struct {
		sig    []byte
		digest []byte
		pk     []byte
	}{
		{
			deHex(t, "3b163bbd90556272b57c35d1185b46824f8e16ca229bdb36f8dfd5eaaee9420723ef7bc3a6c0236568217aa990617cf292b1bef1e7d1d936fb2faef3d846c5751b"),
			digestMessage(t, "what's up jim"),
			deHex(t, "b580e37844e1308218ad8cf7f0a77f70f822e0cf34db7c26e5b9d976f1e62b06436201eb4a9fdb49486fecc402651e9e3e5dd49cdb9fac6638053b2616ab880e"),
		},
		{
			deHex(t, "848ffb6a07e7ce335a2bfe373f1c17573eac320f658ea8cf07426544f2203e9d52dbba4584b0b6c0ed5333d84074002878082aa938fdf68c43367946b2f615d01b"),
			digestMessage(t, "i am the owner"),
			deHex(t, "f80cb44734ef6eba2cff997ca17d1cfb03a85db1b0aa2101a07184e04a3cde02c0f2ecded2918ccb6b86d568cceed83e9beeb749ff8981a718e495aff30ac446"),
		},
	} {
		pk, err := ECDSARecoverPublicKey(test.digest, test.sig)
		require.NoError(t, err)
		assert.ElementsMatch(t, test.pk, pk.SerializeUncompressed()[1:])
	}
}

func BenchmarkECDSARecoverPublicKey(b *testing.B) {
	d := digestMessage(b, "i am the owner")
	s := deHex(b, "848ffb6a07e7ce335a2bfe373f1c17573eac320f658ea8cf07426544f2203e9d52dbba4584b0b6c0ed5333d84074002878082aa938fdf68c43367946b2f615d01b")
	for i := 0; i < b.N; i++ {
		pk, err := ECDSARecoverPublicKey(d, s)
		b.StopTimer()
		assert.NotNil(b, pk)
		require.NoError(b, err)
		b.StartTimer()
	}
}

func TestECDSASign(t *testing.T) {
	tests := []struct {
		message          string
		privateKeyHex    string
		publicKeyHex     string
		signatureHex     string
		isValidSignature bool
	}{
		{
			message:          "foo",
			privateKeyHex:    "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032",
			publicKeyHex:     "047db227d7094ce215c3a0f57e1bcc732551fe351f94249471934567e0f5dc1bf795962b8cccb87a2eb56b29fbe37d614e2f4c3c45b789ae4f1f51f4cb21972ffd",
			signatureHex:     "d155e94305af7e07dd8c32873e5c03cb95c9e05960ef85be9c07f671da58c73718c19adc397a211aa9e87e519e2038c5a3b658618db335f74f800b8e0cfeef4401",
			isValidSignature: true,
		},
		{
			message:          "bar",
			privateKeyHex:    "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032",
			publicKeyHex:     "047db227d7094ce215c3a0f57e1bcc732551fe351f94249471934567e0f5dc1bf795962b8cccb87a2eb56b29fbe37d614e2f4c3c45b789ae4f1f51f4cb21972ffd",
			signatureHex:     "d127ddb88b9be7e79fe046a2055706f8bde67506baeb69989e0eb57be097f93926bb7f1236ed1a66f6445cf48fb547bec72927ae4724af44e78c303d01a002c600",
			isValidSignature: true,
		},
		{
			message:          "foo",
			privateKeyHex:    "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032",
			publicKeyHex:     "047db227d7094ce215c3a0f57e1bcc732551fe351f94249471934567e0f5dc1bf795962b8cccb87a2eb56b29fbe37d614e2f4c3c45b789ae4f1f51f4cb21972ffd",
			signatureHex:     "f155e94305af7e07dd8c32873e5c03cb95c9e05960ef85be9c07f671da58c73718c19adc397a211aa9e87e519e2038c5a3b658618db335f74f800b8e0cfeef4401",
			isValidSignature: false,
		},
	}
	for _, tc := range tests {
		sk, err := ECDSAPrivateKeyFromHexString(tc.privateKeyHex)
		require.NoError(t, err)
		publicKey, err := ECDSAParsePublicKeyFromHex(tc.publicKeyHex)
		require.NoError(t, err)

		require.Equal(t, publicKey.ToECDSA(), sk.PubKey().ToECDSA())

		message, err := Keccak256([]byte(tc.message))
		require.NoError(t, err)
		signature, err := ECDSASign(message[:], sk)
		require.NoError(t, err, "Sign error:", err)
		if tc.isValidSignature {
			require.Equal(t, tc.signatureHex, hex.EncodeToString(signature))
		} else {
			require.NotEqual(t, tc.signatureHex, hex.EncodeToString(signature))
		}
	}
}
