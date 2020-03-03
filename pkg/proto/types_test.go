package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
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
		o := NewUnsignedOrderV1(spk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, tc.timestamp, tc.expiration, tc.fee)
		if b, err := o.BodyMarshalBinary(); assert.NoError(t, err) {
			d, _ := crypto.FastHash(b)
			assert.Equal(t, id, d)
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestOrderV1Validations(t *testing.T) {
	aa, err := NewOptionalAssetFromString("8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS")
	require.NoError(t, err)
	pa, err := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	require.NoError(t, err)
	waves, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)
	tests := []struct {
		amountAsset OptionalAsset
		priceAsset  OptionalAsset
		orderType   OrderType
		price       uint64
		amount      uint64
		fee         uint64
		ts          uint64
		exp         uint64
		err         string
	}{
		{*aa, *aa, Buy, 1234, 5678, 90, 1, 1, "invalid asset pair"},
		{*aa, *pa, Sell, 0, 20, 30, 1, 1, "price should be positive"},
		{*aa, *pa, Buy, math.MaxInt64 + 1, 20, 30, 1, 1, "price is too big"},
		{*aa, *pa, Sell, 10, 0, 30, 1, 1, "amount should be positive"},
		{*aa, *pa, Buy, 10, math.MaxInt64 + 1, 30, 1, 1, "amount is too big"},
		{*aa, *pa, Sell, 10, MaxOrderAmount + 1, 30, 1, 1, "amount is larger than maximum allowed"},
		{*aa, *pa, Buy, 10, 20, 0, 1, 1, "matcher's fee should be positive"},
		{*aa, *pa, Sell, 10, 20, math.MaxInt64 + 2, 1, 1, "matcher's fee is too big"},
		{*aa, *pa, Sell, 10, 20, MaxOrderAmount + 1, 1, 1, "matcher's fee is larger than maximum allowed"},
		{*aa, *waves, Buy, math.MaxInt64, MaxOrderAmount, 1000, 1, 1, "spend amount is too large"},
		{*aa, *waves, Buy, 1, 1, 1000, 1, 1, "spend amount should be positive"},
		{*aa, *waves, Sell, math.MaxInt64, MaxOrderAmount, 1000, 1, 1, "receive amount is too large"},
		{*aa, *waves, Sell, 1, 1, 1000, 1, 1, "receive amount should be positive"},
		{*aa, *waves, Buy, math.MaxInt64 / (100 * PriceConstant), MaxOrderAmount, MaxOrderAmount, 1, 1, "sum of spend asset amount and matcher fee overflows JVM long"},
		{*aa, *pa, Sell, 100000000, 20, 30, 0, 1, "timestamp should be positive"},
		{*aa, *pa, Sell, 100000000, 20, 30, 1, 0, "expiration should be positive"},
	}
	spk, _ := crypto.NewPublicKeyFromBase58("6s3F3S1ZmdJ2B25EqHWgNUSfeHtMaRZJ4RGEB5hgS7QM")
	mpk, _ := crypto.NewPublicKeyFromBase58("7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy")
	for _, tc := range tests {
		o := NewUnsignedOrderV1(spk, mpk, tc.amountAsset, tc.priceAsset, tc.orderType, tc.price, tc.amount, tc.ts, tc.exp, tc.fee)
		v, err := o.Valid()
		assert.False(t, v)
		assert.EqualError(t, err, tc.err)
	}
}

func TestOrderV1BinarySize(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2},
	}
	for _, tc := range tests {
		seed, _ := base58.Decode(tc.seed)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		o := NewUnsignedOrderV1(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee)
		err = o.Sign(MainNetScheme, sk)
		assert.NoError(t, err)
		oBytes, err := o.MarshalBinary()
		assert.NoError(t, err)
		assert.Equal(t, len(oBytes), o.BinarySize())
	}
}

func TestOrderV1SigningRoundTrip(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2},
	}
	for _, tc := range tests {
		seed, _ := base58.Decode(tc.seed)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		o := NewUnsignedOrderV1(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee)
		if err := o.Sign(MainNetScheme, sk); assert.NoError(t, err) {
			if r, err := o.Verify(MainNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
			if b, err := o.MarshalBinary(); assert.NoError(t, err) {
				var ao OrderV1
				if err := ao.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, o.ID, ao.ID)
					assert.Equal(t, o.Signature, ao.Signature)
					assert.Equal(t, o.SenderPK, ao.SenderPK)
					assert.Equal(t, o.MatcherPK, ao.MatcherPK)
					assert.Equal(t, o.AssetPair, ao.AssetPair)
					assert.Equal(t, o.OrderType, ao.OrderType)
					assert.Equal(t, o.Price, ao.Price)
					assert.Equal(t, o.Amount, ao.Amount)
					assert.Equal(t, o.Timestamp, ao.Timestamp)
					assert.Equal(t, o.Expiration, ao.Expiration)
					assert.Equal(t, o.MatcherFee, ao.MatcherFee)
				}
			}
			buf := &bytes.Buffer{}
			s := serializer.New(buf)
			if err := o.Serialize(s); assert.NoError(t, err) {
				var ao OrderV1
				if err := ao.UnmarshalBinary(buf.Bytes()); assert.NoError(t, err) {
					assert.Equal(t, o.ID, ao.ID)
					assert.Equal(t, o.Signature, ao.Signature)
					assert.Equal(t, o.SenderPK, ao.SenderPK)
					assert.Equal(t, o.MatcherPK, ao.MatcherPK)
					assert.Equal(t, o.AssetPair, ao.AssetPair)
					assert.Equal(t, o.OrderType, ao.OrderType)
					assert.Equal(t, o.Price, ao.Price)
					assert.Equal(t, o.Amount, ao.Amount)
					assert.Equal(t, o.Timestamp, ao.Timestamp)
					assert.Equal(t, o.Expiration, ao.Expiration)
					assert.Equal(t, o.MatcherFee, ao.MatcherFee)
				}
			}
		}
	}
}

