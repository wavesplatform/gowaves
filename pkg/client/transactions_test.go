package client

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestNewTransactions(t *testing.T) {
	assert.NotNil(t, NewTransactions(defaultOptions))
}

var unconfirmedSizeJson = `{"size": 4}`

func TestTransactions_UnconfirmedSize(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(unconfirmedSizeJson, 200),
		BaseUrl: "https://testnodes.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Transactions.UnconfirmedSize(context.Background())
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(4), body)
	assert.Equal(t, "https://testnodes.wavesnodes.com/transactions/unconfirmed/size", resp.Request.URL.String())
}

func TestGuessTransaction_Genesis(t *testing.T) {
	genesisJson := `    {
      "type": 1,
      "id": "2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8",
      "fee": 0,
      "timestamp": 1465742577614,
      "signature": "2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8",
      "recipient": "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ",
      "amount": 9999999500000000
    }`

	buf := bytes.NewBufferString(genesisJson)

	rs, err := UnmarshalTransaction(&TransactionTypeVersion{Type: proto.TransactionType(1), Version: 0}, buf)
	require.Nil(t, err)
	require.IsType(t, &proto.Genesis{}, rs)
	genesis := rs.(*proto.Genesis)
	assert.Equal(t, uint64(9999999500000000), genesis.Amount)
}

var transactionInfoPayment = `
{
  "type": 7,
  "id": "95DEg9uS9Ez2RoAQWsBgW8hDmHEJzWB1nMpPZdSp1JbB",
  "sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
  "senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
  "fee": 300000,
  "timestamp": 1533280536821,
  "signature": "PEf6JfPHDL79coKov7mx9BkwR56oiqpF97kJRS3tBqk4g99S1xA26vNvsJszqStG63gb7hdpmoEREBjEiTzUMR3",
  "order1": {
    "id": "D9DLD9FmorZgvcANhL5RjErraFQ9j3ENoH3mPnWCn3nk",
    "sender": "3PLY99toDyrQHTV795KK4cvcxQT31MUa57E",
    "senderPublicKey": "E5eYELyZbw8K5kZpA5DudGLfz6isFaeoUkBz28NmVFor",
    "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
    "assetPair": {
      "amountAsset": "BrjUWjndUanm5VsJkbUip8VRYy6LWJePtxya3FNv4TQa",
      "priceAsset": null
    },
    "orderType": "buy",
    "price": 9979999500,
    "amount": 21211,
    "timestamp": 1533280536804,
    "expiration": 1533280836804,
    "matcherFee": 300000,
    "signature": "2MZDfRq14EHncmRkLn34Zb4fQWebxrTG65VcMpqaEdCWnza5HSPBuKhSj3EfGbu9w3z44wDXRm8RzmzqhczTQQSB"
  },
  "order2": {
    "id": "4KArftCUnZ92J451NpdRRNHRnBGwYtAPfjsMVEhx1vq7",
    "sender": "3P7Rp9qp9qZYgGYtUiP7twR8MzESdqZ4Hsx",
    "senderPublicKey": "CHF8xe8tosuC11NR8hagyifkJGgfQkj9KTkViBXBQj6n",
    "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
    "assetPair": {
      "amountAsset": "BrjUWjndUanm5VsJkbUip8VRYy6LWJePtxya3FNv4TQa",
      "priceAsset": null
    },
    "orderType": "sell",
    "price": 9979999500,
    "amount": 29933,
    "timestamp": 1533280536760,
    "expiration": 1533280836760,
    "matcherFee": 300000,
    "signature": "2UrnTSbQeaGvjtYvgwH2QgpX1pFgDTARpyvE3cKT4CvD7scGHFpji69ieT4gJb2o5Mmt7WDpxSQ9bYnpD7AmnG7j"
  },
  "price": 9979999500,
  "amount": 21211,
  "buyMatcherFee": 300000,
  "sellMatcherFee": 212584,
  "height": 1110500
}
`

func TestTransactions_Info(t *testing.T) {
	id, _ := crypto.NewDigestFromBase58("95DEg9uS9Ez2RoAQWsBgW8hDmHEJzWB1nMpPZdSp1JbB")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(transactionInfoPayment, 200),
		BaseUrl: "https://testnodes.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Transactions.Info(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, &id, body.(*proto.ExchangeV1).ID)
	assert.Equal(t, "https://testnodes.wavesnodes.com/transactions/info/95DEg9uS9Ez2RoAQWsBgW8hDmHEJzWB1nMpPZdSp1JbB", resp.Request.URL.String())
}
