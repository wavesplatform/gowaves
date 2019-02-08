package proto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"strings"
	"testing"
	"time"
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
		if o, err := NewUnsignedOrderV1(spk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, tc.timestamp, tc.expiration, tc.fee); assert.NoError(t, err) {
			if b, err := o.bodyMarshalBinary(); assert.NoError(t, err) {
				d, _ := crypto.FastHash(b)
				assert.Equal(t, id, d)
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}

	}
}

func TestOrderV1Validations(t *testing.T) {
	tests := []struct {
		price  uint64
		amount uint64
		fee    uint64
		err    string
	}{
		{0, 20, 30, "failed to create OrderV1: price should be positive"},
		{10, 0, 30, "failed to create OrderV1: amount should be positive"},
		{10, 20, 0, "failed to create OrderV1: matcher's fee should be positive"},
	}
	spk, _ := crypto.NewPublicKeyFromBase58("6s3F3S1ZmdJ2B25EqHWgNUSfeHtMaRZJ4RGEB5hgS7QM")
	mpk, _ := crypto.NewPublicKeyFromBase58("7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy")
	aa, _ := NewOptionalAssetFromString("8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS")
	pa, _ := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	for _, tc := range tests {
		_, err := NewUnsignedOrderV1(spk, mpk, *aa, *pa, Buy, tc.price, tc.amount, 0, 0, tc.fee)
		assert.EqualError(t, err, tc.err)
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
		sk, pk := crypto.GenerateKeyPair(seed)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		if o, err := NewUnsignedOrderV1(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee); assert.NoError(t, err) {
			if err := o.Sign(sk); assert.NoError(t, err) {
				if r, err := o.Verify(pk); assert.NoError(t, err) {
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
			}
		}
	}
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
		sk, pk := crypto.GenerateKeyPair(seed)
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
		if o, err := NewUnsignedOrderV1(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee); assert.NoError(t, err) {
			if j, err := json.Marshal(o); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
					base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
				assert.Equal(t, ej, string(j))
				if err := o.Sign(sk); assert.NoError(t, err) {
					if j, err := json.Marshal(o); assert.NoError(t, err) {
						ej := fmt.Sprintf("{\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
							base58.Encode(o.ID[:]), base58.Encode(o.Signature[:]), base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
						assert.Equal(t, ej, string(j))
					}
				}
			}
		}
	}
}

func TestOrderV2Validations(t *testing.T) {
	tests := []struct {
		price  uint64
		amount uint64
		fee    uint64
		err    string
	}{
		{0, 20, 30, "failed to create OrderV2: price should be positive"},
		{10, 0, 30, "failed to create OrderV2: amount should be positive"},
		{10, 20, 0, "failed to create OrderV2: matcher's fee should be positive"},
	}
	spk, _ := crypto.NewPublicKeyFromBase58("6s3F3S1ZmdJ2B25EqHWgNUSfeHtMaRZJ4RGEB5hgS7QM")
	mpk, _ := crypto.NewPublicKeyFromBase58("7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy")
	aa, _ := NewOptionalAssetFromString("8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS")
	pa, _ := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	for _, tc := range tests {
		_, err := NewUnsignedOrderV2(spk, mpk, *aa, *pa, Buy, tc.price, tc.amount, 0, 0, tc.fee)
		assert.EqualError(t, err, tc.err)
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
		sk, pk := crypto.GenerateKeyPair(seed)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		if o, err := NewUnsignedOrderV2(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee); assert.NoError(t, err) {
			if err := o.Sign(sk); assert.NoError(t, err) {
				if r, err := o.Verify(pk); assert.NoError(t, err) {
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
		sk, pk := crypto.GenerateKeyPair(seed)
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
		if o, err := NewUnsignedOrderV2(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee); assert.NoError(t, err) {
			if j, err := json.Marshal(o); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"version\":2,\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
					base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
				assert.Equal(t, ej, string(j))
				if err := o.Sign(sk); assert.NoError(t, err) {
					if j, err := json.Marshal(o); assert.NoError(t, err) {
						ej := fmt.Sprintf("{\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"matcherPublicKey\":\"%s\",\"assetPair\":{\"amountAsset\":%s,\"priceAsset\":%s},\"orderType\":\"%s\",\"price\":%d,\"amount\":%d,\"timestamp\":%d,\"expiration\":%d,\"matcherFee\":%d}",
							base58.Encode(o.ID[:]), base58.Encode(o.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.matcher, aas, pas, tc.orderType.String(), tc.price, tc.amount, ts, exp, tc.fee)
						assert.Equal(t, ej, string(j))
					}
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
				assert.Equal(t, Integer, av.GetValueType())
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
				assert.Equal(t, Integer, av.GetValueType())
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
				assert.Equal(t, Boolean, av.GetValueType())
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
				assert.Equal(t, Boolean, av.GetValueType())
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
		{"empty value", ""},
		{"", ""},
	}
	for _, tc := range tests {
		bv, _ := base58.Decode(tc.value)
		v := BinaryDataEntry{tc.key, bv}
		if b, err := v.MarshalBinary(); assert.NoError(t, err) {
			var av BinaryDataEntry
			if err := av.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, tc.key, av.Key)
				assert.Equal(t, tc.key, av.GetKey())
				assert.ElementsMatch(t, bv, av.Value)
				assert.Equal(t, Binary, av.GetValueType())
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
		{"empty value", ""},
		{"", ""},
	}
	for _, tc := range tests {
		bv, _ := base58.Decode(tc.value)
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
				assert.Equal(t, Binary, av.GetValueType())
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
				assert.Equal(t, String, av.GetValueType())
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
				assert.Equal(t, String, av.GetValueType())
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
		var entries DataEntries
		if err := entries.UnmarshalJSON([]byte(tc.json)); assert.NoError(t, err) {
			if b, err := json.Marshal(entries); assert.NoError(t, err) {
				assert.Equal(t, tc.json, string(b))
			}
			assert.ElementsMatch(t, tc.expected, entries)
		}
	}
}

func TestNewAttachmentFromBase58(t *testing.T) {
	att, err := NewAttachmentFromBase58("t")
	require.NoError(t, err)
	assert.Equal(t, att, Attachment("3"))
}

func TestAttachment_UnmarshalJSON(t *testing.T) {
	a := Attachment("")
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

	asset1, err := NewOptionalAssetFromDigest(d)
	require.NoError(t, err)
	assert.True(t, asset1.Present)
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
