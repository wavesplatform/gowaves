package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var leasingActiveJson = `
[
  {
    "type": 8,
    "id": "B4hoL2R8SoWtmkbqjtDykSu3vZiuGkn4G7yYv5xkH4qs",
    "sender": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
    "senderPublicKey": "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw",
    "fee": 500175,
    "timestamp": 1533489396481,
    "signature": "3bq7aQUG3A7LN9uKvyieA8BesDFDer6gPv9rHwCeKF1MsgG8PJHtADd3PbLBqwRheTx4aTvxv2iGRTWgjCQjxPYL",
    "proofs": [
      "3bq7aQUG3A7LN9uKvyieA8BesDFDer6gPv9rHwCeKF1MsgG8PJHtADd3PbLBqwRheTx4aTvxv2iGRTWgjCQjxPYL"
    ],
    "version": 1,
    "amount": 1,
    "recipient": "3N5GRqzDBhjVXnCn44baHcz2GoZy5qLxtTh",
    "height": 342137
  }
]`

func TestNewLeasing(t *testing.T) {
	a := NewAssets(defaultOptions)
	assert.NotNil(t, a)
}

func TestLeasing_Active(t *testing.T) {
	addr, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(leasingActiveJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com/",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Leasing.Active(context.Background(), addr)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.EqualValues(t, proto.LeaseTransaction, body[0].Type)
	assert.Equal(t, "https://testnode1.wavesnodes.com/leasing/active/3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8", resp.Request.URL.String())
}
