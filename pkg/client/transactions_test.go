package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

var transactionInfoExchange = `
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
		Client:  NewMockHttpRequestFromString(transactionInfoExchange, 200),
		BaseUrl: "https://testnodes.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Transactions.Info(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, &id, body.(*ExchangeWithSigTransactionInfo).ID)
	assert.Equal(t, "https://testnodes.wavesnodes.com/transactions/info/95DEg9uS9Ez2RoAQWsBgW8hDmHEJzWB1nMpPZdSp1JbB", resp.Request.URL.String())
}

var dataTransaction = `
{
	"type": 12,
	"id": "74r5tx5BuhnYP3YQ5jo3RwDcH89gaDEdEc9bjUKPiSa8",
	"sender": "3P9QNCmT3Q44zRYXBwKN3azBta9azGqrscm",
	"senderPublicKey": "J48ygzZLEdcR2GbWjjy9eFJDs57Poz6ZajGEyygSMV26",
	"fee": 10000000,
	"timestamp": 1548739929686,
	"proofs": [
		"2bB5ysJXYBumJiLMbQ3o2gqxES5gydQ4bni3aWGiXwBaBDvLEpDNFLgKuj6UnhtS4LUS9R6yVoSVFoT94RCBvzo",
		"3PPgSrFX52vYbAtTVrz8nHjmcv3LQhYd3mP"
	],
	"version": 1,
	"data": [
		{
			"key": "lastPayment",
			"type": "string",
			"value": "GenCSKr8UFrZXrbQ8oAG7W8PDgUY7pe7hrbRmJACuMkS"
		},
		{
			"key": "heightToGetMoney",
			"type": "integer",
			"value": 1372374
		},
		{
			"key": "GenCSKr8UFrZXrbQ8oAG7W8PDgUY7pe7hrbRmJACuMkS",
			"type": "string",
			"value": "used"
		}
	]
}
`

func TestTransactionInfoDataTransaction(t *testing.T) {
	id, _ := crypto.NewDigestFromBase58("74r5tx5BuhnYP3YQ5jo3RwDcH89gaDEdEc9bjUKPiSa8")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(dataTransaction, 200),
		BaseUrl: "https://testnodes.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Transactions.Info(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, &id, body.(*DataTransactionInfo).ID)
	assert.Equal(t, "https://testnodes.wavesnodes.com/transactions/info/74r5tx5BuhnYP3YQ5jo3RwDcH89gaDEdEc9bjUKPiSa8", resp.Request.URL.String())

}

var transactionsUnconfirmedInfoJson = `{
  "type" : 7,
  "id" : "DCLe9jiCYmMBeU2qMwz2SU3bnWRJ9YdCgf3gEpzeGM52",
  "sender" : "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
  "senderPublicKey" : "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
  "fee" : 300000,
  "timestamp" : 1540808498538,
  "signature" : "2hafgo3bLG2WPoWSHt9Z383o84azU4E8X5KpNctZNb3iFsYEEb5eGTbXN9XKWUJSkdufCbAbovSmTn8Mc8hj72Y9",
  "order1" : {
    "id" : "BaDtJVWgaU9uoS6B1agvQefpDDeWnxduJrkBuJNFE2pY",
    "sender" : "3PKe8Y2oyGHJ9z7aooGx3wtdgNXGyzaDB4Y",
    "senderPublicKey" : "8qfeDYoqEQvp8ukUefRV3fCeBHMQZhi1pUn7KUw56hAr",
    "matcherPublicKey" : "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
    "assetPair" : {
      "amountAsset" : null,
      "priceAsset" : "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck"
    },
    "orderType" : "buy",
    "price" : 193,
    "amount" : 10000000000,
    "timestamp" : 1540807822557,
    "expiration" : 1543313422557,
    "matcherFee" : 300000,
    "signature" : "3EZKLzmPcxqXjczWCwzt6bS78BG1Pd8Boi2jphBFGDqkMTm2owcvXbsowpJU1oMR1QJcSyqCMuvfumyQBFc9TgjY"
  },
  "order2" : {
    "id" : "7HMfCo6ixbQjZ945gKig23DP1r3xYdHmCibF3MHs7tPv",
    "sender" : "3P72phQTwwiwc7QcNGoKA4wpUsqCuT5eMJ3",
    "senderPublicKey" : "4gmFnncST7SjHjTpys78uCmUsMCZ8kudAmvjvqpVuYKt",
    "matcherPublicKey" : "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
    "assetPair" : {
      "amountAsset" : null,
      "priceAsset" : "Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck"
    },
    "orderType" : "sell",
    "price" : 193,
    "amount" : 8712566,
    "timestamp" : 1540808498535,
    "expiration" : 1543400498535,
    "matcherFee" : 300000,
    "signature" : "3YVvNN8kPvB127ZPoMjn9ueToB3UFf5Lrbma8HqKEqo7Mb6ABaSg33QofkPXvCf6ehFTRU4cNNunKQqTeHK75Xv9"
  },
  "price" : 193,
  "amount" : 8290156,
  "buyMatcherFee" : 248,
  "sellMatcherFee" : 285455
}`

func TestTransactions_UnconfirmedInfo(t *testing.T) {
	id, _ := crypto.NewDigestFromBase58("DCLe9jiCYmMBeU2qMwz2SU3bnWRJ9YdCgf3gEpzeGM52")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(transactionsUnconfirmedInfoJson, 200),
		BaseUrl: "https://testnodes.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Transactions.UnconfirmedInfo(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, &id, body.(*proto.ExchangeWithSig).ID)
	assert.Equal(t, "https://testnodes.wavesnodes.com/transactions/unconfirmed/info/DCLe9jiCYmMBeU2qMwz2SU3bnWRJ9YdCgf3gEpzeGM52", resp.Request.URL.String())
}

var transactionsByAddressJson = `
[
  [
    {
      "type": 7,
      "id": "9ECrQ5oo3A6s4qtncRC5BCymWdVGYTV94YcSAEnbgQAj",
      "sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
      "senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
      "fee": 300000,
      "timestamp": 1540894037536,
      "signature": "2sQLYgsjy3UnsbF16QpxtYafsduwx2wG9EniN5wxoKtqXo94fhDfoS7JubUwyj5Z6FskMhMZ8gxDgdwWF8jyUN75",
      "order1": {
        "id": "2LtVMBzxbjoX9fRNXJekjnfdwWBZGacudzD2q2ydL1X5",
        "sender": "3PF3sfmNfcys9yBnmtAMJnWXnaDJy6DFb5g",
        "senderPublicKey": "C8A3yxGDnUazYVhj3VXQKhLHDshgXCXg8fChGNGPQgGw",
        "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
        "assetPair": {
          "amountAsset": "zMFqXuoyrn5w17PFurTqxB7GsS71fp9dfk6XFwxbPCy",
          "priceAsset": null
        },
        "orderType": "buy",
        "price": 23354852576,
        "amount": 13553323,
        "timestamp": 1540894017293,
        "expiration": 1540894317293,
        "matcherFee": 300000,
        "signature": "4jD8v1M57fqg3fPpsuru9Vwm254QrtFmyiKkxF7Y3PstXjwWSwBy1jQMisbnbAEZ42XebxiBtLGyFSb16wXtKVNU"
      },
      "order2": {
        "id": "RbhFTZWEufGTcuMsRLGz3xrtEGrLgC5pdukcERMRgyU",
        "sender": "3PMhuLKgEf2Dt1Gvc4nvEHsTpB1ywJisXhJ",
        "senderPublicKey": "2nwZ5Cn2EAzDr6hk5K4paRBCnK1e73TPWbhRqiqEkGwZ",
        "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
        "assetPair": {
          "amountAsset": "zMFqXuoyrn5w17PFurTqxB7GsS71fp9dfk6XFwxbPCy",
          "priceAsset": null
        },
        "orderType": "sell",
        "price": 23354852576,
        "amount": 6051000,
        "timestamp": 1540894035476,
        "expiration": 1541498835476,
        "matcherFee": 300000,
        "signature": "5rskUz2ki6Ma7WxpidJrvSvhqvKrifpFGtKQNBZtd6LJNCyebGc1sgaw5UAaQ7qaTYLV9wGLsL7qpRvG34DAz2MW"
      },
      "price": 23354852576,
      "amount": 6051000,
      "buyMatcherFee": 133937,
      "sellMatcherFee": 300000,
      "height": 1239417
    }
  ]
]`

func TestTransactions_TransactionsByAddress(t *testing.T) {
	id, _ := proto.NewAddressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(transactionsByAddressJson, 200),
		BaseUrl: "https://testnodes.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Transactions.Address(context.Background(), id, 1)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.Equal(t, "https://testnodes.wavesnodes.com/transactions/address/3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3/limit/1", resp.Request.URL.String())
	assert.Equal(t, uint64(300000), body[0].(*proto.ExchangeWithSig).Fee)
}

var transactionUnconfirmedJson = `
[ {
  "type" : 7,
  "id" : "7xkBdbJ7otS77oknAY3thHRo7ni4tavj5pLwNLgBmWsN",
  "sender" : "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
  "senderPublicKey" : "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
  "fee" : 300000,
  "timestamp" : 1540904586992,
  "signature" : "2ppPnJ864Trmb5B5ipv7Z8UCLv9mURyRQjQxJMPXcxThQ4PA5S8cx2xmvaWvFyVB28xBawnHUsvz1H8Fn7iDGjZa",
  "order1" : {
    "id" : "CPSbv9SHEV14osZjpt2YtL8GCLtSVxFaainsq1AER87W",
    "sender" : "3PMJiuaHbydtuDMGenEtHWBNMh3enAsd4ts",
    "senderPublicKey" : "GwEEVXTrxHGisyCsa2HZsqMsbpDoxJFZ4yvVhJCDnzVH",
    "matcherPublicKey" : "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
    "assetPair" : {
      "amountAsset" : "2g8GAuPB9cumz69462ySyhPCU1kAbQ5XcQ2UaHFdsBap",
      "priceAsset" : null
    },
    "orderType" : "buy",
    "price" : 50000000000000,
    "amount" : 1,
    "timestamp" : 1540904593767,
    "expiration" : 1543410193767,
    "matcherFee" : 300000,
    "signature" : "1nXS3at3pAs7DDgYWb9VEpHhYVbfnZFmnEDA9A9FDRe3F3Yy9hixw8e88oPdp2iM5b1grQYqXGFhRHjZBehmwLZ"
  },
  "order2" : {
    "id" : "JeWjU8eD7bo2w4engMAWxjhZHZzuiiaiUZBLhAE1ih9",
    "sender" : "3PMJiuaHbydtuDMGenEtHWBNMh3enAsd4ts",
    "senderPublicKey" : "GwEEVXTrxHGisyCsa2HZsqMsbpDoxJFZ4yvVhJCDnzVH",
    "matcherPublicKey" : "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
    "assetPair" : {
      "amountAsset" : "2g8GAuPB9cumz69462ySyhPCU1kAbQ5XcQ2UaHFdsBap",
      "priceAsset" : null
    },
    "orderType" : "sell",
    "price" : 50000000000000,
    "amount" : 1,
    "timestamp" : 1540904546163,
    "expiration" : 1543410146163,
    "matcherFee" : 300000,
    "signature" : "4Lr6xGb2eUQYo2iiE3njSPBu6EqF9vmKmMwo71iCVE3greasa51MATeqm7JGS3dD3KrjziC5xTrVX7jez4ppNPcf"
  },
  "price" : 50000000000000,
  "amount" : 1,
  "buyMatcherFee" : 300000,
  "sellMatcherFee" : 300000
}]`

func TestTransactions_Unconfirmed(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(transactionUnconfirmedJson, 200),
		BaseUrl: "https://testnodes.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Transactions.Unconfirmed(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.Equal(t, "https://testnodes.wavesnodes.com/transactions/unconfirmed", resp.Request.URL.String())
	assert.Equal(t, uint64(300000), body[0].(*proto.ExchangeWithSig).Fee)
}