func BenchmarkOrderV1SigningRoundTrip(t *testing.B) {
	bts := make([]byte, 0, 1024*1024)
	buf := bytes.NewBuffer(bts)

	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10},
	}
	tc := tests[0]

	seed, _ := base58.Decode(tc.seed)
	sk, pk, _ := crypto.GenerateKeyPair(seed)
	mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
	aa, _ := NewOptionalAssetFromString(tc.amountAsset)
	pa, _ := NewOptionalAssetFromString(tc.priceAsset)
	ts := uint64(time.Now().UnixNano() / 1000000)
	exp := ts + 100*1000
	o := NewUnsignedOrderV1(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee)
	err := o.Sign(MainNetScheme, sk)
	require.NoError(t, err)

	t.Run("serialize", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			s := serializer.New(buf)
			b.StartTimer()
			for j := 0; j < 10; j++ {
				_ = o.Serialize(s)
			}
			b.StopTimer()
		}
	})
	t.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.StopTimer()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			for j := 0; j < 10; j++ {
				_, _ = o.MarshalBinary()
			}
			b.StopTimer()
		}
	})
}

func TestOrderV1ToJSON(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2},
	}
	for _, tc := range tests {
		seed, _ := base58.Decode(tc.seed)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		aas := "null"
		if aa.Present {
			aas = fmt.Sprintf("\"%s\"", aa.ID.String())
		}
		pas := "null"
		if pa.Present {
			pas = fmt.Sprintf("\"%s\"", pa.ID.String())
		}
		exp := ts + 100*1000
		o := NewUnsignedOrderV1(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee)
		if j, err := json.Marshal(o); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
				base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
			assert.Equal(t, ej, string(j))
			if err := o.Sign(MainNetScheme, sk); assert.NoError(t, err) {
				if j, err := json.Marshal(o); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
						base58.Encode(o.ID[:]), base58.Encode(o.Signature[:]), base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
					assert.Equal(t, ej, string(j))
				}
			}
		}
	}
}

func TestOrderV2Validations(t *testing.T) {
	aa, err := NewOptionalAssetFromString("8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS")
	require.NoError(t, err)
	pa, err := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	require.NoError(t, err)
	waves, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)
	tests := []struct {
		amountAsset OptionalAsset
		priceAsset  OptionalAsset
		orderType   OrderType
		price       uint64
		amount      uint64
		fee         uint64
		ts          uint64
		exp         uint64
		err         string
	}{
		{*aa, *aa, Buy, 1234, 5678, 90, 1, 1, "invalid asset pair"},
		{*aa, *pa, Sell, 0, 20, 30, 1, 1, "price should be positive"},
		{*aa, *pa, Buy, math.MaxInt64 + 1, 20, 30, 1, 1, "price is too big"},
		{*aa, *pa, Sell, 10, 0, 30, 1, 1, "amount should be positive"},
		{*aa, *pa, Buy, 10, math.MaxInt64 + 1, 30, 1, 1, "amount is too big"},
		{*aa, *pa, Sell, 10, MaxOrderAmount + 1, 30, 1, 1, "amount is larger than maximum allowed"},
		{*aa, *pa, Buy, 10, 20, 0, 1, 1, "matcher's fee should be positive"},
		{*aa, *pa, Sell, 10, 20, math.MaxInt64 + 2, 1, 1, "matcher's fee is too big"},
		{*aa, *pa, Sell, 10, 20, MaxOrderAmount + 1, 1, 1, "matcher's fee is larger than maximum allowed"},
		{*aa, *waves, Buy, math.MaxInt64, MaxOrderAmount, 1000, 1, 1, "spend amount is too large"},
		{*aa, *waves, Buy, 1, 1, 1000, 1, 1, "spend amount should be positive"},
		{*aa, *waves, Sell, math.MaxInt64, MaxOrderAmount, 1000, 1, 1, "receive amount is too large"},
		{*aa, *waves, Sell, 1, 1, 1000, 1, 1, "receive amount should be positive"},
		{*aa, *waves, Buy, math.MaxInt64 / (100 * PriceConstant), MaxOrderAmount, MaxOrderAmount, 1, 1, "sum of spend asset amount and matcher fee overflows JVM long"},
		{*aa, *pa, Sell, 100000000, 20, 30, 0, 1, "timestamp should be positive"},
		{*aa, *pa, Sell, 100000000, 20, 30, 1, 0, "expiration should be positive"},
	}
	spk, err := crypto.NewPublicKeyFromBase58("6s3F3S1ZmdJ2B25EqHWgNUSfeHtMaRZJ4RGEB5hgS7QM")
	require.NoError(t, err)
	mpk, err := crypto.NewPublicKeyFromBase58("7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy")
	require.NoError(t, err)
	for _, tc := range tests {
		o := NewUnsignedOrderV2(spk, mpk, tc.amountAsset, tc.priceAsset, tc.orderType, tc.price, tc.amount, tc.ts, tc.exp, tc.fee)
		v, err := o.Valid()
		assert.False(t, v)
		assert.EqualError(t, err, tc.err)
	}
}

func TestOrderV2BinarySize(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2},
	}
	for _, tc := range tests {
		seed, _ := base58.Decode(tc.seed)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		o := NewUnsignedOrderV2(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee)
		err = o.Sign(MainNetScheme, sk)
		assert.NoError(t, err)
		oBytes, err := o.MarshalBinary()
		assert.NoError(t, err)
		assert.Equal(t, len(oBytes), o.BinarySize())
	}
}

func TestOrderV2SigningRoundTrip(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2},
	}
	for _, tc := range tests {
		seed, _ := base58.Decode(tc.seed)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		o := NewUnsignedOrderV2(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee)
		if err := o.Sign(MainNetScheme, sk); assert.NoError(t, err) {
			if r, err := o.Verify(MainNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
			if b, err := o.MarshalBinary(); assert.NoError(t, err) {
				var ao OrderV2
				if err := ao.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *o.ID, *ao.ID)
					assert.ElementsMatch(t, o.Proofs.Proofs[0], ao.Proofs.Proofs[0])
					assert.ElementsMatch(t, o.SenderPK, ao.SenderPK)
					assert.ElementsMatch(t, o.MatcherPK, ao.MatcherPK)
					assert.Equal(t, o.AssetPair, ao.AssetPair)
					assert.Equal(t, o.OrderType, ao.OrderType)
					assert.Equal(t, o.Price, ao.Price)
					assert.Equal(t, o.Amount, ao.Amount)
					assert.Equal(t, o.Timestamp, ao.Timestamp)
					assert.Equal(t, o.Expiration, ao.Expiration)
					assert.Equal(t, o.MatcherFee, ao.MatcherFee)
				}
			}
		}
	}
}

