package proto

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"testing"
)

func TestOrderType_String(t *testing.T) {
	ot0 := Buy
	assert.Equal(t, "buy", ot0.String())
	ot1 := Sell
	assert.Equal(t, "sell", ot1.String())
}

func TestOrderType_MarshalJSON(t *testing.T) {
	j, err := Buy.MarshalJSON()
	assert.Nil(t, err)
	assert.Equal(t, "\"buy\"", string(j))
	j, err = Sell.MarshalJSON()
	assert.Nil(t, err)
	assert.Equal(t, "\"sell\"", string(j))
}

func TestOrderType_UnmarshalJSON(t *testing.T) {
	var x OrderType
	err := x.UnmarshalJSON([]byte("\"buy\""))
	assert.Nil(t, err)
	assert.Equal(t, Buy, x)
	err = x.UnmarshalJSON([]byte("\"sell\""))
	assert.Nil(t, err)
	assert.Equal(t, Sell, x)
}

func TestMainNetOrder(t *testing.T) {
	tests := []struct {
		sender      string
		matcher     string
		id          string
		sig         string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		price       uint64
		amount      uint64
		timestamp   uint64
		expiration  uint64
		fee         uint64
	}{
		{"6s3F3S1ZmdJ2B25EqHWgNUSfeHtMaRZJ4RGEB5hgS7QM", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "3UFtKxSe4S7haXgW8398oZgKKhhJCkcc96GU6PXaKwjy",
			"kD3EXkJcj9doZjZ1Xvnh7Jko4dpd6FYQJhyVrBqZsTgPCvRpRxprkhijScKqZEeVsnNqiwrdMEKXEEcMhZdpdgF", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck",
			Buy, 674641, 177872, 1537539673823, 1537543273823, 300000},
	}
	for _, tc := range tests {
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		spk, _ := crypto.NewPublicKeyFromBase58(tc.sender)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		if o, err := NewUnsignedOrder(spk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, tc.timestamp, tc.expiration, tc.fee); assert.NoError(t, err) {
			if b, err := o.bodyMarshalBinary(); assert.NoError(t, err) {
				d, _ := crypto.FastHash(b)
				assert.Equal(t, id, d)
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}

	}
}

func TestOrderValidations(t *testing.T) {
	tests := []struct {
		price  uint64
		amount uint64
		fee    uint64
		err    string
	}{
		{0, 20, 30, "price should be positive"},
		{10, 0, 30, "amount should be positive"},
		{10, 20, 0, "matcher's fee should be positive"},
	}
	spk, _ := crypto.NewPublicKeyFromBase58("6s3F3S1ZmdJ2B25EqHWgNUSfeHtMaRZJ4RGEB5hgS7QM")
	mpk, _ := crypto.NewPublicKeyFromBase58("7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy")
	aa, _ := NewOptionalAssetFromString("8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS")
	pa, _ := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	for _, tc := range tests {
		_, err := NewUnsignedOrder(spk, mpk, *aa, *pa, Buy, tc.price, tc.amount, 0, 0, tc.fee)
		assert.EqualError(t, err, tc.err)
	}
}

//func TestOrderSigningRoundTrip(t *testing.T) {
//	const (
//		senderSeed  = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
//		matcherPK   = "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy"
//		amountAsset = "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS"
//		priceAsset  = "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt"
//	)
//	mpk, _ := base58.Decode(matcherPK)
//	aa, _ := NewAssetFromString(amountAsset)
//	pa, _ := NewAssetFromString(priceAsset)
//	seed, _ := base58.Decode(senderSeed)
//	ssk, spk := GenerateKeyPair(seed)
//	ts := time.Now().Unix() * 1000
//	ex := ts + 100*1000
//	o, err := NewOrder(spk, mpk, *aa, *pa, Sell, 1000, 100, ts, ex, 10)
//	assert.Nil(t, err)
//	so, err := o.Sign(ssk)
//	assert.Nil(t, err)
//	assert.Equal(t, *o, so.Order)
//	v, err := so.Verify(spk)
//	assert.Nil(t, err)
//	assert.True(t, v)
//}
//
//func TestOrderToJSON(t *testing.T) {
//	const (
//		senderSeed  = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
//		matcherPK   = "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy"
//		amountAsset = "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS"
//		priceAsset  = "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt"
//	)
//	mpk, _ := base58.Decode(matcherPK)
//	aa, _ := NewAssetFromString(amountAsset)
//	pa, _ := NewAssetFromString(priceAsset)
//	seed, _ := base58.Decode(senderSeed)
//	ssk, spk := GenerateKeyPair(seed)
//	ts := time.Now().Unix() * 1000
//	ex := ts + 100*1000
//	o, err := NewOrder(spk, mpk, *aa, *pa, Sell, 1000, 100, ts, ex, 10)
//	j, err := json.Marshal(o)
//	assert.Nil(t, err)
//	assert.NotNil(t, j)
//	ej := fmt.Sprintf("{\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":\"%s\",\"priceAsset\":\"%s\"},\"orderType\":\"sell\",\"price\":1000,\"amount\":100,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":10}",
//		base58.Encode(spk), matcherPK, amountAsset, priceAsset, ts, ex)
//	assert.Equal(t, ej, string(j))
//	so, err := o.Sign(ssk)
//	sj, err := json.Marshal(so)
//	assert.Nil(t, err)
//	esj := fmt.Sprintf("{\"id\":\"%s\",\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":\"%s\",\"priceAsset\":\"%s\"},\"orderType\":\"sell\",\"price\":1000,\"amount\":100,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":10,\"signature\":\"%s\"}",
//		base58.Encode(so.ID), base58.Encode(spk), matcherPK, amountAsset, priceAsset, ts, ex, base58.Encode(so.Signature))
//	assert.Equal(t, esj, string(sj))
//}
