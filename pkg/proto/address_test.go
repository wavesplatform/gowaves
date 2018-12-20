package proto

import (
	"crypto/rand"
	"encoding/json"
	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"testing"
)

func TestAddressFromPublicKey(t *testing.T) {
	tests := []struct {
		publicKey string
		scheme    byte
		address   string
	}{
		{"5CnGfSjguYfzWzaRmbxzCbF5qRNGTXEvayytSANkqQ6A", MainNetScheme, "3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v"},
		{"BstqhtQjQN9X78i6mEpaNnf6cMsZZRDVHNv3CqguXbxq", MainNetScheme, "3PQvBCHPnxXprTNq1rwdcDuxt6VGKRTM9wT"},
		{"FckK43s6tQ9BBW77hSKuyRnfnrKuf6B7sEuJzcgkSDVf", MainNetScheme, "3PETfqHg9HyL92nfiujN5fBW6Ac1TYiVAAc"},
		{"5CnGfSjguYfzWzaRmbxzCbF5qRNGTXEvayytSANkqQ6A", TestNetScheme, "3NC7nrggwhk2AbRC7kzv92yDjbVyALeGzE5"},
		{"BstqhtQjQN9X78i6mEpaNnf6cMsZZRDVHNv3CqguXbxq", TestNetScheme, "3NCuNExVvpzSE15QkngdemY9XCyVVGhHA9h"},
		{"5CnGfSjguYfzWzaRmbxzCbF5qRNGTXEvayytSANkqQ6A", 'x', "3cgHWJbRKGEhi32DEe6ucVV24FfF7u2mxit"},
		{"BstqhtQjQN9X78i6mEpaNnf6cMsZZRDVHNv3CqguXbxq", 'x', "3ch55gsEJPV7mSgRsfnd8E3wqs8mSqyTNCj"},
	}
	for _, tc := range tests {
		if b, err := base58.Decode(tc.publicKey); assert.NoError(t, err) {
			var pk crypto.PublicKey
			copy(pk[:], b)
			if address, err := NewAddressFromPublicKey(tc.scheme, pk); assert.NoError(t, err) {
				assert.Equal(t, tc.address, address.String())
			}
		}
	}
}

func TestRecipientJSONRoundTrip(t *testing.T) {
	tests := []struct {
		publicKey string
		scheme    byte
		alias     string
	}{
		{"5CnGfSjguYfzWzaRmbxzCbF5qRNGTXEvayytSANkqQ6A", MainNetScheme, ""},
		{"BstqhtQjQN9X78i6mEpaNnf6cMsZZRDVHNv3CqguXbxq", TestNetScheme, ""},
		{"", MainNetScheme, "alias1"},
		{"", TestNetScheme, "alias2"},
	}
	for _, tc := range tests {
		var r Recipient
		switch {
		case tc.publicKey != "":
			if pk, err := crypto.NewPublicKeyFromBase58(tc.publicKey); assert.NoError(t, err) {
				if a, err := NewAddressFromPublicKey(tc.scheme, pk); assert.NoError(t, err) {
					r = NewRecipientFromAddress(a)
				}
			}
		case tc.alias != "":
			if al, err := NewAlias(tc.scheme, tc.alias); assert.NoError(t, err) {
				r = NewRecipientFromAlias(*al)
			}
		default:
			assert.Fail(t, "incorrect test")
		}
		if js, err := json.Marshal(r); assert.NoError(t, err) {
			r2 := &Recipient{}
			err := json.Unmarshal(js, r2)
			assert.NoError(t, err)
			assert.Equal(t, r.len, r2.len)
			assert.Equal(t, r.Alias, r2.Alias)
			assert.Equal(t, r.Address, r2.Address)
		}
	}
}

func TestAddressFromBytes(t *testing.T) {
	addresses := []string{
		"3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v",
		"3PQvBCHPnxXprTNq1rwdcDuxt6VGKRTM9wT",
		"3PETfqHg9HyL92nfiujN5fBW6Ac1TYiVAAc",
		"3NC7nrggwhk2AbRC7kzv92yDjbVyALeGzE5",
		"3NCuNExVvpzSE15QkngdemY9XCyVVGhHA9h",
		"3cgHWJbRKGEhi32DEe6ucVV24FfF7u2mxit",
		"3ch55gsEJPV7mSgRsfnd8E3wqs8mSqyTNCj"}
	for _, address := range addresses {
		if b, err := base58.Decode(address); assert.NoError(t, err) {
			if a, err := NewAddressFromBytes(b); assert.NoError(t, err) {
				assert.Equal(t, address, a.String())
			}
		}
	}
}

func BenchmarkNewWavesAddressFromPublicKey(b *testing.B) {
	seed := make([]byte, 32)
	_, _ = rand.Read(seed)
	_, pk := crypto.GenerateKeyPair(seed)
	for n := 0; n < b.N; n++ {
		_, _ = NewAddressFromPublicKey(MainNetScheme, pk)
	}
}

func TestAliasFromString(t *testing.T) {
	const (
		alias      = "blah-blah-blah"
		aliasBytes = "6bqk2heWpAcsmshUhfT3QNEB"
	)
	if a, err := NewAlias(TestNetScheme, alias); assert.NoError(t, err) {
		assert.Equal(t, "alias:T:blah-blah-blah", a.String())
		assert.Equal(t, alias, a.Alias)
		if b, err := a.MarshalBinary(); assert.NoError(t, err) {
			assert.Equal(t, aliasBytes, base58.Encode(b))
		}
	}
}

func TestIncorrectAlias(t *testing.T) {
	aliases := []string{"xxx", "xxxl-very-very-very-long-alias-that-is-incorrect", "asd=asd", "QazWsxEdc"}
	for _, alias := range aliases {
		_, err := NewAlias(MainNetScheme, alias)
		assert.Error(t, err)
	}
}

func TestAliasFromBytes(t *testing.T) {
	const (
		alias      = "blah-blah-blah"
		aliasBytes = "6bqk2heWpAcsmshUhfT3QNEB"
	)
	b, err := base58.Decode(aliasBytes)
	assert.Nil(t, err)
	a, err := NewAliasFromBytes(b)
	assert.Nil(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, aliasVersion, a.Version)
	assert.Equal(t, TestNetScheme, a.Scheme)
	assert.Equal(t, alias, a.Alias)
}