func TestOrderV2ToJSON(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2},
	}
	for _, tc := range tests {
		seed, _ := base58.Decode(tc.seed)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		aas := "null"
		if aa.Present {
			aas = fmt.Sprintf("\"%s\"", aa.ID.String())
		}
		pas := "null"
		if pa.Present {
			pas = fmt.Sprintf("\"%s\"", pa.ID.String())
		}
		exp := ts + 100*1000
		o := NewUnsignedOrderV2(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee)
		if j, err := json.Marshal(o); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"version\":2,\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
				base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
			assert.Equal(t, ej, string(j))
			if err := o.Sign(MainNetScheme, sk); assert.NoError(t, err) {
				if j, err := json.Marshal(o); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
						base58.Encode(o.ID[:]), base58.Encode(o.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
					assert.Equal(t, ej, string(j))
				}
			}
		}
	}
}

func TestOrderV3Validations(t *testing.T) {
	aa, err := NewOptionalAssetFromString("8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS")
	require.NoError(t, err)
	pa, err := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	require.NoError(t, err)
	fa, err := NewOptionalAssetFromString("2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt")
	require.NoError(t, err)
	waves, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)
	tests := []struct {
		amountAsset OptionalAsset
		priceAsset  OptionalAsset
		orderType   OrderType
		price       uint64
		amount      uint64
		fee         uint64
		feeAsset    OptionalAsset
		ts          uint64
		exp         uint64
		err         string
	}{
		{*aa, *aa, Buy, 1234, 5678, 90, *fa, 1, 1, "invalid asset pair"},
		{*aa, *pa, Sell, 0, 20, 30, *fa, 1, 1, "price should be positive"},
		{*aa, *pa, Buy, math.MaxInt64 + 1, 20, 30, *fa, 1, 1, "price is too big"},
		{*aa, *pa, Sell, 10, 0, 30, *fa, 1, 1, "amount should be positive"},
		{*aa, *pa, Buy, 10, math.MaxInt64 + 1, 30, *fa, 1, 1, "amount is too big"},
		{*aa, *pa, Sell, 10, MaxOrderAmount + 1, 30, *fa, 1, 1, "amount is larger than maximum allowed"},
		{*aa, *pa, Buy, 10, 20, 0, *waves, 1, 1, "matcher's fee should be positive"},
		{*aa, *pa, Sell, 10, 20, math.MaxInt64 + 2, *fa, 1, 1, "matcher's fee is too big"},
		{*aa, *pa, Sell, 10, 20, MaxOrderAmount + 1, *fa, 1, 1, "matcher's fee is larger than maximum allowed"},
		{*aa, *waves, Buy, math.MaxInt64, MaxOrderAmount, 1000, *fa, 1, 1, "spend amount is too large"},
		{*aa, *waves, Buy, 1, 1, 1000, *fa, 1, 1, "spend amount should be positive"},
		{*aa, *waves, Sell, math.MaxInt64, MaxOrderAmount, 1000, *fa, 1, 1, "receive amount is too large"},
		{*aa, *waves, Sell, 1, 1, 1000, *fa, 1, 1, "receive amount should be positive"},
		{*aa, *waves, Buy, math.MaxInt64 / (100 * PriceConstant), MaxOrderAmount, MaxOrderAmount, *fa, 1, 1, "sum of spend asset amount and matcher fee overflows JVM long"},
		{*aa, *pa, Sell, 100000000, 20, 30, *waves, 0, 1, "timestamp should be positive"},
		{*aa, *pa, Sell, 100000000, 20, 30, *waves, 1, 0, "expiration should be positive"},
	}
	spk, err := crypto.NewPublicKeyFromBase58("6s3F3S1ZmdJ2B25EqHWgNUSfeHtMaRZJ4RGEB5hgS7QM")
	require.NoError(t, err)
	mpk, err := crypto.NewPublicKeyFromBase58("7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy")
	require.NoError(t, err)
	for _, tc := range tests {
		o := NewUnsignedOrderV3(spk, mpk, tc.amountAsset, tc.priceAsset, tc.orderType, tc.price, tc.amount, tc.ts, tc.exp, tc.fee, tc.feeAsset)
		v, err := o.Valid()
		assert.False(t, v)
		assert.EqualError(t, err, tc.err)
	}
}

func TestOrderV3BinarySize(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
		feeAsset    string
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10, "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck"},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1, "WAVES"},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2, "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck"},
	}
	for _, tc := range tests {
		seed, err := base58.Decode(tc.seed)
		require.NoError(t, err)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, err := crypto.NewPublicKeyFromBase58(tc.matcher)
		require.NoError(t, err)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		pa, err := NewOptionalAssetFromString(tc.priceAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)

		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		o := NewUnsignedOrderV3(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee, *fa)
		err = o.Sign(MainNetScheme, sk)
		assert.NoError(t, err)
		oBytes, err := o.MarshalBinary()
		assert.NoError(t, err)
		assert.Equal(t, len(oBytes), o.BinarySize())
	}
}

