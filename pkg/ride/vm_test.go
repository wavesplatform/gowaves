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

/*
func TestExecution(t *testing.T) {
	//state := &MockSmartState{NewestTransactionByIDFunc: func(_ []byte) (proto.Transaction, error) {
	//	return testTransferWithProofs(), nil
	//}}
	//env := &MockRideEnvironment{
	//	transactionFunc: testTransferObject,
	//	stateFunc: func() types.SmartState {
	//		return state
	//	},
	//	schemeFunc: func() byte {
	//		return 'T'
	//	},
	//}
	for _, test := range []struct {
		comment string
		source  string
		env     RideEnvironment
		res     bool
	}{
		//{`V1: true`, "AQa3b8tH", nil, true},
		//{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn", nil, true},
		//{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", nil, true},
		//{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=", nil, false},
		//{`V3: let i = 12345; let s = "12345"; toString(i) == s`, "AwQAAAABaQAAAAAAAAAwOQQAAAABcwIAAAAFMTIzNDUJAAAAAAAAAgkAAaQAAAABBQAAAAFpBQAAAAFz1B1iCw==", nil, true},
		//{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E", nil, true},
		//{`V3: if (false) then {let r = true; r} else {let r = false; r}`, "AwMHBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXI+tfo1", nil, false},
		//{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==", nil, true},
		{`V3: let x = 1; func add(i: Int) = i + 1; add(x) == 2`, "AwQAAAABeAAAAAAAAAAAAQoBAAAAA2FkZAAAAAEAAAABaQkAAGQAAAACBQAAAAFpAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAF4AAAAAAAAAAACfr6U6w==", nil, true},
		//{`V3: let b = base16'0000000000000001'; func add(b: ByteVector) = toInt(b) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAWIJAABkAAAAAgkABLEAAAABBQAAAAFiAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAACX00biA==", nil, true},
		//{`V3: let b = base16'0000000000000001'; func add(v: ByteVector) = toInt(v) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAXYJAABkAAAAAgkABLEAAAABBQAAAAF2AAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAACI7gYxg==", nil, true},
		//{`V3: let b = base16'0000000000000001'; func add(v: ByteVector) = toInt(b) + 1; add(b) == 2`, "AwQAAAABYgEAAAAIAAAAAAAAAAEKAQAAAANhZGQAAAABAAAAAXYJAABkAAAAAgkABLEAAAABBQAAAAFiAAAAAAAAAAABCQAAAAAAAAIJAQAAAANhZGQAAAABBQAAAAFiAAAAAAAAAAAChRvwnQ==", nil, true},
		//{`V3: let data = base64'AAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLw=='; func getStock(data:ByteVector) = toInt(take(drop(data, 8), 8)); getStock(data) == 1`, `AwQAAAAEZGF0YQEAAABwAAAAAAABhqAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWyt9GyysOW84u/u5V5Ah/SzLfef4c28UqXxowxFZS4SLiC6+XBh8D7aJDXyTTjpkPPED06ZPOzUE23V6VYCsLwoBAAAACGdldFN0b2NrAAAAAQAAAARkYXRhCQAEsQAAAAEJAADJAAAAAgkAAMoAAAACBQAAAARkYXRhAAAAAAAAAAAIAAAAAAAAAAAICQAAAAAAAAIJAQAAAAhnZXRTdG9jawAAAAEFAAAABGRhdGEAAAAAAAAAAAFCtabi`, nil, true},
		//{`V3: let ref = 999; func g(a: Int) = ref; func f(ref: Int) = g(ref); f(1) == 999`, "AwQAAAADcmVmAAAAAAAAAAPnCgEAAAABZwAAAAEAAAABYQUAAAADcmVmCgEAAAABZgAAAAEAAAADcmVmCQEAAAABZwAAAAEFAAAAA3JlZgkAAAAAAAACCQEAAAABZgAAAAEAAAAAAAAAAAEAAAAAAAAAA+fjknmW", nil, true},
		//{`let x = 5; 6 > 4`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGAAAAAAAAAAAEYSW6XA==`, nil, true},
		//{`let x = 5; 6 > x`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGBQAAAAF4Gh24hw==`, nil, true},
		//{`let x = 5; 6 >= x`, `AQQAAAABeAAAAAAAAAAABQkAAGcAAAACAAAAAAAAAAAGBQAAAAF4jlxXHA==`, nil, true},
		//{`false`, `AQfeYll6`, nil, false},
		//{`let x =  throw(); true`, `AQQAAAABeAkBAAAABXRocm93AAAAAAa7bgf4`, nil, true},
		//{`let x =  throw(); true || x`, `AQQAAAABeAkBAAAABXRocm93AAAAAAMGBgUAAAABeKRnLds=`, nil, true},
		//{`tx.id == base58''`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAAJBtD70=`, env, false},
		//{`tx.id == base58'H5C8bRzbUTMePSDVVxjiNKDUwk6CKzfZGTP2Rs7aCjsV'`, `BAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAIO7N5luRDUgN1SJ4kFmy/Ni8U2H6k7bpszok5tlLlRVgHwSHyg==`, env, true},
		//{`let x = tx.id == base58'a';true`, `AQQAAAABeAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAASEGjR0kcA==`, env, true},
		//{`tx.proofs[0] != base58'' && tx.proofs[1] == base58''`, `BAMJAQAAAAIhPQAAAAIJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAEAAAAACQAAAAAAAAIJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAQEAAAAAB106gzM=`, env, true},
		//{`match tx {case t : TransferTransaction | MassTransferTransaction | ExchangeTransaction => true; case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNFeGNoYW5nZVRyYW5zYWN0aW9uBgMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24GCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB6Ilvok=`, env, true},
		//{`V2: match transactionById(tx.id) {case  t: Unit => false case _ => true}`, `AgQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAAAXQFAAAAByRtYXRjaDAHBp9TFcQ=`, env, true},
		//{`Up() == UP`, `AwkAAAAAAAACCQEAAAACVXAAAAAABQAAAAJVUPGUxeg=`, nil, true},
		//{`HalfUp() == HALFUP`, `AwkAAAAAAAACCQEAAAAGSGFsZlVwAAAAAAUAAAAGSEFMRlVQbUfpTQ==`, nil, true},
		//{`let a0 = NoAlg() == NOALG; let a1 = Md5() == MD5; let a2 = Sha1() == SHA1; let a3 = Sha224() == SHA224; let a4 = Sha256() == SHA256; let a5 = Sha384() == SHA384; let a6 = Sha512() == SHA512; let a7 = Sha3224() == SHA3224; let a8 = Sha3256() == SHA3256; let a9 = Sha3384() == SHA3384; let a10 = Sha3512() == SHA3512; a0 && a1 && a2 && a3 && a4 && a5 && a6 && a7 && a8 && a9 && a10`, `AwQAAAACYTAJAAAAAAAAAgkBAAAABU5vQWxnAAAAAAUAAAAFTk9BTEcEAAAAAmExCQAAAAAAAAIJAQAAAANNZDUAAAAABQAAAANNRDUEAAAAAmEyCQAAAAAAAAIJAQAAAARTaGExAAAAAAUAAAAEU0hBMQQAAAACYTMJAAAAAAAAAgkBAAAABlNoYTIyNAAAAAAFAAAABlNIQTIyNAQAAAACYTQJAAAAAAAAAgkBAAAABlNoYTI1NgAAAAAFAAAABlNIQTI1NgQAAAACYTUJAAAAAAAAAgkBAAAABlNoYTM4NAAAAAAFAAAABlNIQTM4NAQAAAACYTYJAAAAAAAAAgkBAAAABlNoYTUxMgAAAAAFAAAABlNIQTUxMgQAAAACYTcJAAAAAAAAAgkBAAAAB1NoYTMyMjQAAAAABQAAAAdTSEEzMjI0BAAAAAJhOAkAAAAAAAACCQEAAAAHU2hhMzI1NgAAAAAFAAAAB1NIQTMyNTYEAAAAAmE5CQAAAAAAAAIJAQAAAAdTaGEzMzg0AAAAAAUAAAAHU0hBMzM4NAQAAAADYTEwCQAAAAAAAAIJAQAAAAdTaGEzNTEyAAAAAAUAAAAHU0hBMzUxMgMDAwMDAwMDAwMFAAAAAmEwBQAAAAJhMQcFAAAAAmEyBwUAAAACYTMHBQAAAAJhNAcFAAAAAmE1BwUAAAACYTYHBQAAAAJhNwcFAAAAAmE4BwUAAAACYTkHBQAAAANhMTAHRc/wAA==`, nil, true},
		//{`Unit() == unit`, `AwkAAAAAAAACCQEAAAAEVW5pdAAAAAAFAAAABHVuaXTstg1G`, nil, true},
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

		script, err := Compile(tree)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, script, test.comment)

		res, err := script.Run(test.env)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, res, test.comment)
		r, ok := res.(ScriptResult)
		assert.True(t, ok, test.comment)
		assert.Equal(t, test.res, r.Result(), test.comment)
	}
}
*/
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
