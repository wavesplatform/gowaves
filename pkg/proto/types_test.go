package proto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"

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
		err = o.Sign(TestNetScheme, sk)
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
		if err := o.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := o.Verify(TestNetScheme); assert.NoError(t, err) {
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
	err := o.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	s := serializer.New(io.Discard)

	t.StopTimer()

	t.Run("serialize", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = o.Serialize(s)
		}
		b.StopTimer()
		if err != nil {
			b.FailNow()
		}
	})
	t.Run("marshal", func(b *testing.B) {
		var bts []byte
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bts, err = o.MarshalBinary()
		}
		b.StopTimer()
		if err != nil || len(bts) == 0 {
			b.FailNow()
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
			assert.JSONEq(t, ej, string(j))
			if err := o.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if j, err := json.Marshal(o); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
						base58.Encode(o.ID[:]), base58.Encode(o.Signature[:]), base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
					assert.JSONEq(t, ej, string(j))
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
		err = o.Sign(TestNetScheme, sk)
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
		if err := o.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := o.Verify(TestNetScheme); assert.NoError(t, err) {
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
			assert.JSONEq(t, ej, string(j))
			if err := o.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if j, err := json.Marshal(o); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
						base58.Encode(o.ID[:]), base58.Encode(o.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
					assert.JSONEq(t, ej, string(j))
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
		err = o.Sign(TestNetScheme, sk)
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
		if err := o.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := o.Verify(TestNetScheme); assert.NoError(t, err) {
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
			assert.JSONEq(t, ej, string(j))
			if err := o.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if j, err := json.Marshal(o); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"version\":3,\"id\":\"%s\",\"proofs\":[\"%s\"],\"matcherFeeAssetId\":%s,\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
						base58.Encode(o.ID[:]), base58.Encode(o.Proofs.Proofs[0]), fas, base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
					assert.JSONEq(t, ej, string(j))
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

func TestNewDataEntryFromJSON(t *testing.T) {
	tests := []struct {
		json     string
		expected DataEntry
	}{
		{`{"key":"k1","type":"integer","value":12345}`, &IntegerDataEntry{Key: "k1", Value: 12345}},
		{`{"key":"k2","type":"string","value":"test-string"}`, &StringDataEntry{Key: "k2", Value: "test-string"}},
		{`{"key":"k3","type":"boolean","value":true}`, &BooleanDataEntry{Key: "k3", Value: true}},
		{`{"key":"k4","value":null}`, &DeleteDataEntry{Key: "k4"}},
		{
			`{"key":"k5","type":"binary","value":"base64:JH9xFB0dBYAX9BohYq06cMrtwta9mEoaj0aSVpLApyc="}`,
			&BinaryDataEntry{Key: "k5", Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}},
		},
	}
	for _, tc := range tests {
		actual, err := NewDataEntryFromJSON([]byte(tc.json))
		require.NoError(t, err)
		assert.Equal(t, tc.expected, actual)
		js, err := json.Marshal(actual)
		require.NoError(t, err)
		assert.JSONEq(t, tc.json, string(js))
	}
}

func TestDataEntries_Valid(t *testing.T) {
	ieFail := &IntegerDataEntry{Key: "", Value: 1234567890}
	beFail := &BooleanDataEntry{Key: strings.Repeat("too-big-key", 10), Value: false}
	seFail := &StringDataEntry{Key: "fail-string-entry", Value: strings.Repeat("too-big-value", 2521)}
	tests := []struct {
		entries        DataEntries
		utf16KeyLen    bool
		forbidEmptyKey bool
		err            string
		valid          bool
	}{
		{[]DataEntry{ieFail}, true, true, "invalid entry 0: empty entry key", false},
		{[]DataEntry{seFail}, true, true, "invalid entry 0: value is too large", false},
		{[]DataEntry{seFail}, true, false, "invalid entry 0: value is too large", false},
		{[]DataEntry{beFail, ieFail}, true, true, "invalid entry 0: key is too large", false},
		{[]DataEntry{beFail, ieFail}, false, true, "invalid entry 1: empty entry key", false},
		{[]DataEntry{}, false, true, "", true},
		{[]DataEntry{ieFail}, true, false, "", true},
		{
			[]DataEntry{
				&StringDataEntry{Key: "1", Value: "1"},
				&IntegerDataEntry{Key: "2", Value: 2},
				&BooleanDataEntry{Key: "3", Value: true},
			},
			false,
			true,
			"",
			true,
		},
	}
	for i, tc := range tests {
		err := tc.entries.Valid(tc.forbidEmptyKey, tc.utf16KeyLen)
		if tc.valid {
			assert.NoError(t, err, "#%d", i)
		} else {
			assert.Error(t, err, "#%d", i)
			assert.EqualError(t, err, tc.err, "#%d", i)
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
	for _, test := range []string{
		"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc",
		"1",
		"111111111111111",
	} {
		bv, err := base58.Decode(test)
		require.NoError(t, err)
		v := BinaryArgument{bv}
		if b, err := v.MarshalJSON(); assert.NoError(t, err) {
			js := string(b)
			ejs := fmt.Sprintf("{\"type\":\"binary\",\"value\":\"%s\"}", test)
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
		{"[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true},{\"type\":\"binary\",\"value\":\"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc\"}]",
			Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}},
		},
		{"[{\"type\":\"string\",\"value\":\"blah-blah\"}]",
			Arguments{&StringArgument{Value: "blah-blah"}},
		},
		{"[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true},{\"type\":\"binary\",\"value\":\"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc\"},{\"type\":\"string\",\"value\":\"blah-blah\"}]",
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
		{fc: FunctionCall{Name: "baz", Arguments: Arguments{&BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}}}, js: "{\"function\":\"baz\",\"args\":[{\"type\":\"binary\",\"value\":\"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc\"}]}"},
		{fc: FunctionCall{Name: "foobar0", Arguments: Arguments{&StringArgument{Value: "blah-blah"}}}, js: "{\"function\":\"foobar0\",\"args\":[{\"type\":\"string\",\"value\":\"blah-blah\"}]}"},
		{fc: FunctionCall{Name: "foobar1", Arguments: Arguments{}}, js: "{\"function\":\"foobar1\",\"args\":[]}"},
		{fc: FunctionCall{Name: "foobar2", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}}}, js: "{\"function\":\"foobar2\",\"args\":[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true}]}"},
		{fc: FunctionCall{Name: "foobar3", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}}}, js: "{\"function\":\"foobar3\",\"args\":[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true},{\"type\":\"binary\",\"value\":\"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc\"}]}"},
		{fc: FunctionCall{Name: "foobar4", Arguments: Arguments{&IntegerArgument{Value: 12345}, &BooleanArgument{Value: true}, &BinaryArgument{Value: B58Bytes{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}}, &StringArgument{Value: "blah-blah"}}}, js: "{\"function\":\"foobar4\",\"args\":[{\"type\":\"integer\",\"value\":12345},{\"type\":\"boolean\",\"value\":true},{\"type\":\"binary\",\"value\":\"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc\"},{\"type\":\"string\",\"value\":\"blah-blah\"}]}"},
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
	const (
		ethChainIdByte = 'E'
	)
	ethStub32 := bytes.Repeat([]byte{ethChainIdByte}, 32)

	stubAssetID, err := crypto.NewDigestFromBytes(ethStub32)
	require.NoError(t, err)

	wavesPKStub, err := crypto.NewPublicKeyFromBytes(ethStub32)
	require.NoError(t, err)

	ethereumSignatureBytesStub, err := DecodeFromHexString("0x54119bc5b24d9363b7a1a31a71a2e6194dfeedc5e9644893b0a04bb57004e5b14342c1ce29ee00877da49180fd6d7fb332ff400231f809da7ed0dcb07c504e2d1c")
	require.NoError(t, err)

	t.Run("Verify-Scala", func(t *testing.T) {
		// taken from scala-node tests

		asset := *NewOptionalAssetFromDigest(stubAssetID)

		testEthPubKeyHex := "0xf69531bdb61b48f8cd4963291d07773d09b07081795dae2a43931a5c3cd86e15018836e653bc7c1e6a2718c9b28a9f299d4b86d956488b432ab719d5cc962d2e"
		testEthSenderPK, err := NewEthereumPublicKeyFromHexString(testEthPubKeyHex)
		require.NoError(t, err)

		testEthSigHex := "0xfe56e1cbd6945f1e17ce9f9eb21172dd7810bcc74651dd7d3eaeca5d9ae0409113e5236075841af8195cb4dba3947ae9b99dbd560fd0c43afe89cc0b648690321c"
		testEthSig, err := NewEthereumSignatureFromHexString(testEthSigHex)
		require.NoError(t, err)

		ethOrder := EthereumOrderV4{
			SenderPK:        ethereumPublicKeyBase58Wrapper{inner: &testEthSenderPK},
			Eip712Signature: testEthSig,
			OrderV4: OrderV4{
				PriceMode:       OrderPriceModeFixedDecimals,
				Version:         4,
				ID:              nil,
				Proofs:          nil, // no proofs because order has Eip712Signature
				MatcherFeeAsset: asset,
				OrderBody: OrderBody{
					SenderPK:   crypto.PublicKey{}, // empty because this is ethereum-signed order
					MatcherPK:  wavesPKStub,
					AssetPair:  AssetPair{AmountAsset: asset, PriceAsset: asset},
					OrderType:  Buy,
					Price:      1,
					Amount:     1,
					Timestamp:  123,
					Expiration: 321,
					MatcherFee: 1,
				},
			},
		}

		valid, err := ethOrder.Verify(TestNetScheme)
		require.NoError(t, err)
		require.True(t, valid)
	})

	t.Run("Order_Version_Validation", func(t *testing.T) {
		pbTestOrder := func(version int) *g.Order {
			return &g.Order{
				Version:          int32(version),
				MatcherPublicKey: wavesPKStub.Bytes(),
				AssetPair:        new(g.AssetPair),
				MatcherFee:       new(g.Amount),
				Sender:           &g.Order_Eip712Signature{Eip712Signature: ethereumSignatureBytesStub},
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
			Version:          4,
			MatcherPublicKey: wavesPKStub.Bytes(),
			AssetPair:        new(g.AssetPair),
			MatcherFee:       new(g.Amount),
			Sender:           &g.Order_Eip712Signature{Eip712Signature: ethereumSignatureBytesStub},
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
	const (
		invalidSenderPKHexStub = "0xd10a150ba9a535125481e017a09c2ac6a1ab43fc43f7ab8f0d44635106672dd7de4f775c06b730483862cbc4371a646d86df77b3815593a846b7272ace008c42"
	)
	tests := []struct {
		needCheckSig           bool
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
		prideMode              OrderPriceMode
	}{
		{true, 5, "0xd07059fbd0269ea7cb23792201f3f0834c152d503399ae83e7dbc5ee512ebdef22b459d1423a92d2a2906ba24a50b58435e5d236b4f5a820f3be23a191b412af", "0x5d7ee7f7ca3e71e37b7a2d713fb25d50d694480ee27f3d79a1ce1801dc9c5726", "0x05c8d63b42b3eb072e16efe40c448d5c7719a5ca84f6f04fff5467032186d9bb222943527e054b979ec037a665323bdaa3f9f299bc90e617fbe9c55c54a73b941b", "6fb8H3bKujzno1x15Ggqgrnhsco6NVWr3e5htGGBBCyA", "6Xqvv5kNtdNDn53foQg9sz8MfaPjxcvaftpLG59qnkD6", "6KPVaF7fn3gw1r4Ee87jAnmaZ7GdFy2DN7cdNmx5ywMA", Buy, 1000000000, 107300, 105, 1330 + MaxOrderTTL, 3, OrderPriceModeFixedDecimals},
		{true, 4, "0xd07059fbd0269ea7cb23792201f3f0834c152d503399ae83e7dbc5ee512ebdef22b459d1423a92d2a2906ba24a50b58435e5d236b4f5a820f3be23a191b412af", "0x5d7ee7f7ca3e71e37b7a2d713fb25d50d694480ee27f3d79a1ce1801dc9c5726", "0x33499f53db48be48c622245ba19af8135a241b5e6a6ad950380f612c112189782860ebbf42a1f8a1e55264700b4b143cb25f6e6b45168cbd7703252cec0a6f5c1c", "26fbBp3oU3yZCGAdGwhiT8wveSjQS4JJbvG1JA88Vpsi", "4PUkX4Se2fYX2TKeuZuxhXNpws5LtKPrtxTY91MTXH1S", "5vBKp4b44cSHvPcPrCpHMZYf8DvRVwKsKXsQas63vQRA", Sell, 100540000000, 64100, 10, 1056 + MaxOrderTTL, 3, OrderPriceModeAssetDecimals},
		{true, 3, "0xd07059fbd0269ea7cb23792201f3f0834c152d503399ae83e7dbc5ee512ebdef22b459d1423a92d2a2906ba24a50b58435e5d236b4f5a820f3be23a191b412af", "0x5d7ee7f7ca3e71e37b7a2d713fb25d50d694480ee27f3d79a1ce1801dc9c5726", "0x9e625c07a2c949ad841c7364d21527b2280dd8671495804a7a923a495c451e971bc489ed2d912d99629ac2f9138d3bb4fcd2c485a56c674c1deca5a2f52876881c", "4SFiJcXaqVB2YU6LywpsZ3sjLqNtKJ41aof3jXnZBFPV", "MwU9RtbCZYgMSofewpwoqJVdeDB8MW644ggbLTLH8sv", "6zRVMyyz5miXFfRaFuxvGyFqGBjU1xCGt7VFjyK3eyNx", Buy, 1000000564000, 167300, 190, 310 + MaxOrderTTL, 3, OrderPriceModeDefault},
		{true, 42, "0xd07059fbd0269ea7cb23792201f3f0834c152d503399ae83e7dbc5ee512ebdef22b459d1423a92d2a2906ba24a50b58435e5d236b4f5a820f3be23a191b412af", "0x5d7ee7f7ca3e71e37b7a2d713fb25d50d694480ee27f3d79a1ce1801dc9c5726", "0xd9c8215c7757265a3ae8a4bd2e78b74d5e827a3fe6ede009df8608836baff87734159bb976ceea950b89d247be0fad8ce79af9507e39a7bc8aef6faa50c96cc21b", "2AS8mF66XiKPrEzk3ZQzBC93ef8HWKy2EHBtaU3pWCoh", "QphFC3SdhirC1o365ixoYNUjGVK82gSR7DD5KCMwWke", "3vELR7xbNxFXoxkrNGFQsXXz4gJEacKAgVW9FVCP2Zs9", Sell, 10020000000, 10680, 1110, 10 + MaxOrderTTL, 3, OrderPriceModeFixedDecimals},
		{true, 1, "0xd07059fbd0269ea7cb23792201f3f0834c152d503399ae83e7dbc5ee512ebdef22b459d1423a92d2a2906ba24a50b58435e5d236b4f5a820f3be23a191b412af", "0x5d7ee7f7ca3e71e37b7a2d713fb25d50d694480ee27f3d79a1ce1801dc9c5726", "0x4b0d292d1d4bce507319bb112e78ec32b5b11021ee283853a48f3c1f46e513270c397138ff3e78bedfd22c0e7272293684339f9a6a6d9827a8bdf2c97849e93a1b", "2DKxemVYxp1xaAPf2LEy2ovDSsgDDuNS8BMd1QTBwSWD", "4nzaBVK9vS4uB2cYDyLTekhLKZCbCgFVqrZ1VbznRya1", "q23YK66dnRKTsYhr2EYMiJE8cUovoaAFyFcvh2bgX8o", Buy, 10000060, 15400, 1670, 13450 + MaxOrderTTL, 3, OrderPriceModeDefault},
		{true, 3, "0xd07059fbd0269ea7cb23792201f3f0834c152d503399ae83e7dbc5ee512ebdef22b459d1423a92d2a2906ba24a50b58435e5d236b4f5a820f3be23a191b412af", "0x5d7ee7f7ca3e71e37b7a2d713fb25d50d694480ee27f3d79a1ce1801dc9c5726", "0x22943810479c1de98d03399ef8df972c7c022db77453dff405e9c591840a372f3df32cc7d68ad528fbae587835b42975115ceeed0196cbd24f9517ed02be7c5d1b", "Gcnp8iu4sFWqs5MfnqNSvn3rJD5XCj68KyBSWUYzg25", "9LAbFKB1NgNzTmc6r1HcC6LQmmQMrEZ1JwxT4Rfhi63", "3hthPo1WU8D3UtvsiTy81cC61uJRFftdtRk77FCeTNiH", Sell, 10073460000000, 45100, 1560, 160 + MaxOrderTTL, 3, OrderPriceModeDefault},
		{true, 42, "0xd07059fbd0269ea7cb23792201f3f0834c152d503399ae83e7dbc5ee512ebdef22b459d1423a92d2a2906ba24a50b58435e5d236b4f5a820f3be23a191b412af", "0x5d7ee7f7ca3e71e37b7a2d713fb25d50d694480ee27f3d79a1ce1801dc9c5726", "0x500dcb2515c58905d9f2f10b66a3b08de99094935b3115a196b23dd58c3f34fd2dcb4755dfb2455550a67cb50d0633d0ce03851dd6e2283ee3c1322946fdcf8e1b", "Gcnp8iu4sFWqs5MfnqNSvn3rJD5XCj68KyBSWUYzg25", "", "3hthPo1WU8D3UtvsiTy81cC61uJRFftdtRk77FCeTNiH", Sell, 1007346040000, 451400, 11560, 1605 + MaxOrderTTL, 3, OrderPriceModeAssetDecimals},
		{true, 5, "0xd07059fbd0269ea7cb23792201f3f0834c152d503399ae83e7dbc5ee512ebdef22b459d1423a92d2a2906ba24a50b58435e5d236b4f5a820f3be23a191b412af", "0x5d7ee7f7ca3e71e37b7a2d713fb25d50d694480ee27f3d79a1ce1801dc9c5726", "0xf164a8179e93758879a2214ddacf5b505427282b1f92333fc5477fc43e3d6a7735ea47739226329354a2044043157b6bfaebb488328b81e8348fccbc438a7bfe1b", "Gcnp8iu4sFWqs5MfnqNSvn3rJD5XCj68KyBSWUYzg25", "4nzaBVK9vS4uB2cYDyLTekhLKZCbCgFVqrZ1VbznRya1", "", Buy, 10073460000, 4100, 15160, 1360 + MaxOrderTTL, 3, OrderPriceModeFixedDecimals},

		{true, 1, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0xeabe4f424019b4355888ac9bffce201ed0e4118c9fcbe9291c474fb3745e775e2b7ac81d74e9d696da2bcd8f623c434a8ab5cf3f5675b19dc1f25577a9f572e31b", "E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ", "3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N", "FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2", Sell, 10000000, 1300, 1080, 1031 + MaxOrderTTL, 3, OrderPriceModeDefault},
		{true, 3, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0x018a705460523bcec16f17a762bb3aa91f93cabedc064fc475b21b53e969daf756a4b38bb5cda7d95bc80059c312be751e672edf2d4e66c033ddc9baa1525e891c", "4qoBVcLxf4NEn6e4PyNkYAK4fkTHyH6rK8oKRY1dLfJG", "75NyDiTrambxhHGgBu7JeBqNnTryANZf1v6FTZwwqhDF", "9QT4JoWyk2iC9Y9VBoaEBnEL7gw2ofB8HKKLSEjs3SR", Buy, 100, 345100, 610, 126340 + MaxOrderTTL, 3, OrderPriceModeAssetDecimals},
		{true, 42, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0xc7a3879a4c44365a2011af6a8b6bfd550a0ddaf2a3f3a4264ff995a345cdb60e1222b64f856986c05d4f257b2992cc981a56bcd305069148156270382f6041ac1c", "4uRYdwy5FY7TW66gV7bektQeRfSPJvnvHNLjLzMojJwG", "5zKtndukP6219omWiALaXtuVhMnQoMVQxYkDgGxz8ARG", "mCKpvzjog9DHTwLXNk1NU5aysZ1xpVg3wxLCQ5jWYXR", Sell, 100234000, 176500, 110, 310 + MaxOrderTTL, 3, OrderPriceModeFixedDecimals},

		// test case with EIP-155 v param
		{false, 42, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0xc7a3879a4c44365a2011af6a8b6bfd550a0ddaf2a3f3a4264ff995a345cdb60e1222b64f856986c05d4f257b2992cc981a56bcd305069148156270382f6041ac78", "4uRYdwy5FY7TW66gV7bektQeRfSPJvnvHNLjLzMojJwG", "5zKtndukP6219omWiALaXtuVhMnQoMVQxYkDgGxz8ARG", "mCKpvzjog9DHTwLXNk1NU5aysZ1xpVg3wxLCQ5jWYXR", Sell, 100234000, 176500, 110, 310 + MaxOrderTTL, 3, OrderPriceModeFixedDecimals},
		// test case with v = 1
		{false, 42, "0xae8e4abe5917a6881324538081b71d88d50f53cfd9a2d53e16208c3ff45126649320928e65bbbd2e7c43adb57a9250e8ddcc645f9cacf673d7a9c13d03aa2743", "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4", "0xc7a3879a4c44365a2011af6a8b6bfd550a0ddaf2a3f3a4264ff995a345cdb60e1222b64f856986c05d4f257b2992cc981a56bcd305069148156270382f6041ac01", "4uRYdwy5FY7TW66gV7bektQeRfSPJvnvHNLjLzMojJwG", "5zKtndukP6219omWiALaXtuVhMnQoMVQxYkDgGxz8ARG", "mCKpvzjog9DHTwLXNk1NU5aysZ1xpVg3wxLCQ5jWYXR", Sell, 100234000, 176500, 110, 310 + MaxOrderTTL, 3, OrderPriceModeFixedDecimals},
	}
	for _, tc := range tests {
		order := newEthereumOrderV4(
			t,
			invalidSenderPKHexStub,
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
			tc.prideMode,
		)
		// verify check with invalid senderPK
		valid, err := order.Verify(tc.scheme)
		require.NoError(t, err)
		require.False(t, valid)

		// generate valid senderPK
		err = order.GenerateSenderPK(tc.scheme)
		require.NoError(t, err)
		require.Equal(t, tc.ethSenderPKHex, order.SenderPK.inner.String())

		// verify check with valid senderPK
		valid, err = order.Verify(tc.scheme)
		require.NoError(t, err)
		require.True(t, valid)

		if tc.needCheckSig {
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
}

func TestEthereumOrderV4_VerifyFromJSONWithLeadingPKZeroes(t *testing.T) {
	const (
		js = `
		{
		  "amount": 211125290,
		  "amountAsset": "WAVES",
		  "assetPair": {
			"amountAsset": "WAVES",
			"priceAsset": "34N9YcEETLWn93qYQ64EsP1x89tSruJU44RrEMSXXEPJ"
		  },
		  "eip712Signature": "0x4305a6f070179f7d5fa10557d764373d740ecb24a1177e8c2e01cc03f7c90eda78af2bdc88c964032ed3ae3807eed05c20c981ffe7b30e060f9f145290905b8a1b",
		  "expiration": 1671111399020,
		  "matcherFee": 23627,
		  "matcherFeeAssetId": "34N9YcEETLWn93qYQ64EsP1x89tSruJU44RrEMSXXEPJ",
		  "matcherPublicKey": "9cpfKN9suPNvfeUNphzxXMjcnn974eme8ZhWUjaktzU5",
		  "orderType": "buy",
		  "price": 2357071,
		  "priceAsset": "34N9YcEETLWn93qYQ64EsP1x89tSruJU44RrEMSXXEPJ",
		  "timestamp": 1668605799020,
		  "version": 4,
		  "priceMode": "assetDecimals"
		}`
		expectedSenderPK = "0x0052da038439eaba660a7e5764b7e278efaa22ef3f861b965dfd7a8101b27def602238ff11bdb36887da48afbec98026505e59cbcec23c71b9977ed855aaf3b2"
		scheme           = MainNetScheme
	)
	order := new(EthereumOrderV4)
	err := json.Unmarshal([]byte(js), order)
	require.NoError(t, err)

	err = order.GenerateSenderPK(scheme)
	require.NoError(t, err)
	senderPK := order.SenderPK.inner.String()
	assert.Equal(t, expectedSenderPK, senderPK)

	ok, err := order.Verify(scheme)
	assert.True(t, ok)
	assert.NoError(t, err)
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
			scheme: CustomNetScheme,
			jsonOrder: `
				{
				  "version": 4,
				  "id": "6MvB6QT3TbDymV1zU8VbKpQnJsPLvMXsUDMfMs1rda7t",
				  "sender": "3G9uRSP4uVjTFjGZixYW4arBZUKWHxjnfeW",
				  "senderPublicKey": "5vwTDMooR7Hp57MekN7qHz7fHNVrkn2Nx4CiWdq4cyBR4LNnZWYAr7UfBbzhmSvtNkv6e45aJ4Q4aKCSinyHVw33",
				  "matcherPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
				  "assetPair": {
					"amountAsset": "5fQPsn8hoaVddFG26cWQ5QFdqxWtUPNaZ9zH2E6LYzFn",
					"priceAsset": null
				  },
				  "orderType": "buy",
				  "amount": 1,
				  "price": 100,
				  "timestamp": 1,
				  "expiration": 123,
				  "matcherFee": 100000,
				  "signature": "",
				  "proofs": [],
				  "matcherFeeAssetId": null,
				  "eip712Signature": "0x0a897d382e4e4a066e1d98e5c3c1051864a557c488571ff71e036c0f5a2c7204274cb293cd4aa7ad40f8c2f650e1a2770ecca6aa14a1da883388fa3b5b9fa8b71c",
				  "priceMode": null
				}`,
			orderBodyBase58:    "WHGTNQc2MuaLdLdqibbZKEL9C8aTJkFXzWQvrebyo1qvg3GL3BQf8jQxJEqYJUbWe6V5zHBdZCKH9rtYPF1CT1jJH4KbP7PjiGJfPvQEyzq8UCqsJGKnrWmB9EvnAwdnMVp4CLCVs9fRgfjASZBrSnLQJ9tWFc8TC5TRAhQYWGXvxeNxn2BtnGVyoFsRXF8hy3FaY5aPHyahM5R7Qks",
			orderIDBase58:      "6mYgEbEgrx5M6zMRbpQFbzKHcDy78bYXLgBzygTDGM4a",
			eip712SignatureHex: "0x0a897d382e4e4a066e1d98e5c3c1051864a557c488571ff71e036c0f5a2c7204274cb293cd4aa7ad40f8c2f650e1a2770ecca6aa14a1da883388fa3b5b9fa8b71c",
		},
		{
			scheme: CustomNetScheme,
			jsonOrder: `
				{
				  "version": 4,
				  "id": "49iXxoHPbwVfvTh6437BAWwnGBnkUrFYjtVfwS9bKBL6",
				  "sender": "3G9uRSP4uVjTFjGZixYW4arBZUKWHxjnfeW",
				  "senderPublicKey": "5vwTDMooR7Hp57MekN7qHz7fHNVrkn2Nx4CiWdq4cyBR4LNnZWYAr7UfBbzhmSvtNkv6e45aJ4Q4aKCSinyHVw33",
				  "matcherPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
				  "assetPair": {
					"amountAsset": "5fQPsn8hoaVddFG26cWQ5QFdqxWtUPNaZ9zH2E6LYzFn",
					"priceAsset": null
				  },
				  "orderType": "sell",
				  "amount": 1,
				  "price": 100,
				  "timestamp": 1,
				  "expiration": 123,
				  "matcherFee": 100000,
				  "signature": "",
				  "proofs": [],
				  "matcherFeeAssetId": null,
				  "eip712Signature": "0x6c4385dd5f6f1200b4d0630c9076104f34c801c16a211e505facfd743ba242db4429b966ffa8d2a9aff9037dafda78cfc8f7c5ef1c94493f5954bc7ebdb649281b",
				  "priceMode": null
				}`,
			orderBodyBase58:    "AqRtygFPj6Dru5d3Pc49tBwT1HHrFCvRBoG5AcaBu5aG9KtRUr9dCJznkh3PjVKRU5fHpEcLzTjELSWMitAHt31R2geb1ZQ1XZwp1gcR8HVV5Nzc6rqQUPw3FGJwKA6vXRSy6R5g6s4hxSuPDaZieFdyZ2rgbZ8BPYSyuG8K3kUJ7iBTK1i7bhmLd2Usb4XHCAJR4MAArQmBGZHz74xXvS",
			orderIDBase58:      "8274Mc8WiNQdP3YhinBGkEX79AcZe5th51DJCTW8rEUZ",
			eip712SignatureHex: "0x6c4385dd5f6f1200b4d0630c9076104f34c801c16a211e505facfd743ba242db4429b966ffa8d2a9aff9037dafda78cfc8f7c5ef1c94493f5954bc7ebdb649281b",
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

func TestOrderPriceMode_FromProtobuf(t *testing.T) {
	tests := []struct {
		pbMode   g.Order_PriceMode
		expected OrderPriceMode
		isErr    bool
	}{
		{g.Order_DEFAULT, OrderPriceModeDefault, false},
		{g.Order_FIXED_DECIMALS, OrderPriceModeFixedDecimals, false},
		{g.Order_ASSET_DECIMALS, OrderPriceModeAssetDecimals, false},
		{g.Order_PriceMode(4235), 0, true},
		{g.Order_PriceMode(-1), 0, true},
	}
	for _, tc := range tests {
		var actual OrderPriceMode
		err := actual.FromProtobuf(tc.pbMode)
		if tc.isErr {
			require.Error(t, err)
		} else {
			require.Equal(t, tc.expected, actual)
		}
	}
}

func TestOrderPriceMode_ToProtobuf(t *testing.T) {
	tests := []struct {
		testName string
		expected g.Order_PriceMode
		mode     OrderPriceMode
		panic    bool
	}{
		{"To_Order_DEFAULT", g.Order_DEFAULT, OrderPriceModeDefault, false},
		{"To_Order_FIXED_DECIMALS", g.Order_FIXED_DECIMALS, OrderPriceModeFixedDecimals, false},
		{"To_Order_ASSET_DECIMALS", g.Order_ASSET_DECIMALS, OrderPriceModeAssetDecimals, false},
		{"Invalid", 0, 255, true},
	}
	for i := range tests {
		tc := tests[i]
		t.Run(tc.testName, func(t *testing.T) {
			if tc.panic {
				require.Panics(t, func() {
					tc.mode.ToProtobuf()
				})
			} else {
				actual := tc.mode.ToProtobuf()
				require.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestOrderPriceMode_Valid(t *testing.T) {
	tests := []struct {
		orderVersion byte
		mode         OrderPriceMode
		valid        bool
	}{
		{1, OrderPriceModeFixedDecimals, false},
		{1, OrderPriceModeAssetDecimals, false},
		{1, OrderPriceModeDefault, true},
		{2, OrderPriceModeFixedDecimals, false},
		{2, OrderPriceModeAssetDecimals, false},
		{2, OrderPriceModeDefault, true},
		{3, OrderPriceModeFixedDecimals, false},
		{3, OrderPriceModeAssetDecimals, false},
		{3, OrderPriceModeDefault, true},
		{4, OrderPriceModeFixedDecimals, true},
		{4, OrderPriceModeAssetDecimals, true},
		{4, 255, false},
	}
	for _, tc := range tests {
		ok, err := tc.mode.Valid(tc.orderVersion)
		require.Equal(t, tc.valid, ok)
		if tc.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestByteVectorJSONRoundTrip(t *testing.T) {
	big := make([]byte, 2046)
	bigBase64 := "\"base64:" + strings.Repeat("A", 2728) + "\""
	for _, test := range []struct {
		bv ByteVector
		js string
	}{
		{bv: ByteVector{0x24, 0x7f, 0x71, 0x14, 0x1d, 0x1d, 0x05, 0x80, 0x17, 0xf4, 0x1a, 0x21, 0x62, 0xad, 0x3a, 0x70, 0xca, 0xed, 0xc2, 0xd6, 0xbd, 0x98, 0x4a, 0x1a, 0x8f, 0x46, 0x92, 0x56, 0x92, 0xc0, 0xa7, 0x27}, js: `"3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"`},
		{bv: ByteVector{}, js: `""`},
		{bv: ByteVector(big), js: bigBase64},
	} {
		if b, err := json.Marshal(test.bv); assert.NoError(t, err) {
			assert.Equal(t, test.js, string(b))
			bv := ByteVector{}
			if err := json.Unmarshal(b, &bv); assert.NoError(t, err) {
				assert.Equal(t, test.bv, bv)
			}
		}
	}
}

func TestStateHashDebutUnmarshalJSON(t *testing.T) {
	for _, test := range []struct {
		js  string
		ver string
		h   int
	}{
		{`{"accountScriptHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","aliasHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetBalanceHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetScriptHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","blockId" : "67nh4SNi2oMhrda7ppKMKz8Z92SF22m9D1mcBLSJWPscb2GFDwXUC8Aih4BuJdFBP5Y8Mg143U44epMdP8eMzK34","dataEntryHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","height" : 100,"leaseBalanceHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","leaseStatusHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","sponsorshipHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","stateHash" : "303ae03f0eb9155f2c2352e6e8424d96b257aa3dfe3c4cb1f0827ad4f5d6ce29","version" : "Gowaves v0.0.0","wavesBalanceHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8"}`,
			"Gowaves v0.0.0",
			100,
		},
		{`{"accountScriptHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","aliasHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetBalanceHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetScriptHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","blockId" : "67nh4SNi2oMhrda7ppKMKz8Z92SF22m9D1mcBLSJWPscb2GFDwXUC8Aih4BuJdFBP5Y8Mg143U44epMdP8eMzK34","dataEntryHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","leaseBalanceHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","leaseStatusHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","sponsorshipHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","stateHash" : "303ae03f0eb9155f2c2352e6e8424d96b257aa3dfe3c4cb1f0827ad4f5d6ce29","wavesBalanceHash" : "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8"}`,
			"",
			0,
		},
	} {
		sh := new(StateHash)
		err := json.Unmarshal([]byte(test.js), sh)
		require.NoError(t, err)
		shd := new(StateHashDebug)
		err = json.Unmarshal([]byte(test.js), shd)
		require.NoError(t, err)
		assert.Equal(t, test.ver, shd.Version)
		assert.Equal(t, test.h, int(shd.Height))
		assert.Equal(t, sh, shd.GetStateHash())
	}
}
