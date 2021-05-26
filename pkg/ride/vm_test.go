package ride

import (
	"encoding/json"
	"time"

	"github.com/mr-tron/base58"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

//go:generate moq -pkg ride -out types_moq_test.go ../types SmartState:MockSmartState

func newTransferTransaction() *proto.TransferWithProofs {
	js := `{"type":4,"version":2,"id":"CqjGMbrd5bFmLAv2mUSdphEJSgVWkWa6ZtcMkKmgH2ax","proofs":["5W7hjPpgmmhxevCt4A7y9F8oNJ4V9w2g8jhQgx2qGmBTNsP1p1MpQeKF3cvZULwJ7vQthZfSx2BhL6TWkHSVLzvq"],"senderPublicKey":"14ovLL9a6xbBfftyxGNLKMdbnzGgnaFQjmgUJGdho6nY","assetId":null,"feeAssetId":null,"timestamp":1544715621,"amount":15,"fee":10000,"recipient":"3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"}`
	tv2 := &proto.TransferWithProofs{}
	err := json.Unmarshal([]byte(js), tv2)
	if err != nil {
		panic(err)
	}
	return tv2
}

func newExchangeTransaction() *proto.ExchangeWithProofs {
	js := `{"senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy","amount": 100000000,"fee": 1100000,"type": 7,"version": 2,"sellMatcherFee": 1100000,"sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3","feeAssetId": null,"proofs": ["DGxkASjpPaKxu8bAv3PJpF9hJ9KAiLsB7bLBTEZXYcWmmc65pHiq5ymJNAazRM2aoLCeTLXXNda5hR9LZNayB69"],"price": 790000,  "id": "5aHKTDvWdVWmo9MPDPoYX83x6hyLJ5ji4eopmoUxELR2",  "order2": {    "version": 2,    "id": "CzBrJkpaWz2AHnT3U8baY3eTfRdymuC7dEqiGpas68tD",    "sender": "3PEjQH31dP2ipvrkouUs12ynKShpBcRQFAT",    "senderPublicKey": "BVtDAjf1MmUdPW2yRHEBiSP5yy7EnxzKgQWpajQM8FCx",    "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",    "assetPair": {      "amountAsset": "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF",      "priceAsset": "CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr"    },    "orderType": "sell",    "amount": 100000000,    "price": 790000,    "timestamp": 1557995955609,    "expiration": 1560501555609,    "matcherFee": 1100000,    "signature": "3Aw94WkF4PUeard435jtJTZLESRFMBuxYRYVVf3GrG48aAxLhbvcXdwsrtALLQ3LYbdNdhR1NUUzdMinU8pLiwWc",    "proofs": [      "3Aw94WkF4PUeard435jtJTZLESRFMBuxYRYVVf3GrG48aAxLhbvcXdwsrtALLQ3LYbdNdhR1NUUzdMinU8pLiwWc"    ]  },  "order1": {    "version": 2,    "id": "APLf7qDhU5puSa5h1KChNBobF8VKoy37PcP7BnhoSPvi",    "sender": "3PEyLyxu4yGJAEmuVRy3G4FvEBUYV6ykQWF",    "senderPublicKey": "28sBbJ7pHNG4VFrvNN43sNsdWYyrTFVAwd98W892mxBQ",    "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",    "assetPair": {      "amountAsset": "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF",      "priceAsset": "CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr"    },    "orderType": "buy",    "amount": 100000000,    "price": 790000,    "timestamp": 1557995158094,    "expiration": 1560500758093,    "matcherFee": 1100000,    "signature": "5zUuSSJyv5NU11RPa91fpQaCXR3xvR1ctjQrfxnNREFhMmbXfACzhfFgV18rdvrvm4X3p3iYK3fxS1TXwgSV5m83",    "proofs": [      "5zUuSSJyv5NU11RPa91fpQaCXR3xvR1ctjQrfxnNREFhMmbXfACzhfFgV18rdvrvm4X3p3iYK3fxS1TXwgSV5m83"    ]  },  "buyMatcherFee": 1100000,  "timestamp": 1557995955923,  "height": 1528811}`
	tx := new(proto.ExchangeWithProofs)
	err := json.Unmarshal([]byte(js), tx)
	if err != nil {
		panic(err)
	}
	return tx
}

func newDataTransaction() *proto.DataWithProofs {
	sk, pk, err := crypto.GenerateKeyPair([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	if err != nil {
		panic(err)
	}
	tx := proto.NewUnsignedData(1, pk, 100000, 1568640015000)
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "integer", Value: 100500})
	tx.Entries = append(tx.Entries, &proto.BooleanDataEntry{Key: "boolean", Value: true})
	tx.Entries = append(tx.Entries, &proto.BinaryDataEntry{Key: "binary", Value: []byte{0xCA, 0xFE, 0xBE, 0xBE, 0xDE, 0xAD, 0xBE, 0xEF}})
	tx.Entries = append(tx.Entries, &proto.StringDataEntry{Key: "string", Value: "Hello, World!"})
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "someKey", Value: 12345})
	err = tx.Sign(proto.MainNetScheme, sk)
	if err != nil {
		panic(err)
	}
	return tx
}

func testTransferWithProofs() *proto.TransferWithProofs {
	var scheme byte = 'T'
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	if err != nil {
		panic(err)
	}
	sk, pk, err := crypto.GenerateKeyPair(seed)
	if err != nil {
		panic(err)
	}
	tm, err := time.Parse(time.RFC3339, "2020-10-01T00:00:00+00:00")
	if err != nil {
		panic(err)
	}
	ts := uint64(tm.UnixNano() / 1000000)
	addr, err := proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		panic(err)
	}
	rcp := proto.NewRecipientFromAddress(addr)
	att := []byte("some attachment")
	tx := proto.NewUnsignedTransferWithProofs(3, pk, proto.OptionalAsset{}, proto.OptionalAsset{}, ts, 1234500000000, 100000, rcp, att)
	err = tx.GenerateID(scheme)
	if err != nil {
		panic(err)
	}
	err = tx.Sign(scheme, sk)
	if err != nil {
		panic(err)
	}
	return tx
}

func testTransferObject() rideObject {
	obj, err := transferWithProofsToObject('T', byte_helpers.TransferWithProofs.Transaction)
	if err != nil {
		panic(err)
	}
	return obj
}

func testExchangeWithProofsToObject() rideObject {
	obj, err := exchangeWithProofsToObject('T', byte_helpers.ExchangeWithProofs.Transaction)
	if err != nil {
		panic(err.Error())
	}
	return obj
}
