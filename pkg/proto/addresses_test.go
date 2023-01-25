package proto

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
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
			al := NewAlias(tc.scheme, tc.alias)
			r = NewRecipientFromAlias(*al)
		default:
			assert.Fail(t, "incorrect test")
		}
		if js, err := json.Marshal(r); assert.NoError(t, err) {
			r2 := &Recipient{}
			err := json.Unmarshal(js, r2)
			assert.NoError(t, err)
			assert.Equal(t, r.BinarySize(), r2.BinarySize())
			assert.Equal(t, r.Alias(), r2.Alias())
			assert.Equal(t, r.Address(), r2.Address())
		}
	}
}

func TestRecipient_EqAddr(t *testing.T) {
	tests := []struct {
		rcp  Recipient
		addr WavesAddress
		res  bool
		err  string
	}{
		{NewRecipientFromAddress(WavesAddress{1, 1, 1}), WavesAddress{1, 1, 1}, true, ""},
		{NewRecipientFromAddress(WavesAddress{1, 1, 1}), WavesAddress{1, 2, 3}, false, ""},
		{
			NewRecipientFromAlias(*NewAlias(TestNetScheme, "blah")), WavesAddress{1, 2, 3}, false,
			"failed to compare recipient 'alias:T:blah' with addr '2npUV4bHHS5G9aTk5Wp5A2Exyh27UVz4GRm'",
		},
	}
	for i, tc := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			res, err := tc.rcp.EqAddr(tc.addr)
			if err != nil {
				assert.EqualError(t, err, tc.err)
			}
			assert.Equal(t, tc.res, res)
		})
	}
}

func TestRecipient_EqAlias(t *testing.T) {
	tests := []struct {
		rcp   Recipient
		alias Alias
		res   bool
		err   string
	}{

		{NewRecipientFromAlias(*NewAlias(TestNetScheme, "blah")), *NewAlias(TestNetScheme, "blah"), true, ""},
		{NewRecipientFromAlias(*NewAlias(TestNetScheme, "blah")), *NewAlias(TestNetScheme, "foo"), false, ""},
		{
			NewRecipientFromAddress(WavesAddress{1, 1, 1}), *NewAlias(TestNetScheme, "blah"), false,
			"failed to compare recipient '2nQxJj6jMtYKshwRWMgKDbXCXBw6Tji6sZZ' with alias 'alias:T:blah'",
		},
	}
	for i, tc := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			res, err := tc.rcp.EqAlias(tc.alias)
			if err != nil {
				assert.EqualError(t, err, tc.err)
			}
			assert.Equal(t, tc.res, res)
		})
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
	var addr WavesAddress
	seed := make([]byte, 32)
	_, _ = rand.Read(seed)
	_, pk, err := crypto.GenerateKeyPair(seed)
	if err != nil {
		b.Fatalf("crypto.GenerateKeyPair(): %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		addr, err = NewAddressFromPublicKey(TestNetScheme, pk)
	}
	b.StopTimer()

	if err != nil {
		b.Fatal(err.Error())
	}
	if MustAddressFromPublicKey(TestNetScheme, pk) != addr {
		b.Fatal("different addresses")
	}
}

func TestAliasFromString(t *testing.T) {
	const (
		alias      = "blah-blah-blah"
		aliasBytes = "6bqk2heWpAcsmshUhfT3QNEB"
	)
	a := NewAlias(TestNetScheme, alias)
	assert.Equal(t, "alias:T:blah-blah-blah", a.String())
	assert.Equal(t, alias, a.Alias)
	if b, err := a.MarshalBinary(); assert.NoError(t, err) {
		assert.Equal(t, aliasBytes, base58.Encode(b))
	}

	buf := &bytes.Buffer{}
	if _, err := a.WriteTo(buf); assert.NoError(t, err) {
		require.Equal(t, aliasBytes, base58.Encode(buf.Bytes()))
	}
}

func TestAlias_Valid(t *testing.T) {
	const okScheme = TestNetScheme
	for _, test := range []struct {
		alias  string
		scheme Scheme
		valid  bool
	}{
		{"alias", okScheme, true},
		{"qwerty", okScheme, true},
		{"correct", okScheme, true},
		{"xxx", okScheme, false},
		{"valid", 'I', false},
		{"xxxl-very-very-very-long-alias-that-is-incorrect", okScheme, false},
		{"asd=asd", okScheme, false},
		{"QazWsxEdc", okScheme, false},
	} {
		a := NewAlias(okScheme, test.alias)
		ok, err := a.Valid(test.scheme)
		if test.valid {
			assert.True(t, ok)
			assert.NoError(t, err)
		} else {
			assert.False(t, ok)
			assert.Error(t, err)
		}
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

func TestRecipient_WriteTo(t *testing.T) {
	buf := &bytes.Buffer{}

	addr, _ := NewAddressFromString("3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v")
	rec := NewRecipientFromAddress(addr)
	_, err := rec.WriteTo(buf)
	require.NoError(t, err)
	bin, err := rec.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, bin, buf.Bytes())

	buf.Reset()

	alias, _ := NewAliasFromString("alias:T:blah-blah-blah")
	rec = NewRecipientFromAlias(*alias)
	bin, err = rec.MarshalBinary()
	require.NoError(t, err)
	_, err = rec.WriteTo(buf)
	require.NoError(t, err)
	require.Equal(t, bin, buf.Bytes())

}

func TestEthABIEthAddressEqualsProtoEthAddress(t *testing.T) {
	require.Equal(t, EthereumAddressSize, ethabi.EthereumAddressSize)
}