func TestOrderV3SigningRoundTrip(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
		feeAsset    string
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10, "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck"},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1, "WAVES"},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2, "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck"},
	}
	for _, tc := range tests {
		seed, err := base58.Decode(tc.seed)
		require.NoError(t, err)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, err := crypto.NewPublicKeyFromBase58(tc.matcher)
		require.NoError(t, err)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		pa, err := NewOptionalAssetFromString(tc.priceAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)

		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		o := NewUnsignedOrderV3(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee, *fa)
		if err := o.Sign(MainNetScheme, sk); assert.NoError(t, err) {
			if r, err := o.Verify(MainNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
			if b, err := o.MarshalBinary(); assert.NoError(t, err) {
				var ao OrderV3
				if err := ao.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *o.ID, *ao.ID)
					assert.ElementsMatch(t, o.Proofs.Proofs[0], ao.Proofs.Proofs[0])
					assert.ElementsMatch(t, o.SenderPK, ao.SenderPK)
					assert.ElementsMatch(t, o.MatcherPK, ao.MatcherPK)
					assert.Equal(t, o.AssetPair, ao.AssetPair)
					assert.Equal(t, o.OrderType, ao.OrderType)
					assert.Equal(t, o.Price, ao.Price)
					assert.Equal(t, o.Amount, ao.Amount)
					assert.Equal(t, o.Timestamp, ao.Timestamp)
					assert.Equal(t, o.Expiration, ao.Expiration)
					assert.Equal(t, o.MatcherFee, ao.MatcherFee)
				}
			}
		}
	}
}

func TestOrderV3ToJSON(t *testing.T) {
	tests := []struct {
		seed        string
		matcher     string
		amountAsset string
		priceAsset  string
		orderType   OrderType
		amount      uint64
		price       uint64
		fee         uint64
		feeAsset    string
	}{
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Sell, 1000, 100, 10, "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck"},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "WAVES", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", Buy, 1, 1, 1, "WAVES"},
		{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", "WAVES", Sell, 2, 2, 2, "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck"},
	}
	for _, tc := range tests {
		seed, err := base58.Decode(tc.seed)
		require.NoError(t, err)
		sk, pk, err := crypto.GenerateKeyPair(seed)
		assert.NoError(t, err)
		mpk, err := crypto.NewPublicKeyFromBase58(tc.matcher)
		require.NoError(t, err)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		pa, err := NewOptionalAssetFromString(tc.priceAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		ts := uint64(time.Now().UnixNano() / 1000000)
		aas := "null"
		if aa.Present {
			aas = fmt.Sprintf("\"%s\"", aa.ID.String())
		}
		pas := "null"
		if pa.Present {
			pas = fmt.Sprintf("\"%s\"", pa.ID.String())
		}
		fas := "null"
		if fa.Present {
			fas = fmt.Sprintf("\"%s\"", fa.ID.String())
		}
		exp := ts + 100*1000
		o := NewUnsignedOrderV3(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee, *fa)
		if j, err := json.Marshal(o); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"version\":3,\"matcherFeeAssetId\":%s,\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
				fas, base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
			assert.Equal(t, ej, string(j))
			if err := o.Sign(MainNetScheme, sk); assert.NoError(t, err) {
				if j, err := json.Marshal(o); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"version\":3,\"id\":\"%s\",\"proofs\":[\"%s\"],\"matcherFeeAssetId\":%s,\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
						base58.Encode(o.ID[:]), base58.Encode(o.Proofs.Proofs[0]), fas, base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
					assert.Equal(t, ej, string(j))
				}
			}
		}
	}
}

func TestIntegerDataEntryBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value int64
	}{
		{"some key", 12345},
		{"negative value", -9876543210},
		{"", 1234567890},
		{"", 0},
	}
	for _, tc := range tests {
		v := IntegerDataEntry{tc.key, tc.value}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av IntegerDataEntry
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.Equal(t, tc.value, av.Value)
				assert.Equal(t, DataInteger, av.GetValueType())
			}
		}
	}
}

func TestIntegerDataEntryJSONRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value int64
	}{
		{"some key", 12345},
		{"negative value", -9876543210},
		{"", 1234567890},
		{"", 0},
	}
	for _, tc := range tests {
		v := IntegerDataEntry{tc.key, tc.value}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			ejs := fmt.Sprintf("{\"key\":\"%s\",\"type\":\"integer\",\"value\":%d}", tc.key, tc.value)
			assert.Equal(t, ejs, js)
			var av IntegerDataEntry
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.Equal(t, tc.value, av.Value)
				assert.Equal(t, DataInteger, av.GetValueType())
			}
		}
	}
}

func TestBooleanDataEntryBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value bool
	}{
		{"some key", true},
		{"negative value", false},
		{"", true},
		{"", false},
	}
	for _, tc := range tests {
		v := BooleanDataEntry{tc.key, tc.value}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av BooleanDataEntry
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.Equal(t, tc.value, av.Value)
				assert.Equal(t, DataBoolean, av.GetValueType())
			}
		}
	}
}

func TestBooleanDataEntryJSONRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value bool
	}{
		{"some key", true},
		{"negative value", false},
		{"", true},
		{"", false},
	}
	for _, tc := range tests {
		v := BooleanDataEntry{tc.key, tc.value}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			ejs := fmt.Sprintf("{\"key\":\"%s\",\"type\":\"boolean\",\"value\":%v}", tc.key, tc.value)
			assert.Equal(t, ejs, js)
			var av BooleanDataEntry
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.Equal(t, tc.value, av.Value)
				assert.Equal(t, DataBoolean, av.GetValueType())
			}
		}
	}
}

func TestBinaryDataEntryBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"some key", "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"},
		{"empty value", "1"},
		{"", "1"},
	}
	for _, tc := range tests {
		bv, err := base58.Decode(tc.value)
		require.NoError(t, err)
		v := BinaryDataEntry{tc.key, bv}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av BinaryDataEntry
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.ElementsMatch(t, bv, av.Value)
				assert.Equal(t, DataBinary, av.GetValueType())
			}
		}
	}
}

func TestBinaryDataEntryJSONRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"some key", "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"},
		{"empty value", "1"},
		{"", "1"},
	}
	for _, tc := range tests {
		bv, err := base58.Decode(tc.value)
		require.NoError(t, err)
		v := BinaryDataEntry{tc.key, bv}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			s := fmt.Sprintf("\"base64:%s\"", base64.StdEncoding.EncodeToString(bv))
			ejs := fmt.Sprintf("{\"key\":\"%s\",\"type\":\"binary\",\"value\":%s}", tc.key, s)
			assert.Equal(t, ejs, js)
			var av BinaryDataEntry
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.ElementsMatch(t, bv, av.Value)
				assert.Equal(t, DataBinary, av.GetValueType())
			}
		}
	}
}

func TestStringDataEntryBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"some key", "some value string"},
		{"empty value", ""},
		{"", ""},
		{strings.Repeat("key-", 10), strings.Repeat("value-", 100)},
	}
	for _, tc := range tests {
		v := StringDataEntry{tc.key, tc.value}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av StringDataEntry
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.Equal(t, tc.value, av.Value)
				assert.Equal(t, DataString, av.GetValueType())
			}
		}
	}
}

func TestStringDataEntryJSONRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"some key", "some value string"},
		{"empty value", ""},
		{"", ""},
		{strings.Repeat("key-", 10), strings.Repeat("value-", 100)},
	}
	for _, tc := range tests {
		v := StringDataEntry{tc.key, tc.value}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			ejs := fmt.Sprintf("{\"key\":\"%s\",\"type\":\"string\",\"value\":\"%s\"}", tc.key, tc.value)
			assert.Equal(t, ejs, js)
			var av StringDataEntry
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.Equal(t, tc.value, av.Value)
				assert.Equal(t, DataString, av.GetValueType())
			}
		}
	}
}

func TestDataEntriesUnmarshalJSON(t *testing.T) {
	tests := []struct {
		json     string
		expected DataEntries
	}{
		{"[{\"key\":\"k1\",\"type\":\"integer\",\"value\":12345}]",
			DataEntries{&IntegerDataEntry{Key: "k1", Value: 12345}},
		},
		{"[{\"key\":\"k1\",\"type\":\"integer\",\"value\":12345},{\"key\":\"k2\",\"type\":\"boolean\",\"value\":true}]",
			DataEntries{&IntegerDataEntry{Key: "k1", Value: 12345}, &BooleanDataEntry{Key: "k2", Value: true}},
		},
		{"[{\"key\":\"k1\",\"type\":\"integer\",\"value\":12345},{\"key\":\"k2\",\"type\":\"boolean\",\"value\":true},{\"key\":\"k3\",\"type\":\"binary\",\"value\":\"base64:JH9xFB0dBYAX9BohYq06cMrtwta9mEoaj0aSVpLApyc=\"}]",
			DataEntries{&IntegerDataEntry{Key: "k1", Value: 12345}, &BooleanDataEntry{Key: "k2", Value: true}, &BinaryDataEntry{Key: "k3", Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}},
		},
		{"[{\"key\":\"k4\",\"type\":\"string\",\"value\":\"blah-blah\"}]",
			DataEntries{&StringDataEntry{Key: "k4", Value: "blah-blah"}},
		},
		{"[{\"key\":\"k1\",\"type\":\"integer\",\"value\":12345},{\"key\":\"k2\",\"type\":\"boolean\",\"value\":true},{\"key\":\"k3\",\"type\":\"binary\",\"value\":\"base64:JH9xFB0dBYAX9BohYq06cMrtwta9mEoaj0aSVpLApyc=\"},{\"key\":\"k4\",\"type\":\"string\",\"value\":\"blah-blah\"}]",
			DataEntries{&IntegerDataEntry{Key: "k1", Value: 12345}, &BooleanDataEntry{Key: "k2", Value: true}, &BinaryDataEntry{Key: "k3", Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}, &StringDataEntry{Key: "k4", Value: "blah-blah"}},
		},
	}
	for _, tc := range tests {
		entries := DataEntries{}
		if err := entries.UnmarshalJSON([]byte(tc.json)); assert.NoError(t, err) {
			if b, err := json.Marshal(entries); assert.NoError(t, err) {
				assert.Equal(t, tc.json, string(b))
			}
			assert.ElementsMatch(t, tc.expected, entries)
		}
	}
}

func TestNewLegacyAttachmentFromBase58(t *testing.T) {
	att, err := NewLegacyAttachmentFromBase58("t")
	require.NoError(t, err)
	assert.Equal(t, *att, LegacyAttachment{Value: []byte("3")})
}

func TestAttachment_UnmarshalJSON(t *testing.T) {
	a := LegacyAttachment{Value: []byte{}}
	err := a.UnmarshalJSON([]byte("null"))
	require.NoError(t, err)
	assert.Equal(t, "", a.String())

	err = a.UnmarshalJSON([]byte(`"8Gbmq3u18PmPbWcobY"`))
	require.NoError(t, err)
	assert.Equal(t, "WELCOME BONUS", a.String())

	err = a.UnmarshalJSON([]byte(`""`))
	require.NoError(t, err)
	assert.Equal(t, "", a.String())
}

func TestNewOptionalAssetFromBytes(t *testing.T) {
	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)

	asset1, err := NewOptionalAssetFromBytes(d.Bytes())
	require.NoError(t, err)
	assert.Equal(t, d.String(), asset1.ID.String())
	assert.True(t, asset1.Present)

	asset2, err := NewOptionalAssetFromBytes([]byte{})
	require.NoError(t, err)
	assert.False(t, asset2.Present)
}

func TestNewOptionalAssetFromDigest(t *testing.T) {
	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)

	asset1 := NewOptionalAssetFromDigest(d)
	assert.True(t, asset1.Present)
}

func TestOptionalAsset_Marshal(t *testing.T) {
	d, _ := NewOptionalAssetFromString("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")

	b, _ := d.MarshalBinary()
	d2 := OptionalAsset{}
	_ = d2.UnmarshalBinary(b)

	require.Equal(t, d.String(), d2.String())

	buf := new(bytes.Buffer)
	_, _ = d.WriteTo(buf)
}

func TestOptionalAsset_WriteTo(t *testing.T) {
	d, _ := NewOptionalAssetFromString("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")

	buf := new(bytes.Buffer)
	_, _ = d.WriteTo(buf)

	d2 := OptionalAsset{}
	_ = d2.UnmarshalBinary(buf.Bytes())

	require.Equal(t, d.String(), d2.String())
}

