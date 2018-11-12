package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
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
	assert.Equal(t, &CreateAliasV1{
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

var broadcastResp = `{
  "type": 10,
  "id": "5sXfATyK7xzfrG4AdFXnG3DMy6j3uEY3szZ21w5cGuNt",
  "sender": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "senderPublicKey": "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw",
  "fee": 100000,
  "timestamp": 1540301356669,
  "signature": "4kHdmZpXkCdnquXgEaVXiDpiiibxgd5uPxzmGnFTeD8CZDeHdDik5HEwduNG6WYGLJakd7ZrDMehmKsP8MaMGyqE",
  "proofs": [
    "4kHdmZpXkCdnquXgEaVXiDpiiibxgd5uPxzmGnFTeD8CZDeHdDik5HEwduNG6WYGLJakd7ZrDMehmKsP8MaMGyqE"
  ],
  "version": 1,
  "alias": "1234567"
}`

func TestAlias_Broadcast(t *testing.T) {
	pubKey, err := crypto.NewPublicKeyFromBase58("CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw")
	require.Nil(t, err)

	signature, err := crypto.NewSignatureFromBase58("4kHdmZpXkCdnquXgEaVXiDpiiibxgd5uPxzmGnFTeD8CZDeHdDik5HEwduNG6WYGLJakd7ZrDMehmKsP8MaMGyqE")
	require.Nil(t, err)

	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(broadcastResp, 200),
	})

	req := AliasBroadcastReq{
		SenderPublicKey: pubKey,
		Fee:             100000,
		Timestamp:       1540301356669,
		Signature:       signature,
		Alias:           "12345678",
	}

	body, resp, err :=
		client.Alias.Broadcast(context.Background(), req)
	require.Nil(t, err)
	assert.NotNil(t, resp)

	// response
	digest, err := crypto.NewDigestFromBase58("5sXfATyK7xzfrG4AdFXnG3DMy6j3uEY3szZ21w5cGuNt")
	require.Nil(t, err)

	assert.Equal(t, &CreateAliasV1{
		Type:      10,
		ID:        &digest,
		SenderPK:  pubKey,
		Timestamp: 1540301356669,
		Fee:       100000,
		Signature: &signature,
		Alias:     "1234567",
		Version:   1,
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/alias/broadcast/create", resp.Request.URL.String())
}
