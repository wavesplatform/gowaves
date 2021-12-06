package proto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
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
	//waves, err := NewOptionalAssetFromString("WAVES")
	//require.NoError(t, err)
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
		//TODO: move those validations to exchange transaction tests
		//{*aa, *waves, Buy, math.MaxInt64, MaxOrderAmount, 1000, 1, 1, "spend amount is too large"},
		//{*aa, *waves, Buy, 1, 1, 1000, 1, 1, "spend amount should be positive"},
		//{*aa, *waves, Sell, math.MaxInt64, MaxOrderAmount, 1000, 1, 1, "receive amount is too large"},
		//{*aa, *waves, Sell, 1, 1, 1000, 1, 1, "receive amount should be positive"},
		//{*aa, *waves, Buy, math.MaxInt64 / (100 * PriceConstant), MaxOrderAmount, MaxOrderAmount, 1, 1, "sum of spend asset amount and matcher fee overflows JVM long"},
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
			if r, err := o.Verify(MainNetScheme); assert.NoError(t, err) {
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
	//waves, err := NewOptionalAssetFromString("WAVES")
	//require.NoError(t, err)
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
		//TODO: move validations to exchange transaction diff tests
		//{*aa, *waves, Buy, math.MaxInt64, MaxOrderAmount, 1000, 1, 1, "spend amount is too large"},
		//{*aa, *waves, Buy, 1, 1, 1000, 1, 1, "spend amount should be positive"},
		//{*aa, *waves, Sell, math.MaxInt64, MaxOrderAmount, 1000, 1, 1, "receive amount is too large"},
		//{*aa, *waves, Sell, 1, 1, 1000, 1, 1, "receive amount should be positive"},
		//{*aa, *waves, Buy, math.MaxInt64 / (100 * PriceConstant), MaxOrderAmount, MaxOrderAmount, 1, 1, "sum of spend asset amount and matcher fee overflows JVM long"},
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
			if r, err := o.Verify(MainNetScheme); assert.NoError(t, err) {
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
		//TODO: move validations to exchange transaction tests
		//{*aa, *waves, Buy, math.MaxInt64, MaxOrderAmount, 1000, *fa, 1, 1, "spend amount is too large"},
		//{*aa, *waves, Buy, 1, 1, 1000, *fa, 1, 1, "spend amount should be positive"},
		//{*aa, *waves, Sell, math.MaxInt64, MaxOrderAmount, 1000, *fa, 1, 1, "receive amount is too large"},
		//{*aa, *waves, Sell, 1, 1, 1000, *fa, 1, 1, "receive amount should be positive"},
		//{*aa, *waves, Buy, math.MaxInt64 / (100 * PriceConstant), MaxOrderAmount, MaxOrderAmount, *fa, 1, 1, "sum of spend asset amount and matcher fee overflows JVM long"},
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
			if r, err := o.Verify(MainNetScheme); assert.NoError(t, err) {
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

func TestDeleteDataEntryBinaryRoundTrip(t *testing.T) {
	for _, test := range []string{
		"some key",
		"empty value",
		"",
		strings.Repeat("key-", 10),
	} {
		v := DeleteDataEntry{test}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			assert.Equal(t, byte(0xff), b[len(b)-1])
			var av DeleteDataEntry
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, test, av.Key)
				assert.Equal(t, test, av.GetKey())
				assert.Equal(t, DataDelete, av.GetValueType())
			}
		}
	}
}