func TestScriptJSONRoundTrip(t *testing.T) {
	tests := []struct {
		json string
	}{
		{"\"base64:\""},
		{"\"base64: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\""},
	}
	for _, tc := range tests {
		var s Script
		if err := json.Unmarshal([]byte(tc.json), &s); assert.NoError(t, err) {
			if js, err := json.Marshal(s); assert.NoError(t, err) {
				assert.Equal(t, tc.json, string(js))
			}
		}
	}
}

func TestProofsV1UnmarshalJSON(t *testing.T) {
	tests := []struct {
		json     string
		expected []B58Bytes
	}{
		{"[]",
			[]B58Bytes{},
		},
		{"[\"\"]",
			[]B58Bytes{{}},
		},
		{"[\"\", \"\"]",
			[]B58Bytes{{}, {}},
		},
		{
			"[\"2PPJhhHz2sCoFtJ3Dx3fTDzRXEC6Zm76kQMF7m5aqeL1XjFRysMircz7Cy5zJc77BrYTbEG9pgY4MMRVjv1S1hGx\"]",
			[]B58Bytes{{0x45, 0x52, 0x13, 0x19, 0xaf, 0x82, 0x3e, 0x01, 0x38, 0xf9, 0x99, 0x7a, 0x3a, 0xd0, 0x7f, 0xa3, 0x81, 0xde, 0xce, 0x6b, 0x4d, 0xe4, 0x0c, 0x81, 0x78, 0x4b, 0xd7, 0x15, 0xd4, 0x34, 0x08, 0x22, 0x8c, 0x04, 0xdf, 0x89, 0x7a, 0x7f, 0x95, 0x66, 0xd5, 0x75, 0xc2, 0x0a, 0xbb, 0x97, 0x64, 0x29, 0xe3, 0x48, 0x67, 0xe8, 0x22, 0xeb, 0x6f, 0x93, 0xbb, 0xd8, 0x22, 0xac, 0x11, 0x3c, 0xa8, 0x0d}},
		},
	}
	for _, tc := range tests {
		var p ProofsV1
		if err := json.Unmarshal([]byte(tc.json), &p); assert.NoError(t, err) {
			assert.Equal(t, 1, int(p.Version))
			assert.Equal(t, len(tc.expected), len(p.Proofs))
			assert.ElementsMatch(t, tc.expected, p.Proofs)
		}
	}
}

func TestIntegerArgumentBinarySize(t *testing.T) {
	tests := []int64{12345, -9876543210, 1234567890, 0}
	for _, tc := range tests {
		v := IntegerArgument{tc}
		assert.Equal(t, 9, v.BinarySize())
	}
}

func TestBooleanArgumentBinarySize(t *testing.T) {
	tests := []bool{true, false}
	for _, tc := range tests {
		v := BooleanArgument{tc}
		assert.Equal(t, 1, v.BinarySize())
	}
}

func TestBinaryArgumentBinarySize(t *testing.T) {
	tests := []string{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "", "111111111111111"}
	for _, tc := range tests {
		bv, _ := base58.Decode(tc)
		v := BinaryArgument{bv}
		assert.Equal(t, 1+4+len(bv), v.BinarySize())
	}
}

func TestStringArgumentBinarySize(t *testing.T) {
	tests := []string{"some value string", "", strings.Repeat("value-", 100)}
	for _, tc := range tests {
		v := StringArgument{tc}
		assert.Equal(t, 1+4+len(tc), v.BinarySize())
	}

}

func TestIntegerArgumentBinaryRoundTrip(t *testing.T) {
	tests := []int64{12345, -9876543210, 1234567890, 0}
	for _, tc := range tests {
		v := IntegerArgument{tc}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av IntegerArgument
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc, av.Value)
				assert.Equal(t, ArgumentInteger, av.GetValueType())
			}
		}
	}
}

func TestIntegerArgumentJSONRoundTrip(t *testing.T) {
	tests := []int64{12345, -9876543210, 1234567890, 0}
	for _, tc := range tests {
		v := IntegerArgument{tc}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			ejs := fmt.Sprintf("{\"type\":\"integer\",\"value\":%d}", tc)
			assert.Equal(t, ejs, js)
			var av IntegerArgument
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.Equal(t, tc, av.Value)
				assert.Equal(t, ArgumentInteger, av.GetValueType())
			}
		}
	}
}

func TestBooleanArgumentBinaryRoundTrip(t *testing.T) {
	tests := []bool{true, false}
	for _, tc := range tests {
		v := BooleanArgument{tc}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av BooleanArgument
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc, av.Value)
				assert.Equal(t, ArgumentBoolean, av.GetValueType())
			}
		}
	}
}

func TestBooleanArgumentJSONRoundTrip(t *testing.T) {
	tests := []bool{true, false}
	for _, tc := range tests {
		v := BooleanArgument{tc}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			ejs := fmt.Sprintf("{\"type\":\"boolean\",\"value\":%v}", tc)
			assert.Equal(t, ejs, js)
			var av BooleanArgument
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.Equal(t, tc, av.Value)
				assert.Equal(t, ArgumentBoolean, av.GetValueType())
			}
		}
	}
}

func TestBinaryArgumentBinaryRoundTrip(t *testing.T) {
	tests := []string{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "", "111111111111111"}
	for _, tc := range tests {
		bv, _ := base58.Decode(tc)
		v := BinaryArgument{bv}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av BinaryArgument
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.ElementsMatch(t, bv, av.Value)
				assert.Equal(t, ArgumentBinary, av.GetValueType())
			}
		}
	}
}

func TestBinaryArgumentJSONRoundTrip(t *testing.T) {
	tests := []string{"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc", "1", "111111111111111"}
	for _, tc := range tests {
		bv, err := base58.Decode(tc)
		require.NoError(t, err)
		v := BinaryArgument{bv}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			s := fmt.Sprintf("\"base64:%s\"", base64.StdEncoding.EncodeToString(bv))
			ejs := fmt.Sprintf("{\"type\":\"binary\",\"value\":%s}", s)
			assert.Equal(t, ejs, js)
			var av BinaryArgument
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.ElementsMatch(t, bv, av.Value)
				assert.Equal(t, ArgumentBinary, av.GetValueType())
			}
		}
	}
}

