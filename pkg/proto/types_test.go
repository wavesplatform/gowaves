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

func TestOrderSigningRoundTrip(t *testing.T) {
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
		if o, err := NewUnsignedOrder(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee); assert.NoError(t, err) {
			if err := o.Sign(sk); assert.NoError(t, err) {
				if r, err := o.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
				if b, err := o.MarshalBinary(); assert.NoError(t, err) {
					var ao Order
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

func TestOrderToJSON(t *testing.T) {
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
		if o, err := NewUnsignedOrder(pk, mpk, *aa, *pa, tc.orderType, tc.price, tc.amount, ts, exp, tc.fee); assert.NoError(t, err) {
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
			var s string
			if len(bv) == 0 {
				s = "null"
			} else {
				s = fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString(bv))
			}
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

func TestNewAttachmentFromBase58(t *testing.T) {
	att, err := NewAttachmentFromBase58("t")
	require.NoError(t, err)
	assert.Equal(t, att, Attachment("3"))
}
