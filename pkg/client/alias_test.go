package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestNewAlias(t *testing.T) {
	assert.NotNil(t, NewAlias(defaultOptions))
}

var aliasGetJson = `
{
  "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8"
}
`

func TestAlias_Get(t *testing.T) {
	addr, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(aliasGetJson, 200),
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Alias.Get(context.Background(), "frozen")
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, addr, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/alias/by-alias/frozen", resp.Request.URL.String())
}

var aliasGetByAddressJson = `
["alias:T:v3g4n"]
`

func TestAlias_GetByAddress(t *testing.T) {
	address, err := proto.NewAddressFromString("3MvxxSiCqCA5gvtp3EgjEukpWg4HccdLAMh")
	require.Nil(t, err)
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(aliasGetByAddressJson, 200),
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Alias.GetByAddress(context.Background(), address)
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.Equal(t, "https://testnode1.wavesnodes.com/alias/by-address/3MvxxSiCqCA5gvtp3EgjEukpWg4HccdLAMh", resp.Request.URL.String())
}

var aliasCreateResp = `{
  "type": 10,
  "id": "HTsPwyE6tYwq9MpRb6KbjMvcpNAD5reDVMyYCqoQhbMg",
  "sender": "3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp",
  "senderPublicKey": "J26nL27BBmTgCRye1MdzkFduFDE2aA4agCcuJUyDR2sZ",
  "fee": 100000,
  "timestamp": 1540201826158,
  "signature": "AP7zeZyVReHkiXfqdMQt7ot9A1KujkxoerQteCQ6Fa9gsUWm1qDd4Rc5CzXQuYEeHwaD7UMbP3vzAd58ZdxNMxJ",
  "proofs": [
    "AP7zeZyVReHkiXfqdMQt7ot9A1KujkxoerQteCQ6Fa9gsUWm1qDd4Rc5CzXQuYEeHwaD7UMbP3vzAd58ZdxNMxJ"
  ],
  "version": 1,
  "alias": "wave"
}`

func TestAlias_Create(t *testing.T) {
	address, err := proto.NewAddressFromString("3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp")
	require.Nil(t, err)

	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  "asfasf",
		Client:  NewMockHttpRequestFromString(aliasCreateResp, 200),
	})
	require.NoError(t, err)
	req := AliasCreateReq{
		Sender: address,
		Alias:  "wave",
		Fee:    100000,
	}
	body, resp, err :=
		client.Alias.Create(context.Background(), req)
	require.Nil(t, err)

	// response
	digest, err := crypto.NewDigestFromBase58("HTsPwyE6tYwq9MpRb6KbjMvcpNAD5reDVMyYCqoQhbMg")
	require.Nil(t, err)

	pk, err := crypto.NewPublicKeyFromBase58("J26nL27BBmTgCRye1MdzkFduFDE2aA4agCcuJUyDR2sZ")
	require.Nil(t, err)

	signature, err := crypto.NewSignatureFromBase58("AP7zeZyVReHkiXfqdMQt7ot9A1KujkxoerQteCQ6Fa9gsUWm1qDd4Rc5CzXQuYEeHwaD7UMbP3vzAd58ZdxNMxJ")
	require.Nil(t, err)

	assert.NotNil(t, resp)
	assert.Equal(t, &CreateAliasWithSig{
		Type:      10,
		ID:        &digest,
		SenderPK:  pk,
		Timestamp: 1540201826158,
		Fee:       100000,
		Signature: &signature,
		Alias:     "wave",
		Version:   1,
	}, body)

	assert.NotEmpty(t, resp.Request.Header.Get("X-API-Key"))
	assert.Equal(t, "https://testnode1.wavesnodes.com/alias/create", resp.Request.URL.String())
}