func TestStringArgumentBinaryRoundTrip(t *testing.T) {
	tests := []string{"some value string", "", strings.Repeat("value-", 100)}
	for _, tc := range tests {
		v := StringArgument{tc}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av StringArgument
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc, av.Value)
				assert.Equal(t, ArgumentString, av.GetValueType())
			}
		}
	}
}

func TestStringArgumentJSONRoundTrip(t *testing.T) {
	tests := []string{"some value string", "", strings.Repeat("value-", 100)}
	for _, tc := range tests {
		v := StringArgument{tc}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			ejs := fmt.Sprintf("{\"type\":\"string\",\"value\":\"%s\"}", tc)
			assert.Equal(t, ejs, js)
			var av StringArgument
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.Equal(t, tc, av.Value)
				assert.Equal(t, ArgumentString, av.GetValueType())
			}
		}
	}
}

func TestArgumentsJSONRoundTrip(t *testing.T) {
	tests := []struct {
		js   string
		args Arguments
	}{
		{"[{\"type\":\"integer\",\"value\":12345}]",
			Arguments{&IntegerArgument{Value: 12345}},
		},
		{"[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true}]",
			Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}},
		},
		{"[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true},{\"type\":\"binary\",\"value\":\"base64:JH9xFB0dBYAX9BohYq06cMrtwta9mEoaj0aSVpLApyc=\"}]",
			Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}},
		},
		{"[{\"type\":\"string\",\"value\":\"blah-blah\"}]",
			Arguments{&StringArgument{Value: "blah-blah"}},
		},
		{"[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true},{\"type\":\"binary\",\"value\":\"base64:JH9xFB0dBYAX9BohYq06cMrtwta9mEoaj0aSVpLApyc=\"},{\"type\":\"string\",\"value\":\"blah-blah\"}]",
			Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}, &StringArgument{Value: "blah-blah"}},
		},
	}
	for _, tc := range tests {
		if b, err := json.Marshal(tc.args); assert.NoError(t, err) {
			assert.Equal(t, tc.js, string(b))
			args := Arguments{}
			if err := json.Unmarshal(b, &args); assert.NoError(t, err) {
				assert.ElementsMatch(t, tc.args, args)
			}
		}
	}
}

func TestArgumentsBinaryRoundTrip(t *testing.T) {
	tests := []Arguments{
		{&IntegerArgument{Value: 12345}},
		{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}},
		{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}},
		{&StringArgument{Value: "blah-blah"}},
		{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}, &StringArgument{Value: "blah-blah"}},
	}
	for _, tc := range tests {
		if b, err := tc.MarshalBinary(); assert.NoError(t, err) {
			args := Arguments{}
			if err := args.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.ElementsMatch(t, tc, args)
			}
		}
	}
}

func TestScriptPaymentBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
	}{
		{"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD", 12345},
		{"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD", 98765},
		{"WAVES", 9876543210},
		{"", 0},
		{"", 1234567890},
	}
	for _, tc := range tests {
		a, err := NewOptionalAssetFromString(tc.asset)
		require.NoError(t, err)
		sp := ScriptPayment{Asset: *a, Amount: tc.amount}
		if b, err := sp.MarshalBinary(); assert.NoError(t, err) {
			var av ScriptPayment
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, sp, av)
			}
		}
	}
}

func TestScriptPaymentJSONRoundTrip(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		js     string
	}{
		{"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD", 12345, "{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}"},
		{"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD", 98765, "{\"amount\":98765,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}"},
		{"WAVES", 9876543210, "{\"amount\":9876543210,\"assetId\":null}"},
		{"", 0, "{\"amount\":0,\"assetId\":null}"},
		{"", 1234567890, "{\"amount\":1234567890,\"assetId\":null}"},
	}
	for _, tc := range tests {
		a, err := NewOptionalAssetFromString(tc.asset)
		require.NoError(t, err)
		sp := ScriptPayment{Asset: *a, Amount: tc.amount}
		if b, err := json.Marshal(sp); assert.NoError(t, err) {
			assert.Equal(t, tc.js, string(b))
			var av ScriptPayment
			if err := json.Unmarshal(b, &av); assert.NoError(t, err) {
				assert.Equal(t, sp, av)
			}
		}
	}
}

func TestScriptPaymentsBinaryRoundTrip(t *testing.T) {
	a1, err := NewOptionalAssetFromString("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)
	a2, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)
	tests := []ScriptPayments{
		{{Asset: *a1, Amount: 12345}},
		{{Asset: *a2, Amount: 67890}},
		{{Asset: *a1, Amount: 12345}, {Asset: *a2, Amount: 67890}},
		{{Asset: *a2, Amount: 67890}, {Asset: *a1, Amount: 12345}},
		{{Asset: *a1, Amount: 0}, {Asset: *a1, Amount: 67890}},
		{{Asset: *a2, Amount: 0}, {Asset: *a2, Amount: 12345}},
	}
	for _, tc := range tests {
		if b, err := tc.MarshalBinary(); assert.NoError(t, err) {
			av := ScriptPayments{}
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc, av)
			}
		}
	}
}

func TestScriptsPaymentsJSONRoundTrip(t *testing.T) {
	a1, err := NewOptionalAssetFromString("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)
	a2, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)
	tests := []struct {
		payments ScriptPayments
		js       string
	}{
		{payments: ScriptPayments{{Asset: *a1, Amount: 12345}}, js: "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]"},
		{payments: ScriptPayments{{Asset: *a2, Amount: 67890}}, js: "[{\"amount\":67890,\"assetId\":null}]"},
		{payments: ScriptPayments{{Asset: *a1, Amount: 12345}, {Asset: *a2, Amount: 67890}}, js: "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"},{\"amount\":67890,\"assetId\":null}]"},
		{payments: ScriptPayments{{Asset: *a2, Amount: 67890}, {Asset: *a1, Amount: 12345}}, js: "[{\"amount\":67890,\"assetId\":null},{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]"},
		{payments: ScriptPayments{{Asset: *a1, Amount: 0}, {Asset: *a1, Amount: 67890}}, js: "[{\"amount\":0,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"},{\"amount\":67890,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]"},
		{payments: ScriptPayments{{Asset: *a2, Amount: 0}, {Asset: *a2, Amount: 12345}}, js: "[{\"amount\":0,\"assetId\":null},{\"amount\":12345,\"assetId\":null}]"},
	}
	for _, tc := range tests {
		if b, err := json.Marshal(tc.payments); assert.NoError(t, err) {
			assert.Equal(t, string(b), tc.js)
			av := ScriptPayments{}
			if err := json.Unmarshal(b, &av); assert.NoError(t, err) {
				assert.Equal(t, tc.payments, av)
			}
		}
	}
}