func TestDeleteDataEntryJSONRoundTrip(t *testing.T) {
	for _, test := range []string{
		"some key",
		"empty value",
		"",
		strings.Repeat("key-", 10),
	} {
		v := DeleteDataEntry{test}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			ejs := fmt.Sprintf("{\"key\":\"%s\",\"value\":null}", test)
			assert.Equal(t, ejs, js)
			var av DeleteDataEntry
			if err := av.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.Equal(t, test, av.Key)
				assert.Equal(t, test, av.GetKey())
				assert.Equal(t, DataDelete, av.GetValueType())
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
		{"[{\"key\":\"k1\",\"value\":null}]",
			DataEntries{&DeleteDataEntry{Key: "k1"}},
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

func TestAttachment_UnmarshalJSON(t *testing.T) {
	a := Attachment{}
	err := a.UnmarshalJSON([]byte("null"))
	require.NoError(t, err)
	assert.Equal(t, "", string(a))

	err = a.UnmarshalJSON([]byte(`"8Gbmq3u18PmPbWcobY"`))
	require.NoError(t, err)
	assert.Equal(t, "WELCOME BONUS", string(a))

	err = a.UnmarshalJSON([]byte(`""`))
	require.NoError(t, err)
	assert.Equal(t, "", string(a))
}

func TestNewOptionalAssetFromBytes(t *testing.T) {
	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)

	asset1, err := NewOptionalAssetFromBytes(d.Bytes())
	require.NoError(t, err)
	assert.Equal(t, d.String(), asset1.ID.String())
	assert.True(t, asset1.Present)

	_, err = NewOptionalAssetFromBytes([]byte{})
	require.Error(t, err)
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

func TestProofsV1_Valid(t *testing.T) {
	smallProof := make([]byte, 32)
	normProof := make([]byte, 64)
	bigProof := make([]byte, 65)
	_, err := rand.Read(smallProof)
	require.NoError(t, err)
	_, err = rand.Read(normProof)
	require.NoError(t, err)
	_, err = rand.Read(bigProof)
	require.NoError(t, err)
	p1 := NewProofs()
	p1.Proofs = append(p1.Proofs, smallProof)
	p1.Proofs = append(p1.Proofs, normProof)
	p1.Proofs = append(p1.Proofs, bigProof)
	err = p1.Valid()
	assert.Error(t, err)

	p2 := NewProofs()
	for i := 0; i < 9; i++ {
		p2.Proofs = append(p2.Proofs, smallProof)
	}
	err = p2.Valid()
	assert.Error(t, err)
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

func TestArrayArgumentBinarySize(t *testing.T) {
	for _, test := range []struct {
		args []Argument
		size int
	}{
		{nil, 1 + 4},
		{[]Argument{&IntegerArgument{12345}, &StringArgument{"12345"}, &BooleanArgument{true}, &BooleanArgument{false}}, 1 + 4 + 9 + 1 + 4 + 5 + 1 + 1},
		{[]Argument{&IntegerArgument{12345}, &StringArgument{"12345"}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{0, 1, 2, 3, 4, 5}}}, 1 + 4 + 9 + 1 + 4 + 5 + 1 + 1 + 1 + 4 + 6},
		{[]Argument{&IntegerArgument{12345}, &StringArgument{"12345"}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{0, 1, 2, 3, 4, 5}}, &ListArgument{Items: []Argument{&IntegerArgument{12345}, &StringArgument{"12345"}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{0, 1, 2, 3, 4, 5}}}}}, 1 + 4 + 9 + 1 + 4 + 5 + 1 + 1 + 1 + 4 + 6 + 1 + 4 + 9 + 1 + 4 + 5 + 1 + 1 + 1 + 4 + 6},
	} {
		v := ListArgument{Items: test.args}
		assert.Equal(t, test.size, v.BinarySize())
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

func TestArrayArgumentBinaryRoundTrip(t *testing.T) {
	for _, test := range []Arguments{
		nil,
		{&IntegerArgument{12345}, &StringArgument{"12345"}, &BooleanArgument{true}, &BooleanArgument{false}},
		{&IntegerArgument{0}, &StringArgument{""}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{}}},
		{&IntegerArgument{math.MaxInt32}, &StringArgument{strings.Repeat("12345", 100)}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{0, 1, 2, 3, 4, 5}}, &ListArgument{Items: []Argument{&IntegerArgument{12345}, &StringArgument{"12345"}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{0, 1, 2, 3, 4, 5}}}}},
	} {
		v := ListArgument{Items: test}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var aa ListArgument
			if err := aa.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.NotNil(t, aa)
				assert.Equal(t, test, aa.Items)
				assert.Equal(t, ArgumentList, aa.GetValueType())
			}
		}
	}
}