func TestFunctionCallBinaryRoundTrip(t *testing.T) {
	tests := []FunctionCall{
		{Name: "foo", Arguments: Arguments{&IntegerArgument{Value: 12345}}},
		{Name: "bar", Arguments: Arguments{&BooleanArgument{Value: true}}},
		{Name: "baz", Arguments: Arguments{&BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}}},
		{Name: "foobar0", Arguments: Arguments{&StringArgument{Value: "blah-blah"}}},
		{Name: "foobar1", Arguments: Arguments{}},
		{Name: "foobar2", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}}},
		{Name: "foobar3", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}}},
		{Name: "foobar4", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}, &StringArgument{Value: "blah-blah"}}},
	}
	for _, tc := range tests {
		if b, err := tc.MarshalBinary(); assert.NoError(t, err) {
			fc := FunctionCall{}
			if err := fc.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc, fc)
			}
		}
	}
}

func TestFunctionCallJSONRoundTrip(t *testing.T) {
	tests := []struct {
		fc FunctionCall
		js string
	}{
		{fc: FunctionCall{Name: "foo", Arguments: Arguments{&IntegerArgument{Value: 12345}}}, js: "{\"function\":\"foo\",\"args\":[{\"type\":\"integer\",\"value\":12345}]}"},
		{fc: FunctionCall{Name: "bar", Arguments: Arguments{&BooleanArgument{Value: true}}}, js: "{\"function\":\"bar\",\"args\":[{\"type\":\"boolean\",\"value\":true}]}"},
		{fc: FunctionCall{Name: "baz", Arguments: Arguments{&BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}}}, js: "{\"function\":\"baz\",\"args\":[{\"type\":\"binary\",\"value\":\"base64:JH9xFB0dBYAX9BohYq06cMrtwta9mEoaj0aSVpLApyc=\"}]}"},
		{fc: FunctionCall{Name: "foobar0", Arguments: Arguments{&StringArgument{Value: "blah-blah"}}}, js: "{\"function\":\"foobar0\",\"args\":[{\"type\":\"string\",\"value\":\"blah-blah\"}]}"},
		{fc: FunctionCall{Name: "foobar1", Arguments: Arguments{}}, js: "{\"function\":\"foobar1\",\"args\":[]}"},
		{fc: FunctionCall{Name: "foobar2", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}}}, js: "{\"function\":\"foobar2\",\"args\":[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true}]}"},
		{fc: FunctionCall{Name: "foobar3", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}}}, js: "{\"function\":\"foobar3\",\"args\":[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true},{\"type\":\"binary\",\"value\":\"base64:JH9xFB0dBYAX9BohYq06cMrtwta9mEoaj0aSVpLApyc=\"}]}"},
		{fc: FunctionCall{Name: "foobar4", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}, &StringArgument{Value: "blah-blah"}}}, js: "{\"function\":\"foobar4\",\"args\":[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true},{\"type\":\"binary\",\"value\":\"base64:JH9xFB0dBYAX9BohYq06cMrtwta9mEoaj0aSVpLApyc=\"},{\"type\":\"string\",\"value\":\"blah-blah\"}]}"},
	}
	for _, tc := range tests {
		if b, err := json.Marshal(tc.fc); assert.NoError(t, err) {
			assert.Equal(t, tc.js, string(b))
			fc := FunctionCall{}
			if err := json.Unmarshal(b, &fc); assert.NoError(t, err) {
				assert.Equal(t, tc.fc, fc)
			}
		}
	}
}

func TestScriptResultBinaryRoundTrip(t *testing.T) {
	waves, err := NewOptionalAssetFromString("WAVES")
	assert.NoError(t, err)
	asset0, err := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	assert.NoError(t, err)
	asset1, err := NewOptionalAssetFromString("Ft5X1v1LTa1ABafufpaCWyVj7KkaxUWE6xBhW6sNFJck")
	assert.NoError(t, err)
	addr0, err := NewAddressFromString("3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v")
	assert.NoError(t, err)
	rcp := NewRecipientFromAddress(addr0)
	tests := []ScriptResult{
		{
			Writes: []DataEntry{
				&IntegerDataEntry{"some key", 12345},
				&BooleanDataEntry{"negative value", false},
				&StringDataEntry{"some key", "some value string"},
				&BinaryDataEntry{Key: "k3", Value: []byte{0x24, 0x7f, 0x71, 0x14, 0x1d}},
				&IntegerDataEntry{"some key2", -12345},
				&BooleanDataEntry{"negative value2", true},
				&StringDataEntry{"some key143", "some value2 string"},
				&BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}},
			},
			Transfers: []ScriptResultTransfer{
				{Amount: math.MaxInt64, Asset: *waves, Recipient: rcp},
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: -10, Asset: *asset0, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
		},
		{
			Writes: []DataEntry{
				&IntegerDataEntry{"some key", 12345},
			},
		},
		{
			Transfers: []ScriptResultTransfer{
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: -10, Asset: *asset0, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
		},
	}
	for _, tc := range tests {
		if b, err := tc.MarshalWithAddresses(); assert.NoError(t, err) {
			sr := ScriptResult{}
			if err := sr.UnmarshalWithAddresses(b); assert.NoError(t, err) {
				assert.Equal(t, tc, sr)
			}
		}
	}
	// Should not work with alias recipients.
	alias, err := NewAliasFromString("alias:T:blah-blah-blah")
	assert.NoError(t, err)
	sr := tests[0]
	sr.Transfers[0].Recipient = NewRecipientFromAlias(*alias)
	_, err = sr.MarshalWithAddresses()
	assert.Error(t, err)
}