func TestArrayArgumentJSONRoundTrip(t *testing.T) {
	for _, test := range []Arguments{
		nil,
		{&IntegerArgument{12345}, &StringArgument{"12345"}, &BooleanArgument{true}, &BooleanArgument{false}},
		{&IntegerArgument{0}, &StringArgument{""}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{}}},
		{&IntegerArgument{math.MaxInt32}, &StringArgument{strings.Repeat("12345", 100)}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{0, 1, 2, 3, 4, 5}}, &ListArgument{Items: []Argument{&IntegerArgument{12345}, &StringArgument{"12345"}, &BooleanArgument{true}, &BooleanArgument{false}, &BinaryArgument{[]byte{0, 1, 2, 3, 4, 5}}}}},
	} {
		v := ListArgument{Items: test}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			var aa ListArgument
			if err := aa.UnmarshalJSON(b); assert.NoError(t, err) {
				assert.NotNil(t, aa)
				assert.Equal(t, test, aa.Items)
				assert.Equal(t, ArgumentList, aa.GetValueType())
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
		{fc: FunctionCall{Name: "foobar5", Arguments: Arguments{&ListArgument{Items: []Argument{&StringArgument{Value: "bar"}, &StringArgument{Value: "baz"}}}}}, js: "{\"function\":\"foobar5\",\"args\":[{\"type\":\"list\",\"value\":[{\"type\":\"string\",\"value\":\"bar\"},{\"type\":\"string\",\"value\":\"baz\"}]}]}"},
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

func createStateHash() StateHash {
	return StateHash{
		BlockID: NewBlockIDFromSignature(crypto.MustSignatureFromBase58("2UwZrKyjx7Bs4RYkEk5SLCdtr9w6GR1EDbpS3TH9DGJKcxSCuQP4nivk4YPFpQTqWmoXXPPUiy6riF3JwhikbSQu")),
		SumHash: crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
		FieldsHashes: FieldsHashes{
			DataEntryHash:     crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			AccountScriptHash: crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			AssetScriptHash:   crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			LeaseStatusHash:   crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			SponsorshipHash:   crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			AliasesHash:       crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			WavesBalanceHash:  crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			AssetBalanceHash:  crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ2RLn55UPzUDVgnh64EwYAU5iCj6z6"),
			LeaseBalanceHash:  crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"),
		},
	}
}

func TestStateHashJSONRoundTrip(t *testing.T) {
	sh := createStateHash()
	shJs, err := sh.MarshalJSON()
	assert.NoError(t, err)
	var sh2 StateHash
	err = sh2.UnmarshalJSON(shJs)
	assert.NoError(t, err)
	assert.Equal(t, sh, sh2)
}

func TestStateHashBinaryRoundTrip(t *testing.T) {
	sh := createStateHash()
	shBytes := sh.MarshalBinary()
	var sh2 StateHash
	err := sh2.UnmarshalBinary(shBytes)
	assert.NoError(t, err)
	assert.Equal(t, sh, sh2)
}

func TestStateHash_GenerateSumHash(t *testing.T) {
	sh := createStateHash()
	prevHash := crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	correctSumHash := crypto.MustDigestFromBase58("9ckTqHUsRap8YerHv1EijZMeBRaSFibdTkPqjmK9hoNy")
	err := sh.GenerateSumHash(prevHash[:])
	assert.NoError(t, err)
	assert.Equal(t, correctSumHash, sh.SumHash)
}

func TestEthereumOrderV4(t *testing.T) {
	ethStub32 := bytes.Repeat([]byte{CustomNetScheme}, 32)

	stubAssetID, err := crypto.NewDigestFromBytes(ethStub32)
	require.NoError(t, err)

	wavesPKStub, err := crypto.NewPublicKeyFromBytes(ethStub32)
	require.NoError(t, err)

	ethereumPKBytesStub, err := DecodeFromHexString("0xd10a150ba9a535125481e017a09c2ac6a1ab43fc43f7ab8f0d44635106672dd7de4f775c06b730483862cbc4371a646d86df77b3815593a846b7272ace008c42")
	require.NoError(t, err)

	ethereumSignatureBytesStub, err := DecodeFromHexString("0x54119bc5b24d9363b7a1a31a71a2e6194dfeedc5e9644893b0a04bb57004e5b14342c1ce29ee00877da49180fd6d7fb332ff400231f809da7ed0dcb07c504e2d1c")
	require.NoError(t, err)

	t.Run("Verify-Scala", func(t *testing.T) {
		// taken from scala-node

		testAssetPair := AssetPair{
			AmountAsset: OptionalAsset{
				Present: true,
				ID:      stubAssetID,
			},
			PriceAsset: OptionalAsset{
				Present: true,
				ID:      stubAssetID,
			},
		}

		testEthPubKeyHex := "0xd10a150ba9a535125481e017a09c2ac6a1ab43fc43f7ab8f0d44635106672dd7de4f775c06b730483862cbc4371a646d86df77b3815593a846b7272ace008c42"
		testEthSenderPK, err := NewEthereumPublicKeyFromHexString(testEthPubKeyHex)
		require.NoError(t, err)

		testEthSigHex := "0x54119bc5b24d9363b7a1a31a71a2e6194dfeedc5e9644893b0a04bb57004e5b14342c1ce29ee00877da49180fd6d7fb332ff400231f809da7ed0dcb07c504e2d1c"
		testEthSig, err := NewEthereumSignatureFromHexString(testEthSigHex)
		require.NoError(t, err)

		ethOrder := EthereumOrderV4{
			SenderPK:        testEthSenderPK,
			Eip712Signature: testEthSig,
			OrderV4: OrderV4{
				Version: 1,
				ID:      nil,
				Proofs:  nil, // no proofs because order has Eip712Signature
				MatcherFeeAsset: OptionalAsset{
					Present: true,
					ID:      stubAssetID,
				}, // waves asset by default
				OrderBody: OrderBody{
					SenderPK:   crypto.PublicKey{}, // empty because this is ethereum-signed order
					MatcherPK:  wavesPKStub,
					AssetPair:  testAssetPair,
					OrderType:  Buy,
					Price:      1,
					Amount:     1,
					Timestamp:  123,
					Expiration: 321,
					MatcherFee: 1,
				},
			},
		}

		valid, err := ethOrder.Verify(CustomNetScheme)
		require.NoError(t, err)
		require.True(t, valid)
	})

	t.Run("Order_Version_Validation", func(t *testing.T) {
		pbTestOrder := func(version int) *g.Order {
			return &g.Order{
				SenderPublicKey:  ethereumPKBytesStub,
				Version:          int32(version),
				MatcherPublicKey: wavesPKStub.Bytes(),
				AssetPair:        new(g.AssetPair),
				MatcherFee:       new(g.Amount),
				Eip712Signature:  ethereumSignatureBytesStub,
			}
		}
		for i := 1; i < 3; i++ {
			pc := ProtobufConverter{}
			_ = pc.extractOrder(pbTestOrder(i))
			require.Error(t, pc.err)
			require.Equal(t, "eip712Signature available only in OrderV4", pc.err.Error())
		}
		pc := ProtobufConverter{}
		_ = pc.extractOrder(pbTestOrder(4))
		require.NoError(t, pc.err)
	})

	t.Run("Contains_Proofs", func(t *testing.T) {
		pbTestOrder := &g.Order{
			SenderPublicKey:  ethereumPKBytesStub,
			Version:          4,
			MatcherPublicKey: wavesPKStub.Bytes(),
			AssetPair:        new(g.AssetPair),
			MatcherFee:       new(g.Amount),
			Eip712Signature:  ethereumSignatureBytesStub,
			Proofs:           [][]byte{[]byte("proof stub1"), []byte("proof stub2")},
		}

		pc := ProtobufConverter{}
		order := pc.extractOrder(pbTestOrder)
		require.NoError(t, pc.err)
		valid, err := order.Valid()
		require.False(t, valid)
		require.Error(t, err)
		require.Equal(t, "eip712Signature excludes proofs", err.Error())
	})
}

func TestEthereumOrderV4_VerifyAndSig(t *testing.T) {
	tests := []struct {
		scheme                 Scheme
		ethSenderPKHex         string
		ethSenderSKHex         string
		ehtSignatureHex        string
		matcherPublicKeyBase58 string
		amountAssetBase58      string
		priceAssetBase58       string
		orderType              OrderType
		price                  uint64
		amount                 uint64
		ts                     uint64
		exp                    uint64
		fee                    uint64
	}{
		{1, "0xc4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53", "0x837cd5bde5402623b2d09c9779bc585cafe5bb1a3d94b369b0b2264f7e1ef45c", "0xfbc3628ab9d7799d172b2398177204297d627c1fd601f98b5a9e2fb726e0d9e77208279f5ab9fba108d155be44eddb31670ca356f724bb8196ac47165eb3eac91b", "E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ", "3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N", "FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2", Sell, 10000000, 1300, 1080, 1031 + MaxOrderTTL, 3},
		{3, "0xc4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53", "0x837cd5bde5402623b2d09c9779bc585cafe5bb1a3d94b369b0b2264f7e1ef45c", "0x1b75b49fb34ef47456cfc9e7bdfc2c240a72919c9044ecf9d6aecf728818d31417a38e461c838b3e91c95104a030c575da7980b74284d8bbfeddbc010a06a5cf1b", "4qoBVcLxf4NEn6e4PyNkYAK4fkTHyH6rK8oKRY1dLfJG", "75NyDiTrambxhHGgBu7JeBqNnTryANZf1v6FTZwwqhDF", "9QT4JoWyk2iC9Y9VBoaEBnEL7gw2ofB8HKKLSEjs3SR", Buy, 100, 345100, 610, 126340 + MaxOrderTTL, 3},
		{42, "0xc4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53", "0x837cd5bde5402623b2d09c9779bc585cafe5bb1a3d94b369b0b2264f7e1ef45c", "0x4c2d9e57907b8b85a05cca6538e842d74802bf38a54c9fca04fd147a9daa444127938f91dd37935e39bde842113d8ca2a1ce26531099205249259ae7b5c30d2e1c", "4uRYdwy5FY7TW66gV7bektQeRfSPJvnvHNLjLzMojJwG", "5zKtndukP6219omWiALaXtuVhMnQoMVQxYkDgGxz8ARG", "mCKpvzjog9DHTwLXNk1NU5aysZ1xpVg3wxLCQ5jWYXR", Sell, 100234000, 176500, 110, 310 + MaxOrderTTL, 3},
		{5, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0x6a07d660074d4fdefc90211f1adcde0de5ee9140ece4640e6ae9828f779b03e545e747de3af73c0c66a9529791b103c1092e698e9f54803b93247d02f68de0c81b", "2HPPrEGV2jqa5W5npRdjV5DTV4wpXnDx1GvKU3dLmnhu", "2AeWVoR5D5CnJbpioSB8qYnYSqJH1ji8Wzf1ubzxsvrK", "4vYK24GfY9F7gEptuXTfLic93dbcVTtxiQoKNZ6zKz5z", Sell, 100034000000, 10053, 104, 180 + MaxOrderTTL, 3},
		{5, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0x047719abade2831f574af600c158c63756cc8c8bf5a807219e44e3a441964c8400dd69b87357b27ffc925987d149f84bc043e879438573c6f3f0a09ac4280cf21c", "6fb8H3bKujzno1x15Ggqgrnhsco6NVWr3e5htGGBBCyA", "6Xqvv5kNtdNDn53foQg9sz8MfaPjxcvaftpLG59qnkD6", "6KPVaF7fn3gw1r4Ee87jAnmaZ7GdFy2DN7cdNmx5ywMA", Buy, 1000000000, 107300, 105, 1330 + MaxOrderTTL, 3},
		{4, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0x0c87e2e9326f81ddcca722f166407bb8a7298bc2d6c0c6841af87bbe5726089c705e7524cc90241cdb472fb7155a38261bebab4285bec34f5616f3106a267ec61c", "26fbBp3oU3yZCGAdGwhiT8wveSjQS4JJbvG1JA88Vpsi", "4PUkX4Se2fYX2TKeuZuxhXNpws5LtKPrtxTY91MTXH1S", "5vBKp4b44cSHvPcPrCpHMZYf8DvRVwKsKXsQas63vQRA", Sell, 100540000000, 64100, 10, 1056 + MaxOrderTTL, 3},
		{3, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0xd9e3b4a8790a43b815996f35aac1522d1e77c9b28d4ddb94e8d947521f75215a7781c01ec59aaeb06130efa2026c39482feaaa312b3d97d4ca3e11eb1e0799851b", "4SFiJcXaqVB2YU6LywpsZ3sjLqNtKJ41aof3jXnZBFPV", "MwU9RtbCZYgMSofewpwoqJVdeDB8MW644ggbLTLH8sv", "6zRVMyyz5miXFfRaFuxvGyFqGBjU1xCGt7VFjyK3eyNx", Buy, 1000000564000, 167300, 190, 310 + MaxOrderTTL, 3},
		{42, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0x198900735a8e2ed273ba3a5ee968f792436b693893b1276b58022889fea9cd6e714b02bdb2ca0fb26cf67a226a9ffa1566f8564f597d42011f7f18c595a936f41c", "2AS8mF66XiKPrEzk3ZQzBC93ef8HWKy2EHBtaU3pWCoh", "QphFC3SdhirC1o365ixoYNUjGVK82gSR7DD5KCMwWke", "3vELR7xbNxFXoxkrNGFQsXXz4gJEacKAgVW9FVCP2Zs9", Sell, 10020000000, 10680, 1110, 10 + MaxOrderTTL, 3},
		{1, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0x5cc58387244c123c163aff1a6c3ecfa3df25e9f6afbd83108899d63bc30fcca01c9537e72825170552141118758464504f54cd4fea2084271efed6b1349e99551b", "2DKxemVYxp1xaAPf2LEy2ovDSsgDDuNS8BMd1QTBwSWD", "4nzaBVK9vS4uB2cYDyLTekhLKZCbCgFVqrZ1VbznRya1", "q23YK66dnRKTsYhr2EYMiJE8cUovoaAFyFcvh2bgX8o", Buy, 10000060, 15400, 1670, 13450 + MaxOrderTTL, 3},
		{3, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0x30409072070e7d666416351551f086832b8f8e62bb8959b69af418b2dac054873c2ce09b4f4e8113d9e928476e8153b8e3637e1f227ee683cf49a063119848c51c", "Gcnp8iu4sFWqs5MfnqNSvn3rJD5XCj68KyBSWUYzg25", "9LAbFKB1NgNzTmc6r1HcC6LQmmQMrEZ1JwxT4Rfhi63", "3hthPo1WU8D3UtvsiTy81cC61uJRFftdtRk77FCeTNiH", Sell, 10073460000000, 45100, 1560, 160 + MaxOrderTTL, 3},
		{42, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0xac2d86ec268c297c62ac379b70938a0a93fc5973ef0f5bbcef0717147061b83207fdb7f2f85450eb4dcc34dd13e8aa78945f85634a8b86a0feba6214e2abdaca1c", "Gcnp8iu4sFWqs5MfnqNSvn3rJD5XCj68KyBSWUYzg25", "", "3hthPo1WU8D3UtvsiTy81cC61uJRFftdtRk77FCeTNiH", Sell, 1007346040000, 451400, 11560, 1605 + MaxOrderTTL, 3},
		{5, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0xda7968d160fb1e47c00b428610cedbfd20e27a6a6bc466b60db11dabe59023d71d2475a6883367e1f9c04e62eb811e33c8d0e26969e5b5566cbcd83e84497ef71c", "Gcnp8iu4sFWqs5MfnqNSvn3rJD5XCj68KyBSWUYzg25", "4nzaBVK9vS4uB2cYDyLTekhLKZCbCgFVqrZ1VbznRya1", "", Buy, 10073460000, 4100, 15160, 1360 + MaxOrderTTL, 3},
	}
	for _, tc := range tests {
		order := newEthereumOrderV4(
			t,
			tc.ethSenderPKHex,
			tc.ehtSignatureHex,
			tc.matcherPublicKeyBase58,
			tc.amountAssetBase58,
			tc.priceAssetBase58,
			tc.orderType,
			tc.price,
			tc.amount,
			tc.ts,
			tc.exp,
			tc.fee,
		)
		// verify check
		valid, err := order.Verify(tc.scheme)
		require.NoError(t, err)
		require.True(t, valid)

		// EthereumSign check
		secretKey, err := crypto.ECDSAPrivateKeyFromHexString(tc.ethSenderSKHex)
		require.NoError(t, err)
		expectedSig := order.Eip712Signature
		order.Eip712Signature = EthereumSignature{}
		err = order.EthereumSign(tc.scheme, (*EthereumPrivateKey)(secretKey))
		require.NoError(t, err)
		require.Equal(t, expectedSig, order.Eip712Signature)
	}
}

func TestEthereumOrderV4_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		scheme             Scheme
		jsonOrder          string
		orderBodyBase58    string
		orderIDBase58      string
		eip712SignatureHex string
	}{
		{
			scheme: TestNetScheme,
			jsonOrder: `
				{
				   "version":4,
				   "id":"fystnKAsFF8U55VGEjUVcT4qkxcinEMBjY7t4wTphB5",
				   "sender":"3N2sMJ78BuYwoLHreuwjbk6dZgsnudxecBR",
				   "senderPublicKey":"5BQPcwDXaZexgonPb8ipDrLRXY3RHn1kFLP9fqp1s6M6xiRhC4LvsAq2HueXCMzkpuXsrLnuBA3SdkJyuhNZXMCd",
				   "matcherPublicKey":"9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
				   "assetPair":{
					  "amountAsset":"5fQPsn8hoaVddFG26cWQ5QFdqxWtUPNaZ9zH2E6LYzFn",
					  "priceAsset":null
				   },
				   "orderType":"sell",
				   "amount":1,
				   "price":100,
				   "timestamp":1,
				   "expiration":123,
				   "matcherFee":100000,
				   "signature":"",
				   "proofs":[
			
				   ],
				   "matcherFeeAssetId":null,
				   "eip712Signature":"0xc8ba2bdafd27742546b3be34883efc51d6cdffbb235798d7b51876c6854791f019b0522d7a39b6f2087cba46ae86919b71a2d9d7920dfc8e00246d8f02a258f21b"
				}`,
			orderBodyBase58:    "J2bf9Vo1nP6PGNJhQDCH9YQDW9nRBXajxTAk9Jqc5MXKNKPVHbPZJzMA4ByaWT9zLph2JQZ9sC5Dh9KLtARRiPBW4hzwq6xezdEmPAky1KjJM6eebYYcFzpkY1ZNaF68fNXbUmzSDS4Lu2SQPxKm6UowSSWV9SVjpwRNAjKbCg6cnBD2A3mVZ5Vw4iWgmHchAH9P73fW9x56jzWuSCvTSZsyVAGWdL8UGiLaC59txcskQwanfZfd5QuPTZ1NTSmy925rz6n32dcCmM1sJLTrV9gRmqRQLpzgvzQA9VgWC9bzvmQW",
			orderIDBase58:      "fystnKAsFF8U55VGEjUVcT4qkxcinEMBjY7t4wTphB5",
			eip712SignatureHex: "0xc8ba2bdafd27742546b3be34883efc51d6cdffbb235798d7b51876c6854791f019b0522d7a39b6f2087cba46ae86919b71a2d9d7920dfc8e00246d8f02a258f21b",
		},
		{
			scheme: TestNetScheme,
			jsonOrder: `
				{
				   "version":4,
				   "id":"6XTFGCXmEDeXrsWXVXWty876HNTMBeX6vTgQoAw3WnH2",
				   "sender":"3N2sMJ78BuYwoLHreuwjbk6dZgsnudxecBR",
				   "senderPublicKey":"5BQPcwDXaZexgonPb8ipDrLRXY3RHn1kFLP9fqp1s6M6xiRhC4LvsAq2HueXCMzkpuXsrLnuBA3SdkJyuhNZXMCd",
				   "matcherPublicKey":"9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
				   "assetPair":{
					  "amountAsset":"5fQPsn8hoaVddFG26cWQ5QFdqxWtUPNaZ9zH2E6LYzFn",
					  "priceAsset":null
				   },
				   "orderType":"buy",
				   "amount":1,
				   "price":100,
				   "timestamp":1,
				   "expiration":123,
				   "matcherFee":100000,
				   "signature":"",
				   "proofs":[
					  
				   ],
				   "matcherFeeAssetId":null,
				   "eip712Signature":"0xe5ff562bfb0296e95b631365599c87f1c5002597bf56a131f289765275d2580f5344c62999404c37cd858ea037328ac91eca16ad1ce69c345ebb52fde70b66251c"
				}`,
			orderBodyBase58:    "shFRJaXeYerjLQyMtvdYADWwg97iDVfxLcAXSFVYRsvGkujuY76NnKEfoztWQpsSKYzZiTr1BdVVrPn3AqJfDfmkA2ss5PGtU6DZbVZEUbc6U6vtrYA7986kVioKbySLJheEA8bnd3TPNWJXPs1bNA48nUkRiE7s7mbWGDCSzjncjwmPu4hbEHd6TAw2G2UxA5kVp2ivYFjHUDSm1NkZEHfgqtFLtZmrTYMZCCuCH1PiBo37K4RRnyvXogxdLp3pWZtCdCwWjVRUS19EFZDjyXn13XiZKtHSbu7y2hhgKFKyu",
			orderIDBase58:      "6XTFGCXmEDeXrsWXVXWty876HNTMBeX6vTgQoAw3WnH2",
			eip712SignatureHex: "0xe5ff562bfb0296e95b631365599c87f1c5002597bf56a131f289765275d2580f5344c62999404c37cd858ea037328ac91eca16ad1ce69c345ebb52fde70b66251c",
		},
	}
	for _, tc := range tests {
		var (
			err                     error
			expectedOrderID         []byte
			expectedOrderBodyBytes  []byte
			expectedEip712Signature EthereumSignature
		)
		expectedOrderID, err = base58.Decode(tc.orderIDBase58)
		require.NoError(t, err)
		expectedEip712Signature, err = NewEthereumSignatureFromHexString(tc.eip712SignatureHex)
		require.NoError(t, err)
		expectedOrderBodyBytes, err = base58.Decode(tc.orderBodyBase58)
		require.NoError(t, err)

		order := EthereumOrderV4{}
		err = json.Unmarshal([]byte(tc.jsonOrder), &order)
		require.NoError(t, err)

		require.Equal(t, expectedEip712Signature, order.Eip712Signature)

		actualOrderBody, err := order.BodyMarshalBinary(tc.scheme)
		require.NoError(t, err)
		require.Equal(t, expectedOrderBodyBytes, actualOrderBody)

		err = order.GenerateID(tc.scheme)
		require.NoError(t, err)
		actualOrderID, err := order.GetID()
		require.NoError(t, err)
		require.Equal(t, expectedOrderID, actualOrderID)
	}
}
