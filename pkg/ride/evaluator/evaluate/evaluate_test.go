package evaluate

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

const seed = "test test"

func newTransferTransaction() *proto.TransferV2 {
	js := `{"type":4,"version":2,"id":"CqjGMbrd5bFmLAv2mUSdphEJSgVWkWa6ZtcMkKmgH2ax","proofs":["5W7hjPpgmmhxevCt4A7y9F8oNJ4V9w2g8jhQgx2qGmBTNsP1p1MpQeKF3cvZULwJ7vQthZfSx2BhL6TWkHSVLzvq"],"senderPublicKey":"14ovLL9a6xbBfftyxGNLKMdbnzGgnaFQjmgUJGdho6nY","assetId":null,"feeAssetId":null,"timestamp":1544715621,"amount":15,"fee":10000,"recipient":"3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"}`
	tv2 := &proto.TransferV2{}
	err := json.Unmarshal([]byte(js), tv2)
	if err != nil {
		panic(err)
	}
	return tv2
}

func newExchangeTransaction() *proto.ExchangeV2 {
	js := `{"senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy","amount": 100000000,"fee": 1100000,"type": 7,"version": 2,"sellMatcherFee": 1100000,"sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3","feeAssetId": null,"proofs": ["DGxkASjpPaKxu8bAv3PJpF9hJ9KAiLsB7bLBTEZXYcWmmc65pHiq5ymJNAazRM2aoLCeTLXXNda5hR9LZNayB69"],"price": 790000,  "id": "5aHKTDvWdVWmo9MPDPoYX83x6hyLJ5ji4eopmoUxELR2",  "order2": {    "version": 2,    "id": "CzBrJkpaWz2AHnT3U8baY3eTfRdymuC7dEqiGpas68tD",    "sender": "3PEjQH31dP2ipvrkouUs12ynKShpBcRQFAT",    "senderPublicKey": "BVtDAjf1MmUdPW2yRHEBiSP5yy7EnxzKgQWpajQM8FCx",    "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",    "assetPair": {      "amountAsset": "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF",      "priceAsset": "CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr"    },    "orderType": "sell",    "amount": 100000000,    "price": 790000,    "timestamp": 1557995955609,    "expiration": 1560501555609,    "matcherFee": 1100000,    "signature": "3Aw94WkF4PUeard435jtJTZLESRFMBuxYRYVVf3GrG48aAxLhbvcXdwsrtALLQ3LYbdNdhR1NUUzdMinU8pLiwWc",    "proofs": [      "3Aw94WkF4PUeard435jtJTZLESRFMBuxYRYVVf3GrG48aAxLhbvcXdwsrtALLQ3LYbdNdhR1NUUzdMinU8pLiwWc"    ]  },  "order1": {    "version": 2,    "id": "APLf7qDhU5puSa5h1KChNBobF8VKoy37PcP7BnhoSPvi",    "sender": "3PEyLyxu4yGJAEmuVRy3G4FvEBUYV6ykQWF",    "senderPublicKey": "28sBbJ7pHNG4VFrvNN43sNsdWYyrTFVAwd98W892mxBQ",    "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",    "assetPair": {      "amountAsset": "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF",      "priceAsset": "CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr"    },    "orderType": "buy",    "amount": 100000000,    "price": 790000,    "timestamp": 1557995158094,    "expiration": 1560500758093,    "matcherFee": 1100000,    "signature": "5zUuSSJyv5NU11RPa91fpQaCXR3xvR1ctjQrfxnNREFhMmbXfACzhfFgV18rdvrvm4X3p3iYK3fxS1TXwgSV5m83",    "proofs": [      "5zUuSSJyv5NU11RPa91fpQaCXR3xvR1ctjQrfxnNREFhMmbXfACzhfFgV18rdvrvm4X3p3iYK3fxS1TXwgSV5m83"    ]  },  "buyMatcherFee": 1100000,  "timestamp": 1557995955923,  "height": 1528811}`
	tx := new(proto.ExchangeV2)
	err := json.Unmarshal([]byte(js), tx)
	if err != nil {
		panic(err)
	}
	return tx
}

func defaultScope(version int) Scope {
	tx, err := NewVariablesFromTransaction(proto.MainNetScheme, newTransferTransaction())
	if err != nil {
		panic(err)
	}
	dataEntries := map[string]proto.DataEntry{
		"integer": &proto.IntegerDataEntry{Key: "integer", Value: 100500},
		"boolean": &proto.BooleanDataEntry{Key: "boolean", Value: true},
		"binary":  &proto.BinaryDataEntry{Key: "binary", Value: []byte("hello")},
		"string":  &proto.StringDataEntry{Key: "string", Value: "world"},
	}
	state := mockstate.State{
		AccountsBalance: 5,
		DataEntries:     dataEntries,
	}
	scope := NewScope(version, proto.MainNetScheme, state)
	scope.SetHeight(5)
	scope.SetTransaction(tx)
	return scope
}

func scopeWithExchangeTx(version int) Scope {
	tx, err := NewVariablesFromTransaction(proto.MainNetScheme, newExchangeTransaction())
	if err != nil {
		panic(err)
	}
	dataEntries := map[string]proto.DataEntry{
		"integer": &proto.IntegerDataEntry{Key: "integer", Value: 100500},
		"boolean": &proto.BooleanDataEntry{Key: "boolean", Value: true},
		"binary":  &proto.BinaryDataEntry{Key: "binary", Value: []byte("hello")},
		"string":  &proto.StringDataEntry{Key: "string", Value: "world"},
	}
	state := mockstate.State{
		AccountsBalance: 5,
		DataEntries:     dataEntries,
	}
	scope := NewScope(version, proto.MainNetScheme, state)
	scope.SetHeight(5)
	scope.SetTransaction(tx)
	return scope
}

func scopeV1withDataTransaction() Scope {
	sk, pk, err := crypto.GenerateKeyPair([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	if err != nil {
		panic(err)
	}
	tx := proto.NewUnsignedData(pk, 100000, 1568640015000)
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "integer", Value: 100500})
	tx.Entries = append(tx.Entries, &proto.BooleanDataEntry{Key: "boolean", Value: true})
	tx.Entries = append(tx.Entries, &proto.BinaryDataEntry{Key: "binary", Value: []byte{0xCA, 0xFE, 0xBE, 0xBE, 0xDE, 0xAD, 0xBE, 0xEF}})
	tx.Entries = append(tx.Entries, &proto.StringDataEntry{Key: "string", Value: "Hello, World!"})
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "someKey", Value: 12345})
	err = tx.Sign(sk)
	if err != nil {
		panic(err)
	}
	tv, err := NewVariablesFromTransaction(proto.MainNetScheme, tx)
	if err != nil {
		panic(err)
	}
	scope := NewScope(1, proto.MainNetScheme, mockstate.State{})
	scope.SetTransaction(tv)
	scope.SetHeight(12345)
	return scope
}

const (
	longScript = `match tx {
  case t : TransferTransaction | MassTransferTransaction | ExchangeTransaction => true
  case _ => false
}`
	hashes = `
let a0 = NoAlg() == NOALG
let a1 = Md5() == MD5
let a2 = Sha1() == SHA1
let a3 = Sha224() == SHA224
let a4 = Sha256() == SHA256
let a5 = Sha384() == SHA384
let a6 = Sha512() == SHA512
let a7 = Sha3224() == SHA3224
let a8 = Sha3256() == SHA3256
let a9 = Sha3384() == SHA3384
let a10 = Sha3512() == SHA3512

a0 && a1 && a2 && a3 && a4 && a5 && a6 && a7 && a8 && a9 && a10
`
	rsaVerify = `
let pk = fromBase64String("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAkDg8m0bCDX7fTbBlHZm+BZIHVOfC2I4klRbjSqwFi/eCdfhGjYRYvu/frpSO0LIm0beKOUvwat6DY4dEhNt2PW3UeQvT2udRQ9VBcpwaJlLreCr837sn4fa9UG9FQFaGofSww1O9eBBjwMXeZr1jOzR9RBIwoL1TQkIkZGaDXRltEaMxtNnzotPfF3vGIZZuZX4CjiitHaSC0zlmQrEL3BDqqoLwo3jq8U3Zz8XUMyQElwufGRbZqdFCeiIs/EoHiJm8q8CVExRoxB0H/vE2uDFK/OXLGTgfwnDlrCa/qGt9Zsb8raUSz9IIHx72XB+kOXTt/GOuW7x2dJvTJIqKTwIDAQAB")
let msg = fromBase64String("REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=")
let sig = fromBase64String("OXVKJwtSoenRmwizPtpjh3sCNmOpU1tnXUnyzl+PEI1P9Rx20GkxkIXlysFT2WdbPn/HsfGMwGJW7YhrVkDXy4uAQxUxSgQouvfZoqGSPp1NtM8iVJOGyKiepgB3GxRzQsev2G8Ik47eNkEDVQa47ct9j198Wvnkf88yjSkK0KxR057MWAi20ipNLirW4ZHDAf1giv68mniKfKxsPWahOA/7JYkv18sxcsISQqRXM8nGI1UuSLt9ER7kIzyAk2mgPCiVlj0hoPGUytmbiUqvEM4QaJfCpR0wVO4f/fob6jwKkGT6wbtia+5xCD7bESIHH8ISDrdexZ01QyNP2r4enw==")
rsaVerify(SHA3256, msg, sig, pk)
`
)

func TestEval(t *testing.T) {
	for _, test := range []struct {
		name   string
		script string
		scope  Scope
		result bool
	}{
		{`let x = 5; 6 > 4`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGAAAAAAAAAAAEYSW6XA==`, defaultScope(2), true},
		{`let x = 5; 6 > x`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGBQAAAAF4Gh24hw==`, defaultScope(2), true},
		{`let x = 5; 6 >= x`, `AQQAAAABeAAAAAAAAAAABQkAAGcAAAACAAAAAAAAAAAGBQAAAAF4jlxXHA==`, defaultScope(2), true},
		{`true`, `AQa3b8tH`, defaultScope(2), true},
		{`false`, `AQfeYll6`, defaultScope(2), false},
		{`let x =  throw(); true`, `AQQAAAABeAkBAAAABXRocm93AAAAAAa7bgf4`, defaultScope(2), true},
		{`let x =  throw();true || x`, `AQQAAAABeAkBAAAABXRocm93AAAAAAMGBgUAAAABeKRnLds=`, defaultScope(2), true},
		{`tx.id == base58''`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAAJBtD70=`, defaultScope(2), false},
		{`tx.id == base58'CqjGMbrd5bFmLAv2mUSdphEJSgVWkWa6ZtcMkKmgH2ax'`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAIK/sOVMfQLb6FHT+QbJpYq4m7jlQoC3GPCMpxfHPeT5F5CUKdw==`, defaultScope(2), true},
		{`let x = tx.id == base58'a';true`, `AQQAAAABeAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAASEGjR0kcA==`, defaultScope(2), true},
		{`tx.proofs[0] == base58'5W7hjPpgmmhxevCt4A7y9F8oNJ4V9w2g8jhQgx2qGmBTNsP1p1MpQeKF3cvZULwJ7vQthZfSx2BhL6TWkHSVLzvq'`, `AQkAAAAAAAACCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAABAAAAQOEtF8V5p+9JHReO90FmBf+yKZW1lLJGBsnkZww94TJ8bNcxWIKfohMXm4BsKKIBUTXLaS6Vcgyw1UTNN5iICQ719Fxf`, defaultScope(2), true},
		{longScript, `AQQAAAAHJG1hdGNoMAUAAAACdHgDAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNFeGNoYW5nZVRyYW5zYWN0aW9uBgMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24GCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB6Ilvok=`, defaultScope(2), true},
		{`match transactionById(tx.id) {case  t: Unit => true case _ => false }`, `AQQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAAAXQFAAAAByRtYXRjaDAGB1+iIek=`, defaultScope(2), true},
		//TODO: The RIDE compiler is broken, test after repair
		// {`Ceiling() == CEILING`, ``, true},
		// {`Floor() == FLOOR`, ``, true},
		// {`HalfEven() == HALFEVEN`, ``, true},
		{`Down() == DOWN`, `AgkAAAAAAAACCQEAAAAERG93bgAAAAAFAAAABERPV052K6LQ`, defaultScope(3), true},
		{`Up() == UP`, `AwkAAAAAAAACCQEAAAACVXAAAAAABQAAAAJVUPGUxeg=`, defaultScope(3), true},
		{`HalfUp() == HALFUP`, `AwkAAAAAAAACCQEAAAAGSGFsZlVwAAAAAAUAAAAGSEFMRlVQbUfpTQ==`, defaultScope(3), true},
		{`HalfDown() == HALFDOWN`, `AgkAAAAAAAACCQEAAAAERG93bgAAAAAFAAAABERPV052K6LQ`, defaultScope(3), true},
		{hashes, `AwQAAAACYTAJAAAAAAAAAgkBAAAABU5vQWxnAAAAAAUAAAAFTk9BTEcEAAAAAmExCQAAAAAAAAIJAQAAAANNZDUAAAAABQAAAANNRDUEAAAAAmEyCQAAAAAAAAIJAQAAAARTaGExAAAAAAUAAAAEU0hBMQQAAAACYTMJAAAAAAAAAgkBAAAABlNoYTIyNAAAAAAFAAAABlNIQTIyNAQAAAACYTQJAAAAAAAAAgkBAAAABlNoYTI1NgAAAAAFAAAABlNIQTI1NgQAAAACYTUJAAAAAAAAAgkBAAAABlNoYTM4NAAAAAAFAAAABlNIQTM4NAQAAAACYTYJAAAAAAAAAgkBAAAABlNoYTUxMgAAAAAFAAAABlNIQTUxMgQAAAACYTcJAAAAAAAAAgkBAAAAB1NoYTMyMjQAAAAABQAAAAdTSEEzMjI0BAAAAAJhOAkAAAAAAAACCQEAAAAHU2hhMzI1NgAAAAAFAAAAB1NIQTMyNTYEAAAAAmE5CQAAAAAAAAIJAQAAAAdTaGEzMzg0AAAAAAUAAAAHU0hBMzM4NAQAAAADYTEwCQAAAAAAAAIJAQAAAAdTaGEzNTEyAAAAAAUAAAAHU0hBMzUxMgMDAwMDAwMDAwMFAAAAAmEwBQAAAAJhMQcFAAAAAmEyBwUAAAACYTMHBQAAAAJhNAcFAAAAAmE1BwUAAAACYTYHBQAAAAJhNwcFAAAAAmE4BwUAAAACYTkHBQAAAANhMTAHRc/wAA==`, defaultScope(3), true},
		{`Unit() == unit`, `AwkAAAAAAAACCQEAAAAEVW5pdAAAAAAFAAAABHVuaXTstg1G`, defaultScope(3), true},
		//{`Unit() == unit`, `AAIDAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAACdHgBAAAACHZlcmlmaWVyAAAAAAkAAAAAAAACBQAAAAR1bml0CQEAAAAEVW5pdAAAAADWI4ad`, true},
	} {
		r, err := reader.NewReaderFromBase64(test.script)
		require.NoError(t, err)

		script, err := BuildScript(r)
		require.NoError(t, err)

		rs, err := Eval(script.Verifier, test.scope)
		require.NoError(t, err)
		assert.Equal(t, test.result, rs, fmt.Sprintf("text: %s", test.name))
	}
}

func BenchmarkEval(b *testing.B) {
	base64 := "AQQAAAABeAkBAAAAEWFkZHJlc3NGcm9tU3RyaW5nAAAAAQIAAAAjM1BKYUR5cHJ2ZWt2UFhQdUF0eHJhcGFjdURKb3BnSlJhVTMEAAAAAWEFAAAAAXgEAAAAAWIFAAAAAWEEAAAAAWMFAAAAAWIEAAAAAWQFAAAAAWMEAAAAAWUFAAAAAWQEAAAAAWYFAAAAAWUJAAAAAAAAAgUAAAABZgUAAAABZS5FHzs="
	_ = `
let x = addressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")

let a = x
let b = a
let c = b
let d = c
let e = d
let f = e

f == e

`

	s := defaultScope(2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r, _ := reader.NewReaderFromBase64(base64)
		script, _ := BuildScript(r)
		b.StartTimer()
		_, _ = Eval(script.Verifier, s)
	}
}

const merkle = `
let rootHash = base64'eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk='
let leafData = base64'AAAm+w=='
let merkleProof = base64'ACBSs2di6rY+9N3mrpQVRNZLGAdRX2WBD6XkrOXuhh42XwEgKhB3Aiij6jqLRuQhrwqv6e05kr89tyxkuFYwUuMCQB8AIKLhp/AFQkokTe/NMQnKFL5eTMvDlFejApmJxPY6Rp8XACAWrdgB8DwvPA8D04E9HgUjhKghAn5aqtZnuKcmpLHztQAgd2OG15WYz90r1WipgXwjdq9WhvMIAtvGlm6E3WYY12oAIJXPPVIdbwOTdUJvCgMI4iape2gvR55vsrO2OmJJtZUNASAya23YyBl+EpKytL9+7cPdkeMMWSjk0Bc0GNnqIisofQ=='

checkMerkleProof(rootHash, merkleProof, leafData)`

func TestFunctions(t *testing.T) {
	for _, test := range []struct {
		code   int
		name   string
		text   string
		script string
		scope  Scope
		result bool
		error  bool
	}{
		//{-1, `parseIntValue`, `parseInt("12345") == 12345`, `AwkAAAAAAAACCQAEtgAAAAECAAAABTEyMzQ1AAAAAAAAADA57cmovA==`, defaultScope(3), true, false},
		//{-1, `value`, `let c = if true then 1 else Unit();value(c) == 1`, `AwQAAAABYwMGAAAAAAAAAAABCQEAAAAEVW5pdAAAAAAJAAAAAAAAAgkBAAAABXZhbHVlAAAAAQUAAAABYwAAAAAAAAAAARfpQ5M=`, defaultScope(3), true, false},
		//{-1, `valueOrErrorMessage`, `let c = if true then 1 else Unit(); valueOrErrorMessage(c, "ALARM!!!") == 1`, `AwQAAAABYwMGAAAAAAAAAAABCQEAAAAEVW5pdAAAAAAJAAAAAAAAAgkBAAAAE3ZhbHVlT3JFcnJvck1lc3NhZ2UAAAACBQAAAAFjAgAAAAhBTEFSTSEhIQAAAAAAAAAAAa5tVyw=`, defaultScope(3), true, false},
		//{-1, `addressFromString`, `addressFromString("12345") == unit`, `AwkAAAAAAAACCQEAAAARYWRkcmVzc0Zyb21TdHJpbmcAAAABAgAAAAUxMjM0NQUAAAAEdW5pdJEPLPE=`, defaultScope(3), true, false},
		//{-1, `addressFromString`, `addressFromString("3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc") == Address(base58'3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc')`, `AwkAAAAAAAACCQEAAAARYWRkcmVzc0Zyb21TdHJpbmcAAAABAgAAACMzUDlERURQNVZieVhReUt0WERVdDJjclJQbjVCN2dzNnVqYwkBAAAAB0FkZHJlc3MAAAABAQAAABoBV0/fzRv7GRFL0qw2njIBPHDG0DpGJ4ecv6EI6ng=`, defaultScope(3), true, false},
		//{-1, `addressFromStringValue`, `addressFromStringValue("3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc") == Address(base58'3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc')`, `AwkAAAAAAAACCQEAAAAcQGV4dHJVc2VyKGFkZHJlc3NGcm9tU3RyaW5nKQAAAAECAAAAIzNQOURFRFA1VmJ5WFF5S3RYRFV0MmNyUlBuNUI3Z3M2dWpjCQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFXT9/NG/sZEUvSrDaeMgE8cMbQOkYnh5y/56rYHQ==`, defaultScope(3), true, false},
		//{-1, `getIntegerFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getInteger(a, "integer") == 100500`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQAEGgAAAAIFAAAAAWECAAAAB2ludGVnZXIAAAAAAAABiJTtgrwb`, defaultScope(3), true, false},
		//{-1, `getIntegerValueFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getIntegerValue(a, "integer") == 100500`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA1MCkAAAACBQAAAAFhAgAAAAdpbnRlZ2VyAAAAAAAAAYiUEnGoww==`, defaultScope(3), true, false},
		//{-1, `getBooleanFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getBoolean(a, "boolean") == true`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQAEGwAAAAIFAAAAAWECAAAAB2Jvb2xlYW4GQ1SwZw==`, defaultScope(3), true, false},
		//{-1, `getBooleanValueFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getBooleanValue(a, "boolean") == true`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA1MSkAAAACBQAAAAFhAgAAAAdib29sZWFuBiG4UlQ=`, defaultScope(3), true, false},
		//{-1, `getBinaryFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getBinary(a, "binary") == base16'68656c6c6f'`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQAEHAAAAAIFAAAAAWECAAAABmJpbmFyeQEAAAAFaGVsbG8AbKgo`, defaultScope(3), true, false},
		//{-1, `getBinaryValueFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getBinaryValue(a, "binary") == base16'68656c6c6f'`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA1MikAAAACBQAAAAFhAgAAAAZiaW5hcnkBAAAABWhlbGxvJ1b3yw==`, defaultScope(3), true, false},
		//{-1, `getStringFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getString(a, "string") == "world"`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA1MikAAAACBQAAAAFhAgAAAAZiaW5hcnkBAAAABWhlbGxvJ1b3yw==`, defaultScope(3), true, false},
		//{-1, `getStringValueFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); getStringValue(a, "string") == "world"`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwkAAAAAAAACCQAEHQAAAAIFAAAAAWECAAAABnN0cmluZwIAAAAFd29ybGSFdQnb`, defaultScope(3), true, false},
		//{-1, `getIntegerFromArrayByKey`, `let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getInteger(d, "integer") == 100500`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEAAAAAIFAAAAAWQCAAAAB2ludGVnZXIAAAAAAAABiJSeStXa`, defaultScope(3), true, false},
		//{-1, `getIntegerValueFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getIntegerValue(d, "integer") == 100500`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA0MCkAAAACBQAAAAFkAgAAAAdpbnRlZ2VyAAAAAAAAAYiUmn7ujg==`, defaultScope(3), true, false},
		//{-1, `getBooleanFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBoolean(d, "boolean") == true`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEQAAAAIFAAAAAWQCAAAAB2Jvb2xlYW4GaWuehg==`, defaultScope(3), true, false},
		//{-1, `getBooleanValueFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBooleanValue(d, "boolean") == true`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA0MSkAAAACBQAAAAFkAgAAAAdib29sZWFuBt3vwJY=`, defaultScope(3), true, false},
		//{-1, `getBinaryFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBinary(d, "binary") == base16'68656c6c6f'`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEgAAAAIFAAAAAWQCAAAABmJpbmFyeQEAAAAFaGVsbG+so7oZ`, defaultScope(3), true, false},
		//{-1, `getBinaryValueFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBinaryValue(d, "binary") == base16'68656c6c6f'`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA0MikAAAACBQAAAAFkAgAAAAZiaW5hcnkBAAAABWhlbGxvpcldYg==`, defaultScope(3), true, false},
		//{-1, `getStringFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getString(d, "string") == "world"`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEwAAAAIFAAAAAWQCAAAABnN0cmluZwIAAAAFd29ybGRFTMLs`, defaultScope(3), true, false},
		//{-1, `getStringValueFromArrayByKey`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getStringValue(d, "string") == "world"`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA0MykAAAACBQAAAAFkAgAAAAZzdHJpbmcCAAAABXdvcmxkCbSDLQ==`, defaultScope(3), true, false},
		//{-1, `getIntegerFromArrayByIndex`, `let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getInteger(d, 0) == 100500`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAKZ2V0SW50ZWdlcgAAAAIFAAAAAWQAAAAAAAAAAAAAAAAAAAABiJTdCjRc`, defaultScope(3), true, false},
		//{-1, `getIntegerValueFromArrayByIndex`, `let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getIntegerValue(d, 0) == 100500`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAVQGV4dHJVc2VyKGdldEludGVnZXIpAAAAAgUAAAABZAAAAAAAAAAAAAAAAAAAAAGIlOyDHCY=`, defaultScope(3), true, false},
		//{-1, `getBooleanFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBoolean(d, 1) == true`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAKZ2V0Qm9vbGVhbgAAAAIFAAAAAWQAAAAAAAAAAAEGlS0yug==`, defaultScope(3), true, false},
		//{-1, `getBooleanValueFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBooleanValue(d, 1) == true`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAVQGV4dHJVc2VyKGdldEJvb2xlYW4pAAAAAgUAAAABZAAAAAAAAAAAAQY8zZ6Y`, defaultScope(3), true, false},
		//{-1, `getBinaryFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBinary(d, 2) == base16'68656c6c6f'`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAJZ2V0QmluYXJ5AAAAAgUAAAABZAAAAAAAAAAAAgEAAAAFaGVsbG/jc7GJ`, defaultScope(3), true, false},
		//{-1, `getBinaryValueFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getBinaryValue(d, 2) == base16'68656c6c6f'`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAUQGV4dHJVc2VyKGdldEJpbmFyeSkAAAACBQAAAAFkAAAAAAAAAAACAQAAAAVoZWxsbwxEPw4=`, defaultScope(3), true, false},
		//{-1, `getStringFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getString(d, 3) == "world"`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAJZ2V0U3RyaW5nAAAAAgUAAAABZAAAAAAAAAAAAwIAAAAFd29ybGTcG8rI`, defaultScope(3), true, false},
		//{-1, `getStringValueFromArrayByIndex`, `let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getStringValue(d, 3) == "world"`, `AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQEAAAAUQGV4dHJVc2VyKGdldFN0cmluZykAAAACBQAAAAFkAAAAAAAAAAADAgAAAAV3b3JsZOGBO8c=`, defaultScope(3), true, false},
		//{-1, `compare Recipient with Address`, `let a = Address(base58'3PKpKgcwArHQVmYWUg6Ljxx31VueBStUKBR'); match tx {case tt: TransferTransaction => tt.recipient == a case _ => false}`, `AwQAAAABYQkBAAAAB0FkZHJlc3MAAAABAQAAABoBV8Q0LvvkEO83LtpdRUhgK760itMpcq1W7AQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR0BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0dAAAAAlyZWNpcGllbnQFAAAAAWEHQOLkRA==`, defaultScope(3), false, false},
		//{-1, `compare Recipient with Address`, `let a = Address(base58'3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3'); match tx {case tt: TransferTransaction => tt.recipient == a case _ => false}`, `AwQAAAABYQkBAAAAB0FkZHJlc3MAAAABAQAAABoBVwX3L9Q7Ao0/8ZNhoE70/41bHPBwqbd27gQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR0BQAAAAckbWF0Y2gwCQAAAAAAAAIIBQAAAAJ0dAAAAAlyZWNpcGllbnQFAAAAAWEHd9vYmA==`, defaultScope(3), true, false},
		//{-1, `compare Address with Recipient`, `let a = Address(base58'3PKpKgcwArHQVmYWUg6Ljxx31VueBStUKBR'); match tx {case tt: TransferTransaction => a == tt.recipient case _ => false}`, `AwQAAAABYQkBAAAAB0FkZHJlc3MAAAABAQAAABoBV8Q0LvvkEO83LtpdRUhgK760itMpcq1W7AQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR0BQAAAAckbWF0Y2gwCQAAAAAAAAIFAAAAAWEIBQAAAAJ0dAAAAAlyZWNpcGllbnQHG1tX4w==`, defaultScope(3), false, false},
		//{-1, `compare Address with Recipient`, `let a = Address(base58'3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3'); match tx {case tt: TransferTransaction => a == tt.recipient case _ => false}`, `AwQAAAABYQkBAAAAB0FkZHJlc3MAAAABAQAAABoBVwX3L9Q7Ao0/8ZNhoE70/41bHPBwqbd27gQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR0BQAAAAckbWF0Y2gwCQAAAAAAAAIFAAAAAWEIBQAAAAJ0dAAAAAlyZWNpcGllbnQHw8RWfw==`, defaultScope(3), true, false},
		//
		//{-1, `getIntegerFromDataTransactionByKey`, `match tx {case d: DataTransaction => extract(getInteger(d.data, "integer")) == 100500 case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABZAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAHZXh0cmFjdAAAAAEJAAQQAAAAAggFAAAAAWQAAAAEZGF0YQIAAAAHaW50ZWdlcgAAAAAAAAGIlAfN4Sfl`, scopeV1withDataTransaction(), true, false},
		//{-1, `getIntegerFromDataTransactionByKey`, `match tx {case dt: DataTransaction => let a = match getInteger(dt.data, "someKey") {case v: Int => v case _ => -1}; a >= 0 case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAACZHQFAAAAByRtYXRjaDAEAAAAAWEEAAAAByRtYXRjaDEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAAB3NvbWVLZXkDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAA0ludAQAAAABdgUAAAAHJG1hdGNoMQUAAAABdgD//////////wkAAGcAAAACBQAAAAFhAAAAAAAAAAAAB1mStww=`, scopeV1withDataTransaction(), true, false},
		//{-1, `getIntegerFromDataTransactionByKey`, `match tx {case dt: DataTransaction => let x = match getInteger(dt.data, "someKey") {case i: Int => true case _ => false};let y = match getInteger(dt.data, "someKey") {case v: Int => v case _ => -1}; x && y >= 0 case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAACZHQFAAAAByRtYXRjaDAEAAAAAXgEAAAAByRtYXRjaDEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAAB3NvbWVLZXkDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAA0ludAQAAAABaQUAAAAHJG1hdGNoMQYHBAAAAAF5BAAAAAckbWF0Y2gxCQAEEAAAAAIIBQAAAAJkdAAAAARkYXRhAgAAAAdzb21lS2V5AwkAAAEAAAACBQAAAAckbWF0Y2gxAgAAAANJbnQEAAAAAXYFAAAAByRtYXRjaDEFAAAAAXYA//////////8DBQAAAAF4CQAAZwAAAAIFAAAAAXkAAAAAAAAAAAAHB5sznFY=`, scopeV1withDataTransaction(), true, false},
		//{-1, `matchIntegerFromDataTransactionByKey`, `let x = match tx {case d: DataTransaction => match getInteger(d.data, "integer") {case i: Int => i case _ => 0}case _ => 0}; x == 100500`, `AQQAAAABeAQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABZAUAAAAHJG1hdGNoMAQAAAAHJG1hdGNoMQkABBAAAAACCAUAAAABZAAAAARkYXRhAgAAAAdpbnRlZ2VyAwkAAAEAAAACBQAAAAckbWF0Y2gxAgAAAANJbnQEAAAAAWkFAAAAByRtYXRjaDEFAAAAAWkAAAAAAAAAAAAAAAAAAAAAAAAJAAAAAAAAAgUAAAABeAAAAAAAAAGIlApOoB4=`, scopeV1withDataTransaction(), true, false},
		//{-1, `matchIntegerFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); let i = getInteger(a, "integer"); let x = match i {case i: Int => i case _ => 0}; x == 100500`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwQAAAABaQkABBoAAAACBQAAAAFhAgAAAAdpbnRlZ2VyBAAAAAF4BAAAAAckbWF0Y2gwBQAAAAFpAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAANJbnQEAAAAAWkFAAAAByRtYXRjaDAFAAAAAWkAAAAAAAAAAAAJAAAAAAAAAgUAAAABeAAAAAAAAAGIlKWtlDk=`, defaultScope(3), true, false},
		//{-1, `ifIntegerFromState`, `let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); let i = getInteger(a, "integer"); let x = if i != 0 then i else 0; x == 100500`, `AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwQAAAABaQkABBoAAAACBQAAAAFhAgAAAAdpbnRlZ2VyBAAAAAF4AwkBAAAAAiE9AAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQAAAAAAAAAAAAkAAAAAAAACBQAAAAF4AAAAAAAAAYiU1cZgMA==`, defaultScope(3), true, false},
		//
		//{-1, `string concatenation`, `let a = base16'cafe'; let b = base16'bebe'; toBase58String(a) + "/" + toBase58String(b) == "GSy/FWu"`, `AwQAAAABYQEAAAACyv4EAAAAAWIBAAAAAr6+CQAAAAAAAAIJAAEsAAAAAgkAASwAAAACCQACWAAAAAEFAAAAAWECAAAAAS8JAAJYAAAAAQUAAAABYgIAAAAHR1N5L0ZXdc2NqKQ=`, defaultScope(3), true, false},
		//{-1, `match on ByteVector`, `match tx {case etx: ExchangeTransaction => match etx.sellOrder.assetPair.amountAsset {case ByteVector => true case _ => false} case _ => false}`, `AwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE0V4Y2hhbmdlVHJhbnNhY3Rpb24EAAAAA2V0eAUAAAAHJG1hdGNoMAQAAAAHJG1hdGNoMQgICAUAAAADZXR4AAAACXNlbGxPcmRlcgAAAAlhc3NldFBhaXIAAAALYW1vdW50QXNzZXQEAAAACkJ5dGVWZWN0b3IFAAAAByRtYXRjaDEGB76y+jI=`, scopeWithExchangeTx(3), true, false},

		{-1, `3P8M8XGF2uzDazV5fzdKNxrbC3YqCWScKxw`, ``, `AwoBAAAAGVJlbW92ZVVuZGVyc2NvcmVJZlByZXNlbnQAAAABAAAACXJlbWFpbmluZwMJAABmAAAAAgkAATEAAAABBQAAAAlyZW1haW5pbmcAAAAAAAAAAAAJAAEwAAAAAgUAAAAJcmVtYWluaW5nAAAAAAAAAAABBQAAAAlyZW1haW5pbmcKAQAAABJQYXJzZU5leHRBdHRyaWJ1dGUAAAABAAAACXJlbWFpbmluZwQAAAABcwkAATEAAAABBQAAAAlyZW1haW5pbmcDCQAAZgAAAAIFAAAAAXMAAAAAAAAAAAAEAAAAAm5uCQEAAAANcGFyc2VJbnRWYWx1ZQAAAAEJAAEvAAAAAgUAAAAJcmVtYWluaW5nAAAAAAAAAAACBAAAAAF2CQABLwAAAAIJAAEwAAAAAgUAAAAJcmVtYWluaW5nAAAAAAAAAAACBQAAAAJubgQAAAAMdG1wUmVtYWluaW5nCQABMAAAAAIFAAAACXJlbWFpbmluZwkAAGQAAAACBQAAAAJubgAAAAAAAAAAAgQAAAAOcmVtYWluaW5nU3RhdGUJAQAAABlSZW1vdmVVbmRlcnNjb3JlSWZQcmVzZW50AAAAAQUAAAAMdG1wUmVtYWluaW5nCQAETAAAAAIFAAAAAXYJAARMAAAAAgUAAAAOcmVtYWluaW5nU3RhdGUFAAAAA25pbAkAAAIAAAABAgAAADRFbXB0eSBzdHJpbmcgd2FzIHBhc3NlZCBpbnRvIHBhcnNlTmV4dEF0dHJpYnV0ZSBmdW5jCgEAAAATUGFyc2VHYW1lUmF3RGF0YVN0cgAAAAEAAAALcmF3U3RhdGVTdHIEAAAACWdhbWVTdGF0ZQkBAAAAElBhcnNlTmV4dEF0dHJpYnV0ZQAAAAEFAAAAC3Jhd1N0YXRlU3RyBAAAAAxwbGF5ZXJDaG9pY2UJAQAAABJQYXJzZU5leHRBdHRyaWJ1dGUAAAABCQABkQAAAAIFAAAACWdhbWVTdGF0ZQAAAAAAAAAAAQQAAAAOcGxheWVyUHViS2V5NTgJAQAAABJQYXJzZU5leHRBdHRyaWJ1dGUAAAABCQABkQAAAAIFAAAADHBsYXllckNob2ljZQAAAAAAAAAAAQQAAAANc3RhcnRlZEhlaWdodAkBAAAAElBhcnNlTmV4dEF0dHJpYnV0ZQAAAAEJAAGRAAAAAgUAAAAOcGxheWVyUHViS2V5NTgAAAAAAAAAAAEEAAAABndpbkFtdAkBAAAAElBhcnNlTmV4dEF0dHJpYnV0ZQAAAAEJAAGRAAAAAgUAAAANc3RhcnRlZEhlaWdodAAAAAAAAAAAAQkABEwAAAACCQABkQAAAAIFAAAACWdhbWVTdGF0ZQAAAAAAAAAAAAkABEwAAAACCQABkQAAAAIFAAAADHBsYXllckNob2ljZQAAAAAAAAAAAAkABEwAAAACCQABkQAAAAIFAAAADnBsYXllclB1YktleTU4AAAAAAAAAAAACQAETAAAAAIJAAGRAAAAAgUAAAANc3RhcnRlZEhlaWdodAAAAAAAAAAAAAkABEwAAAACCQABkQAAAAIFAAAABndpbkFtdAAAAAAAAAAAAAUAAAADbmlsCQAAAAAAAAIJAQAAABNQYXJzZUdhbWVSYXdEYXRhU3RyAAAAAQIAAABWMDNXT05fMDUzNTY0Ml80NDM4OXBhNmlOaHgxaEZEcU5abVNBVEp1ZldaMUVMbUtkOUh4eXpQUUtIdWMzXzA3MTYxMDU1N18wOTExNDAwMDAwMF8wMTYJAARMAAAAAgIAAAADV09OCQAETAAAAAICAAAABTM1NjQyCQAETAAAAAICAAAALDM4OXBhNmlOaHgxaEZEcU5abVNBVEp1ZldaMUVMbUtkOUh4eXpQUUtIdWMzCQAETAAAAAICAAAABzE2MTA1NTcJAARMAAAAAgIAAAAJMTE0MDAwMDAwCQAETAAAAAICAAAAATYFAAAAA25pbBz/HnU=`, scopeWithExchangeTx(3), true, false},

		//{0, "EQ", `5 == 5`, `AQkAAAAAAAACAAAAAAAAAAAFAAAAAAAAAAAFqWG0Fw==`, defaultScope(2), true, false},
		//{1, "ISINSTANCEOF", `match tx {case t : TransferTransaction => true case _  => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB5yQ/+k=`, defaultScope(2), true, false},
		//{2, `THROW`, `true && throw("mess")`, `AQMGCQAAAgAAAAECAAAABG1lc3MH7PDwAQ==`, defaultScope(2), false, true},
		//{100, `SUM_LONG`, `1 + 1 > 0`, `AQkAAGYAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAABiJjSk`, defaultScope(2), true, false},
		//{101, `SUB_LONG`, `2 - 1 > 0`, `AQkAAGYAAAACCQAAZQAAAAIAAAAAAAAAAAIAAAAAAAAAAAEAAAAAAAAAAABqsps1`, defaultScope(2), true, false},
		//{102, `GT_LONG`, `1 > 0`, `AQkAAGYAAAACAAAAAAAAAAABAAAAAAAAAAAAyAIM4w==`, defaultScope(2), true, false},
		//{103, `GE_LONG`, `1 >= 0`, `AQkAAGcAAAACAAAAAAAAAAABAAAAAAAAAAAAm30DnQ==`, defaultScope(2), true, false},
		//{104, `MUL_LONG`, `2 * 2>0`, `AQkAAGYAAAACCQAAaAAAAAIAAAAAAAAAAAIAAAAAAAAAAAIAAAAAAAAAAABCMM5o`, defaultScope(2), true, false},
		//{105, `DIV_LONG`, `4 / 2>0`, `AQkAAGYAAAACCQAAaQAAAAIAAAAAAAAAAAQAAAAAAAAAAAIAAAAAAAAAAAAadVma`, defaultScope(2), true, false},
		//{105, `DIV_LONG`, `10000 / (27+121) == 67`, `AwkAAAAAAAACCQAAaQAAAAIAAAAAAAAAJxAJAABkAAAAAgAAAAAAAAAAGwAAAAAAAAAAeQAAAAAAAAAAQ1vSVaQ=`, defaultScope(2), true, false},
		//{106, `MOD_LONG`, `-10 % 6>0`, `AQkAAGYAAAACCQAAagAAAAIA//////////YAAAAAAAAAAAYAAAAAAAAAAAB5rBSH`, defaultScope(2), true, false},
		//{106, `MOD_LONG`, `10000 % 100 == 0`, `AwkAAAAAAAACCQAAagAAAAIAAAAAAAAAJxAAAAAAAAAAAGQAAAAAAAAAAAAmFt9K`, defaultScope(2), true, false},
		//{107, `FRACTION`, `fraction(10, 5, 2)>0`, `AQkAAGYAAAACCQAAawAAAAMAAAAAAAAAAAoAAAAAAAAAAAUAAAAAAAAAAAIAAAAAAAAAAACRyFu2`, defaultScope(2), true, false},
		//{108, `POW`, `pow(12, 1, 3456, 3, 2, Down()) == 187`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIJAQAAAAREb3duAAAAAAAAAAAAAAAAu9llw2M=`, defaultScope(3), true, false},
		//{108, `POW`, `pow(12, 1, 3456, 3, 2, UP) == 187`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIFAAAAAlVQAAAAAAAAAAC7FUMwCQ==`, defaultScope(3), false, false},
		//{108, `POW`, `pow(12, 1, 3456, 3, 2, UP) == 188`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIFAAAAAlVQAAAAAAAAAAC8evjDQQ==`, defaultScope(3), true, false},
		//{109, `LOG`, `log(16, 0, 2, 0, 0, CEILING) == 4`, `AwkAAAAAAAACCQAAbQAAAAYAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAAAFAAAAB0NFSUxJTkcAAAAAAAAAAARh6Dy6`, defaultScope(3), true, false},
		//{109, `LOG`, `log(100, 0, 10, 0, 0, CEILING) == 2`, `AwkAAAAAAAACCQAAbQAAAAYAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAFAAAAB0NFSUxJTkcAAAAAAAAAAAJ7Op42`, defaultScope(3), true, false},
		//
		//{200, `SIZE_BYTES`, `size(base58'abcd') > 0`, `AQkAAGYAAAACCQAAyAAAAAEBAAAAA2QGAgAAAAAAAAAAACMcdM4=`, defaultScope(2), true, false},
		//{201, `TAKE_BYTES`, `size(take(base58'abcd', 2)) == 2`, `AQkAAAAAAAACCQAAyAAAAAEJAADJAAAAAgEAAAADZAYCAAAAAAAAAAACAAAAAAAAAAACccrCZg==`, defaultScope(2), true, false},
		//{202, `DROP_BYTES`, `size(drop(base58'abcd', 2)) > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAADKAAAAAgEAAAADZAYCAAAAAAAAAAACAAAAAAAAAAAA+srbUQ==`, defaultScope(2), true, false},
		//{203, `SUM_BYTES`, `size(base58'ab' + base58'cd') > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAADLAAAAAgEAAAACB5wBAAAAAggSAAAAAAAAAAAAo+LRIA==`, defaultScope(2), true, false},
		//
		//{300, `SUM_STRING`, `"ab"+"cd" == "abcd"`, `AQkAAAAAAAACCQABLAAAAAICAAAAAmFiAgAAAAJjZAIAAAAEYWJjZMBJvls=`, defaultScope(2), true, false},
		//{303, `TAKE_STRING`, `take("abcd", 2) == "ab"`, `AQkAAAAAAAACCQABLwAAAAICAAAABGFiY2QAAAAAAAAAAAICAAAAAmFiiXc+oQ==`, defaultScope(2), true, false},
		//{304, `DROP_STRING`, `drop("abcd", 2) == "cd"`, `AQkAAAAAAAACCQABMAAAAAICAAAABGFiY2QAAAAAAAAAAAICAAAAAmNkZQdjWQ==`, defaultScope(2), true, false},
		//{305, `SIZE_STRING`, `size("abcd") == 4`, `AQkAAAAAAAACCQABMQAAAAECAAAABGFiY2QAAAAAAAAAAAScZzsq`, defaultScope(2), true, false},
		//
		//{400, `SIZE_LIST`, `size(tx.proofs) == 8`, `AwkAAAAAAAACCQABkAAAAAEIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAgEd23x`, defaultScope(2), true, false},
		//{401, `GET_LIST`, `size(tx.proofs[0]) > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAAAAAAAAAAAAFF6iVo=`, defaultScope(2), true, false},
		//{410, `LONG_TO_BYTES`, `toBytes(1) == base58'11111112'`, `AQkAAAAAAAACCQABmgAAAAEAAAAAAAAAAAEBAAAACAAAAAAAAAABm8cc1g==`, defaultScope(2), true, false},
		//{411, `STRING_TO_BYTES`, `toBytes("привет") == base58'4wUjatAwfVDjaHQVX'`, `AQkAAAAAAAACCQABmwAAAAECAAAADNC/0YDQuNCy0LXRggEAAAAM0L/RgNC40LLQtdGCuUGFxw==`, defaultScope(2), true, false},
		//{412, `BOOLEAN_TO_BYTES`, `toBytes(true) == base58'2'`, `AQkAAAAAAAACCQABnAAAAAEGAQAAAAEBJRrQbw==`, defaultScope(2), true, false},
		//{420, `LONG_TO_STRING`, `toString(5) == "5"`, `AQkAAAAAAAACCQABpAAAAAEAAAAAAAAAAAUCAAAAATXPb5tR`, defaultScope(2), true, false},
		//{421, `BOOLEAN_TO_STRING`, `toString(true) == "true"`, `AQkAAAAAAAACCQABpQAAAAEGAgAAAAR0cnVlL6ZrWg==`, defaultScope(2), true, false},
		//
		//{500, `SIGVERIFY`, `sigVerify(tx.bodyBytes, tx.proofs[0], base58'14ovLL9a6xbBfftyxGNLKMdbnzGgnaFQjmgUJGdho6nY')`, `AQkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAABAAAAIAD5y2Wf7zxfv7l+9tcWxyLAbktd9nCbdvFMnxmREqV1igWi3A==`, defaultScope(3), true, false},
		//{501, `KECCAK256`, `keccak256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfUAAAABAQAAAAEhAQAAAAEhKeR77g==`, defaultScope(3), true, false},
		//{502, `BLAKE256`, `blake2b256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfYAAAABAQAAAAEhAQAAAAEh50D2WA==`, defaultScope(3), true, false},
		//{503, `SHA256`, `sha256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfcAAAABAQAAAAEhAQAAAAEhVojmeg==`, defaultScope(3), true, false},
		//{504, `RSAVERIFY`, rsaVerify, `AwQAAAACcGsJAAJbAAAAAQIAAAGITUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUFrRGc4bTBiQ0RYN2ZUYkJsSFptK0JaSUhWT2ZDMkk0a2xSYmpTcXdGaS9lQ2RmaEdqWVJZdnUvZnJwU08wTEltMGJlS09VdndhdDZEWTRkRWhOdDJQVzNVZVF2VDJ1ZFJROVZCY3B3YUpsTHJlQ3I4MzdzbjRmYTlVRzlGUUZhR29mU3d3MU85ZUJCandNWGVacjFqT3pSOVJCSXdvTDFUUWtJa1pHYURYUmx0RWFNeHRObnpvdFBmRjN2R0laWnVaWDRDamlpdEhhU0MwemxtUXJFTDNCRHFxb0x3bzNqcThVM1p6OFhVTXlRRWx3dWZHUmJacWRGQ2VpSXMvRW9IaUptOHE4Q1ZFeFJveEIwSC92RTJ1REZLL09YTEdUZ2Z3bkRsckNhL3FHdDlac2I4cmFVU3o5SUlIeDcyWEIra09YVHQvR091Vzd4MmRKdlRKSXFLVHdJREFRQUIEAAAAA21zZwkAAlsAAAABAgAAAFBSRUlpTjJoRFFVeElKVlF6ZGsxelFTcFhjbFJSZWxFeFZXZCtZR1FvT3l4MEtIZHVQekZtY1U4elVXb3NXaUE3YUZsb09XcGxjbEF4UENVPQQAAAADc2lnCQACWwAAAAECAAABWE9YVktKd3RTb2VuUm13aXpQdHBqaDNzQ05tT3BVMXRuWFVueXpsK1BFSTFQOVJ4MjBHa3hrSVhseXNGVDJXZGJQbi9Ic2ZHTXdHSlc3WWhyVmtEWHk0dUFReFV4U2dRb3V2ZlpvcUdTUHAxTnRNOGlWSk9HeUtpZXBnQjNHeFJ6UXNldjJHOElrNDdlTmtFRFZRYTQ3Y3Q5ajE5OFd2bmtmODh5alNrSzBLeFIwNTdNV0FpMjBpcE5MaXJXNFpIREFmMWdpdjY4bW5pS2ZLeHNQV2FoT0EvN0pZa3YxOHN4Y3NJU1FxUlhNOG5HSTFVdVNMdDlFUjdrSXp5QWsybWdQQ2lWbGowaG9QR1V5dG1iaVVxdkVNNFFhSmZDcFIwd1ZPNGYvZm9iNmp3S2tHVDZ3YnRpYSs1eENEN2JFU0lISDhJU0RyZGV4WjAxUXlOUDJyNGVudz09CQAB+AAAAAQFAAAAB1NIQTMyNTYFAAAAA21zZwUAAAADc2lnBQAAAAJwa8wcz28=`, defaultScope(3), true, false},
		//
		//{600, `TOBASE58`, `toBase58String(base58'a') == "a"`, `AQkAAAAAAAACCQACWAAAAAEBAAAAASECAAAAAWFcT4nY`, defaultScope(2), true, false},
		//{601, `FROMBASE58`, `fromBase58String("a") == base58'a'`, `AQkAAAAAAAACCQACWQAAAAECAAAAAWEBAAAAASEB1Qmd`, defaultScope(2), true, false},
		//{601, `FROMBASE58`, `fromBase58String(extract("")) == base58''`, `AwkAAAAAAAACCQACWQAAAAEJAQAAAAdleHRyYWN0AAAAAQIAAAAAAQAAAAAt2xTN`, defaultScope(2), true, false},
		//{602, `TOBASE64`, `toBase64String(base16'544553547465737454455354') == "VEVTVHRlc3RURVNU"`, `AwkAAAAAAAACCQACWgAAAAEBAAAADFRFU1R0ZXN0VEVTVAIAAAAQVkVWVFZIUmxjM1JVUlZOVd6DVfc=`, defaultScope(2), true, false},
		//{603, `FROMBASE64`, `base16'544553547465737454455354' == fromBase64String("VEVTVHRlc3RURVNU")`, `AwkAAAAAAAACAQAAAAxURVNUdGVzdFRFU1QJAAJbAAAAAQIAAAAQVkVWVFZIUmxjM1JVUlZOVV+c29Q=`, defaultScope(2), true, false},
		//{604, `TOBASE16`, `toBase16String(base64'VEVTVHRlc3RURVNU') == "544553547465737454455354"`, `AwkAAAAAAAACCQACXAAAAAEBAAAADFRFU1R0ZXN0VEVTVAIAAAAYNTQ0NTUzNTQ3NDY1NzM3NDU0NDU1MzU07NMrMQ==`, defaultScope(3), true, false},
		//{605, `FROMBASE16`, `fromBase16String("544553547465737454455354") == base64'VEVTVHRlc3RURVNU'`, `AwkAAAAAAAACCQACXQAAAAECAAAAGDU0NDU1MzU0NzQ2NTczNzQ1NDQ1NTM1NAEAAAAMVEVTVHRlc3RURVNUFBEa5A==`, defaultScope(3), true, false},
		//
		//{700, `CHECKMERKLEPROOF`, merkle, `AwQAAAAIcm9vdEhhc2gBAAAAIHofX5tx3h2d1wP1HzKQvR0sC1TMgS4JACS9DCFQSKGJBAAAAAhsZWFmRGF0YQEAAAAEAAAm+wQAAAALbWVya2xlUHJvb2YBAAAA7gAgUrNnYuq2PvTd5q6UFUTWSxgHUV9lgQ+l5Kzl7oYeNl8BICoQdwIoo+o6i0bkIa8Kr+ntOZK/PbcsZLhWMFLjAkAfACCi4afwBUJKJE3vzTEJyhS+XkzLw5RXowKZicT2OkafFwAgFq3YAfA8LzwPA9OBPR4FI4SoIQJ+WqrWZ7inJqSx87UAIHdjhteVmM/dK9VoqYF8I3avVobzCALbxpZuhN1mGNdqACCVzz1SHW8Dk3VCbwoDCOImqXtoL0eeb7KztjpiSbWVDQEgMmtt2MgZfhKSsrS/fu3D3ZHjDFko5NAXNBjZ6iIrKH0JAAK8AAAAAwUAAAAIcm9vdEhhc2gFAAAAC21lcmtsZVByb29mBQAAAAhsZWFmRGF0YXe8Icg=`, defaultScope(3), true, false},
		//
		//{1000, `GETTRANSACTIONBYID`, `match transactionById(tx.id) {case  t: Unit => true case _ => false }`, `AQQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAAAXQFAAAAByRtYXRjaDAGB1+iIek=`, defaultScope(2), true, false},
		//{1001, `TRANSACTIONHEIGHTBYID`, `transactionHeightById(base58'aaaa') == 5`, `AQkAAAAAAAACCQAD6QAAAAEBAAAAA2P4ZwAAAAAAAAAABSLhRM4=`, defaultScope(2), false, false},
		//{1003, `ACCOUNTASSETBALANCE`, `assetBalance(tx.sender, base58'BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD') == 5`, `AQkAAAAAAAACCQAD6wAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIBAAAAIJxQIls8iGUc1935JolBz6bYc37eoPDtScOAM0lTNhY0AAAAAAAAAAAFjp6PBg==`, defaultScope(2), true, false},
		//{1061, `ADDRESSTOSTRING`, `toString(Address(base58'3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg')) == "3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg"`, `AwkAAAAAAAACCQAEJQAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVcMIZxOsk2Gw5Avd0ztqi+phtb1Bb83MiUCAAAAIzNQMzMzNnJOU1NVOGJEQXFEYjZTNWpOczhESmIyYmZObXBnkXj7Cg==`, defaultScope(3), true, false},
		//{1061, `ADDRESSTOSTRING`, `toString(Address(base58'3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg')) == "3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpf"`, `AwkAAAAAAAACCQAEJQAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVcMIZxOsk2Gw5Avd0ztqi+phtb1Bb83MiUCAAAAIzNQMzMzNnJOU1NVOGJEQXFEYjZTNWpOczhESmIyYmZObXBmb/6mcg==`, defaultScope(3), false, false},
		//{1100, `CONS`, `size([1, "2"]) == 2`, `AwkAAAAAAAACCQABkAAAAAEJAARMAAAAAgAAAAAAAAAAAQkABEwAAAACAgAAAAEyBQAAAANuaWwAAAAAAAAAAAKuUcc0`, defaultScope(3), true, false},
		//{1100, `CONS`, `size(cons(1, nil)) == 1`, `AwkAAAAAAAACCQABkAAAAAEJAARMAAAAAgAAAAAAAAAAAQUAAAADbmlsAAAAAAAAAAABX96esw==`, defaultScope(3), true, false},
		//{1100, `CONS`, `[1, 2, 3, 4, 5][4] == 5`, `AwkAAAAAAAACCQABkQAAAAIJAARMAAAAAgAAAAAAAAAAAQkABEwAAAACAAAAAAAAAAACCQAETAAAAAIAAAAAAAAAAAMJAARMAAAAAgAAAAAAAAAABAkABEwAAAACAAAAAAAAAAAFBQAAAANuaWwAAAAAAAAAAAQAAAAAAAAAAAVrPjYC`, defaultScope(3), true, false},
		//{1100, `CONS`, `[1, 2, 3, 4, 5][4] == 4`, `AwkAAAAAAAACCQABkQAAAAIJAARMAAAAAgAAAAAAAAAAAQkABEwAAAACAAAAAAAAAAACCQAETAAAAAIAAAAAAAAAAAMJAARMAAAAAgAAAAAAAAAABAkABEwAAAACAAAAAAAAAAAFBQAAAANuaWwAAAAAAAAAAAQAAAAAAAAAAASbi8eN`, defaultScope(3), false, false},
		//{1200, `UTF8STR`, `toUtf8String(base16'536f6d65207465737420737472696e67') == "Some test string"`, `AwkAAAAAAAACCQAEsAAAAAEBAAAAEFNvbWUgdGVzdCBzdHJpbmcCAAAAEFNvbWUgdGVzdCBzdHJpbme0Wj5y`, defaultScope(3), true, false},
		//{1200, `UTF8STR`, `toUtf8String(base16'536f6d65207465737420737472696e67') == "blah-blah-blah"`, `AwkAAAAAAAACCQAEsAAAAAEBAAAAEFNvbWUgdGVzdCBzdHJpbmcCAAAADmJsYWgtYmxhaC1ibGFojpjG3g==`, defaultScope(3), false, false},
		//{1201, `TOINT`, `toInt(base16'0000000000003039') == 12345`, `AwkAAAAAAAACCQAEsQAAAAEBAAAACAAAAAAAADA5AAAAAAAAADA5WVzTeQ==`, defaultScope(3), true, false},
		//{1201, `TOINT`, `toInt(base16'3930000000000000') == 12345`, `AwkAAAAAAAACCQAEsQAAAAEBAAAACDkwAAAAAAAAAAAAAAAAADA5Vq02Hg==`, defaultScope(3), false, false},
		//{1202, `TOINT_OFF`, `toInt(base16'ffffff0000000000003039', 3) == 12345`, `AwkAAAAAAAACCQAEsgAAAAIBAAAAC////wAAAAAAADA5AAAAAAAAAAADAAAAAAAAADA5pGJt2g==`, defaultScope(3), true, false},
		//{1202, `TOINT_OFF`, `toInt(base16'ffffff0000000000003039', 2) == 12345`, `AwkAAAAAAAACCQAEsgAAAAIBAAAAC////wAAAAAAADA5AAAAAAAAAAACAAAAAAAAADA57UQA4Q==`, defaultScope(3), false, false},
		//{1203, `INDEXOF`, `indexOf("cafe bebe dead beef cafe bebe", "bebe") == 5`, `AwkAAAAAAAACCQAEswAAAAICAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAFyqpjwQ==`, defaultScope(3), true, false},
		//{1203, `INDEXOF`, `indexOf("cafe bebe dead beef cafe bebe", "fox") == unit`, `AwkAAAAAAAACCQAEswAAAAICAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAANmb3gFAAAABHVuaXS7twzl`, defaultScope(3), true, false},
		//{1204, `INDEXOFN`, `indexOf("cafe bebe dead beef cafe bebe", "bebe", 0) == 5`, `AwkAAAAAAAACCQAEtAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAAAAAAAAAAAAAFFBPTAA==`, defaultScope(3), true, false},
		//{1204, `INDEXOFN`, `indexOf("cafe bebe dead beef cafe bebe", "bebe", 10) == 25`, `AwkAAAAAAAACCQAEtAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAKAAAAAAAAAAAZVBpWMw==`, defaultScope(3), true, false},
		//{1204, `INDEXOFN`, `indexOf("cafe bebe dead beef cafe bebe", "dead", 10) == 10`, `AwkAAAAAAAACCQAEtAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARkZWFkAAAAAAAAAAAKAAAAAAAAAAAKstuWEQ==`, defaultScope(3), true, false},
		//{1204, `INDEXOFN`, `indexOf("cafe bebe dead beef cafe bebe", "dead", 11) == unit`, `AwkAAAAAAAACCQAEtAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARkZWFkAAAAAAAAAAALBQAAAAR1bml0f2q2UQ==`, defaultScope(3), true, false},
		//{1205, `SPLIT`, `split("abcd", "") == ["a", "b", "c", "d"]`, `AwkAAAAAAAACCQAEtQAAAAICAAAABGFiY2QCAAAAAAkABEwAAAACAgAAAAFhCQAETAAAAAICAAAAAWIJAARMAAAAAgIAAAABYwkABEwAAAACAgAAAAFkBQAAAANuaWwrnSMu`, defaultScope(3), true, false},
		//{1205, `SPLIT`, `split("one two three", " ") == ["one", "two", "three"]`, `AwkAAAAAAAACCQAEtQAAAAICAAAADW9uZSB0d28gdGhyZWUCAAAAASAJAARMAAAAAgIAAAADb25lCQAETAAAAAICAAAAA3R3bwkABEwAAAACAgAAAAV0aHJlZQUAAAADbmlsdBcUog==`, defaultScope(3), true, false},
		//{1206, `PARSEINT`, `parseInt("12345") == 12345`, `AwkAAAAAAAACCQAEtgAAAAECAAAABTEyMzQ1AAAAAAAAADA57cmovA==`, defaultScope(3), true, false},
		//{1206, `PARSEINT`, `parseInt("0x12345") == unit`, `AwkAAAAAAAACCQAEtgAAAAECAAAABzB4MTIzNDUFAAAABHVuaXQvncQM`, defaultScope(3), true, false},
		//{1207, `LASTINDEXOF`, `lastIndexOf("cafe bebe dead beef cafe bebe", "bebe") == 25`, `AwkAAAAAAAACCQAEtwAAAAICAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAZDUvNng==`, defaultScope(3), true, false},
		//{1207, `LASTINDEXOF`, `lastIndexOf("cafe bebe dead beef cafe bebe", "fox") == unit`, `AwkAAAAAAAACCQAEtwAAAAICAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAANmb3gFAAAABHVuaXSK8YYp`, defaultScope(3), true, false},
		//{1208, `LASTINDEXOFN`, `lastIndexOf("cafe bebe dead beef cafe bebe", "bebe", 30) == 25`, `AwkAAAAAAAACCQAEuAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAeAAAAAAAAAAAZus4/9A==`, defaultScope(3), true, false},
		//{1208, `LASTINDEXOFN`, `lastIndexOf("cafe bebe dead beef cafe bebe", "bebe", 10) == 5`, `AwkAAAAAAAACCQAEuAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARiZWJlAAAAAAAAAAAKAAAAAAAAAAAFrGUCxA==`, defaultScope(3), true, false},
		//{1208, `LASTINDEXOFN`, `lastIndexOf("cafe bebe dead beef cafe bebe", "dead", 13) == 10`, `AwkAAAAAAAACCQAEuAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARkZWFkAAAAAAAAAAANAAAAAAAAAAAKepNV2A==`, defaultScope(3), true, false},
		//{1208, `LASTINDEXOFN`, `lastIndexOf("cafe bebe dead beef cafe bebe", "dead", 11) == 10`, `AwkAAAAAAAACCQAEuAAAAAMCAAAAHWNhZmUgYmViZSBkZWFkIGJlZWYgY2FmZSBiZWJlAgAAAARkZWFkAAAAAAAAAAALAAAAAAAAAAAKcxKwfA==`, defaultScope(3), true, false},
	} {
		r, err := reader.NewReaderFromBase64(test.script)
		require.NoError(t, err)

		script, err := BuildScript(r)
		require.NoError(t, err)

		rs, err := Eval(script.Verifier, test.scope)
		if test.error {
			assert.Error(t, err, "No error in "+test.name)
		} else {
			assert.NoError(t, err, "Unexpected error in: "+test.name)
		}
		assert.Equal(t, test.result, rs, fmt.Sprintf("func name: %s, code: %d, text: %s", test.name, test.code, test.text))
	}
}

func TestOverlapping(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let ref = 999
func g(a: Int) = ref
func f(ref: Int) = g(ref)
f(1) == 999
`

	s := "AwQAAAADcmVmAAAAAAAAAAPnCgEAAAABZwAAAAEAAAABYQUAAAADcmVmCgEAAAABZgAAAAEAAAADcmVmCQEAAAABZwAAAAEFAAAAA3JlZgkAAAAAAAACCQEAAAABZgAAAAEAAAAAAAAAAAEAAAAAAAAAA+fjknmW"

	r, err := reader.NewReaderFromBase64(s)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	tx := byte_helpers.TransferV2.Transaction.Clone()
	obj, err := NewVariablesFromTransaction(proto.MainNetScheme, tx)
	require.NoError(t, err)
	rs, err := script.Verify(proto.MainNetScheme, mockstate.State{}, obj, nil, nil)
	require.NoError(t, err)
	require.Equal(t, true, rs)
}

func TestUserFunctionsInExpression(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func g() = 5

g() == 5
`
	b64 := `AwoBAAAAAWcAAAAAAAAAAAAAAAAFCQAAAAAAAAIJAQAAAAFnAAAAAAAAAAAAAAAABWtYRqw=`

	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	tx := byte_helpers.TransferV2.Transaction.Clone()
	obj, err := NewVariablesFromTransaction(proto.MainNetScheme, tx)
	require.NoError(t, err)
	rs, err := script.Verify(proto.MainNetScheme, mockstate.State{}, obj, nil, nil)
	require.NoError(t, err)
	require.Equal(t, true, rs)
}

// variables refers to each other in the same scope
func TestRerefOnEachOther(t *testing.T) {
	/*
	   let x = 5;
	   let y = x;
	   let x = y;
	*/

	tree := &Block{
		Let: &LetExpr{
			Name:  "x",
			Value: NewLong(5),
		},
		Body: &Block{
			Let: &LetExpr{
				Name:  "y",
				Value: &RefExpr{Name: "x"},
			},
			Body: &Block{
				Let: &LetExpr{
					Name:  "x",
					Value: &RefExpr{Name: "y"},
				},
				Body: &RefExpr{Name: "x"},
			},
		},
	}

	rs, err := tree.Evaluate(NewScope(3, proto.MainNetScheme, nil))
	require.NoError(t, err)
	require.Equal(t, NewLong(5), rs)
}

func TestSimpleFuncEvaluate(t *testing.T) {
	tree := &FunctionCall{
		Name: "1206",
		Argc: 1,
		Argv: Params(NewString("12345")),
	}

	s := NewScope(3, proto.MainNetScheme, nil)

	rs, err := tree.Evaluate(s)
	require.NoError(t, err)
	require.Equal(t, NewLong(12345), rs)
}

func TestDataFunctions(t *testing.T) {
	secret, public, err := crypto.GenerateKeyPair([]byte(seed))
	require.NoError(t, err)
	data := proto.NewUnsignedData(public, 10000, 1544715621)

	require.NoError(t, data.AppendEntry(&proto.IntegerDataEntry{
		Key:   "integer",
		Value: 100500,
	}))
	require.NoError(t, data.AppendEntry(&proto.BooleanDataEntry{
		Key:   "boolean",
		Value: true,
	}))
	require.NoError(t, data.AppendEntry(&proto.BinaryDataEntry{
		Key:   "binary",
		Value: []byte("hello"),
	}))
	require.NoError(t, data.AppendEntry(&proto.StringDataEntry{
		Key:   "string",
		Value: "world",
	}))

	require.NoError(t, data.Sign(secret))

	vars, err := NewVariablesFromTransaction(proto.MainNetScheme, data)
	require.NoError(t, err)

	scope := NewScope(2, proto.MainNetScheme, mockstate.State{})
	scope.SetHeight(100500)
	scope.SetTransaction(vars)
	for _, test := range []struct {
		FuncCode int
		FuncName string
		Code     string
		Base64   string
		Result   bool
	}{
		{1040, "DATA_LONG_FROM_ARRAY", `match tx {case t: DataTransaction => getInteger(t.data, "integer") == 100500 case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQAEEAAAAAIIBQAAAAF0AAAABGRhdGECAAAAB2ludGVnZXIAAAAAAAABiJQHp2oJqg==`, true},
		{1041, "DATA_BOOLEAN_FROM_ARRAY", `match tx {case t: DataTransaction => getBoolean(t.data, "boolean") == true case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQAEEQAAAAIIBQAAAAF0AAAABGRhdGECAAAAB2Jvb2xlYW4GBw5ToUs=`, true},
		{1042, "DATA_BYTES_FROM_ARRAY", `match tx {case t: DataTransaction => getBinary(t.data, "binary") == base58'Cn8eVZg' case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQAEEgAAAAIIBQAAAAF0AAAABGRhdGECAAAABmJpbmFyeQEAAAAFaGVsbG8HDogmeQ==`, true},
		{1043, "DATA_STRING_FROM_ARRAY", `match tx {case t: DataTransaction => getString(t.data, "string") == "world" case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQAEEwAAAAIIBQAAAAF0AAAABGRhdGECAAAABnN0cmluZwIAAAAFd29ybGQH7+G/UA==`, true},

		{0, "UserDataIntegerFromArrayByIndex", `match tx {case t : DataTransaction => getInteger(t.data, 0) == 100500 case _ => true}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAKZ2V0SW50ZWdlcgAAAAIIBQAAAAF0AAAABGRhdGEAAAAAAAAAAAAAAAAAAAABiJQGwLSDPw==`, true},
		{0, "UserDataBooleanFromArrayByIndex", `match tx {case t : DataTransaction => getBoolean(t.data, 1) == true case _ => true}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAKZ2V0Qm9vbGVhbgAAAAIIBQAAAAF0AAAABGRhdGEAAAAAAAAAAAEGBk7sdw4=`, true},
		{0, "UserDataBinaryFromArrayByIndex", `match tx {case t : DataTransaction => getBinary(t.data, 2) == base58'Cn8eVZg' case _ => true}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAJZ2V0QmluYXJ5AAAAAggFAAAAAXQAAAAEZGF0YQAAAAAAAAAAAgEAAAAFaGVsbG8GRLZgkQ==`, true},
		{0, "UserDataStringFromArrayByIndex", `match tx {case t : DataTransaction => getString(t.data, 3) == "world" case _ => false}`, `AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAAAAAAACCQEAAAAJZ2V0U3RyaW5nAAAAAggFAAAAAXQAAAAEZGF0YQAAAAAAAAAAAwIAAAAFd29ybGQHKKHsFw==`, true},
	} {
		reader, err := reader.NewReaderFromBase64(test.Base64)
		require.NoError(t, err)

		script, err := BuildScript(reader)
		require.NoError(t, err)

		rs, err := Eval(script.Verifier, scope)
		assert.NoError(t, err)
		assert.Equal(t, test.Result, rs, fmt.Sprintf("func name: %s, code: %d, text: %s", test.FuncName, test.FuncCode, test.Code))
	}
}

func Benchmark_Verify(b *testing.B) {
	b.ReportAllocs()
	t := newTransferTransaction()
	body, err := t.BodyMarshalBinary()
	if err != nil {
		b.Fail()
	}
	sig, err := crypto.NewSignatureFromBytes(t.Proofs.Proofs[0])
	if err != nil {
		b.Fail()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs := crypto.Verify(t.SenderPK, sig, body)
		if !rs {
			b.Fail()
		}
	}
}

func TestEvaluateBlockV3False(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}
func fn(name: String) = {
    name
}
fn("bbb") == "aaa"
`

	b64 := "AwoBAAAAAmZuAAAAAQAAAARuYW1lBQAAAARuYW1lCQAAAAAAAAIJAQAAAAJmbgAAAAECAAAAA2JiYgIAAAADYWFhbCbxUQ=="
	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	s := defaultScope(3)

	rs, err := Eval(script.Verifier, s)
	require.NoError(t, err)
	require.False(t, rs, rs)
}

func TestEvaluateBlockV3True(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
let zz = "ccc"

func fn(name: String) = zz

fn("abc") == "ccc"
`

	b64 := "AwQAAAACenoCAAAAA2NjYwoBAAAAAmZuAAAAAQAAAARuYW1lBQAAAAJ6egkAAAAAAAACCQEAAAACZm4AAAABAgAAAANhYmMCAAAAA2NjYyBIzew="
	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	s := defaultScope(3)

	rs, err := Eval(script.Verifier, s)
	require.NoError(t, err)
	require.True(t, rs, rs)
}

func invokeTxWithFunctionCall(tx *proto.InvokeScriptV1, fc *proto.FunctionCall) {
	tx.FunctionCall = *fc
}

func TestDappCallable(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func getPreviousAnswer(address: String) = {
  address
}

@Callable(i)
func tellme(question: String) = {
    let answer = getPreviousAnswer(question)

    WriteSet([
        DataEntry(answer + "_q", question),
        DataEntry(answer + "_a", answer)
        ])
}

@Verifier(tx)
func verify() = {
    getPreviousAnswer(toString(tx.sender)) == "1"
}

`
	b64 := "AAIDAAAAAAAAAAAAAAABAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEAAAAHYWRkcmVzcwUAAAAHYWRkcmVzcwAAAAEAAAABaQEAAAAGdGVsbG1lAAAAAQAAAAhxdWVzdGlvbgQAAAAGYW5zd2VyCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAABBQAAAAhxdWVzdGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9xBQAAAAhxdWVzdGlvbgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9hBQAAAAZhbnN3ZXIFAAAAA25pbAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAAAAAAAAgkBAAAAEWdldFByZXZpb3VzQW5zd2VyAAAAAQkABCUAAAABCAUAAAACdHgAAAAGc2VuZGVyAgAAAAEx7gicPQ=="

	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	tx := byte_helpers.InvokeScriptV1.Transaction.Clone()
	invokeTxWithFunctionCall(tx, &proto.FunctionCall{
		Default:   false,
		Name:      "tellme",
		Arguments: proto.Arguments{proto.NewStringArgument("abc")},
	})

	rs, err := script.CallFunction(proto.MainNetScheme, mockstate.State{}, tx, nil, nil)
	require.NoError(t, err)
	require.Equal(t,
		&proto.ScriptResult{
			Writes: []proto.DataEntry{
				&proto.StringDataEntry{Key: "abc_q", Value: "abc"},
				&proto.StringDataEntry{Key: "abc_a", Value: "abc"},
			},
		},
		rs,
	)
}

func TestDappDefaultFunc(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func getPreviousAnswer(address: String) = {
  address
}

@Callable(i)
func tellme(question: String) = {
    let answer = getPreviousAnswer(question)

    WriteSet([
        DataEntry(answer + "_q", question),
        DataEntry(answer + "_a", answer)
        ])
}

@Callable(invocation)
func default() = {
    let sender0 = invocation.caller.bytes
    WriteSet([DataEntry("a", "b"), DataEntry("sender", sender0)])
}

@Verifier(tx)
func verify() = {
    getPreviousAnswer(toString(tx.sender)) == "1"
}
`
	b64 := "AAIDAAAAAAAAAAAAAAABAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEAAAAHYWRkcmVzcwUAAAAHYWRkcmVzcwAAAAIAAAABaQEAAAAGdGVsbG1lAAAAAQAAAAhxdWVzdGlvbgQAAAAGYW5zd2VyCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAABBQAAAAhxdWVzdGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9xBQAAAAhxdWVzdGlvbgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9hBQAAAAZhbnN3ZXIFAAAAA25pbAAAAAppbnZvY2F0aW9uAQAAAAdkZWZhdWx0AAAAAAQAAAAHc2VuZGVyMAgIBQAAAAppbnZvY2F0aW9uAAAABmNhbGxlcgAAAAVieXRlcwkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAABYQIAAAABYgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAGc2VuZGVyBQAAAAdzZW5kZXIwBQAAAANuaWwAAAABAAAAAnR4AQAAAAZ2ZXJpZnkAAAAACQAAAAAAAAIJAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEJAAQlAAAAAQgFAAAAAnR4AAAABnNlbmRlcgIAAAABMcP91gY="
	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	tx := byte_helpers.InvokeScriptV1.Transaction.Clone()
	invokeTxWithFunctionCall(tx, &proto.FunctionCall{
		Default:   true,
		Name:      "",
		Arguments: proto.Arguments{},
	})

	addr, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, tx.SenderPK)

	rs, err := script.CallFunction(proto.MainNetScheme, mockstate.State{}, tx, nil, nil)
	require.NoError(t, err)
	require.Equal(t,
		&proto.ScriptResult{
			Writes: []proto.DataEntry{
				&proto.StringDataEntry{Key: "a", Value: "b"},
				&proto.BinaryDataEntry{Key: "sender", Value: addr.Bytes()},
			},
		},
		rs,
	)
}

func TestDappVerify(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func getPreviousAnswer(address: String) = {
 address
}

@Callable(i)
func tellme(question: String) = {
   let answer = getPreviousAnswer(question)

   WriteSet([
       DataEntry(answer + "_q", question),
       DataEntry(answer + "_a", answer)
       ])
}

@Callable(invocation)
func default() = {
   let sender0 = invocation.caller.bytes
   WriteSet([DataEntry("a", "b"), DataEntry("sender", sender0)])
}

@Verifier(tx)
func verify() = {
   getPreviousAnswer(toString(tx.sender)) == "1"
}
`
	b64 := "AAIDAAAAAAAAAAAAAAABAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEAAAAHYWRkcmVzcwUAAAAHYWRkcmVzcwAAAAIAAAABaQEAAAAGdGVsbG1lAAAAAQAAAAhxdWVzdGlvbgQAAAAGYW5zd2VyCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAABBQAAAAhxdWVzdGlvbgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9xBQAAAAhxdWVzdGlvbgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgkAASwAAAACBQAAAAZhbnN3ZXICAAAAAl9hBQAAAAZhbnN3ZXIFAAAAA25pbAAAAAppbnZvY2F0aW9uAQAAAAdkZWZhdWx0AAAAAAQAAAAHc2VuZGVyMAgIBQAAAAppbnZvY2F0aW9uAAAABmNhbGxlcgAAAAVieXRlcwkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAABYQIAAAABYgkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAGc2VuZGVyBQAAAAdzZW5kZXIwBQAAAANuaWwAAAABAAAAAnR4AQAAAAZ2ZXJpZnkAAAAACQAAAAAAAAIJAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAEJAAQlAAAAAQgFAAAAAnR4AAAABnNlbmRlcgIAAAABMcP91gY="
	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	tx := byte_helpers.TransferV2.Transaction.Clone()
	obj, err := NewVariablesFromTransaction(proto.MainNetScheme, tx)
	require.NoError(t, err)
	rs, err := script.Verify(proto.MainNetScheme, mockstate.State{}, obj, nil, nil)
	require.NoError(t, err)
	require.Equal(t, false, rs)
}

func TestDappVerifySuccessful(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let x = 100500

func getPreviousAnswer() = {
 x
}

@Verifier(tx)
func verify() = {
   getPreviousAnswer() == 100500
}
`
	b64 := "AAIDAAAAAAAAAAAAAAACAAAAAAF4AAAAAAAAAYiUAQAAABFnZXRQcmV2aW91c0Fuc3dlcgAAAAAFAAAAAXgAAAAAAAAAAQAAAAJ0eAEAAAAGdmVyaWZ5AAAAAAkAAAAAAAACCQEAAAARZ2V0UHJldmlvdXNBbnN3ZXIAAAAAAAAAAAAAAYiUa4pU5Q=="
	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	tx := byte_helpers.TransferV2.Transaction.Clone()
	obj, err := NewVariablesFromTransaction(proto.MainNetScheme, tx)
	require.NoError(t, err)
	rs, err := script.Verify(proto.MainNetScheme, mockstate.State{}, obj, nil, nil)
	require.NoError(t, err)
	require.Equal(t, true, rs)
}

func TestTransferSet(t *testing.T) {
	_ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func tellme(question: String) = {
    TransferSet([ScriptTransfer(i.caller, 100, unit)])
}`

	b64 := "AAIDAAAAAAAAAAAAAAAAAAAAAQAAAAFpAQAAAAZ0ZWxsbWUAAAABAAAACHF1ZXN0aW9uCQEAAAALVHJhbnNmZXJTZXQAAAABCQAETAAAAAIJAQAAAA5TY3JpcHRUcmFuc2ZlcgAAAAMIBQAAAAFpAAAABmNhbGxlcgAAAAAAAAAAZAUAAAAEdW5pdAUAAAADbmlsAAAAAH5a2L0="
	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	tx := byte_helpers.InvokeScriptV1.Transaction.Clone()
	invokeTxWithFunctionCall(tx, &proto.FunctionCall{
		Default:   false,
		Name:      "tellme",
		Arguments: proto.Arguments{proto.NewIntegerArgument(100500)},
	})

	addr, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, tx.SenderPK)

	rs, err := script.CallFunction(proto.MainNetScheme, mockstate.State{}, tx, nil, nil)
	require.NoError(t, err)
	scriptTransfer := proto.ScriptResultTransfer{
		Recipient: proto.NewRecipientFromAddress(addr),
		Amount:    100,
		Asset:     proto.OptionalAsset{Present: false},
	}
	require.NoError(t, err)
	require.Equal(t,
		&proto.ScriptResult{
			Transfers: []proto.ScriptResultTransfer{scriptTransfer},
		},
		rs,
	)
}

func TestScriptResult(t *testing.T) {
	var _ = `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func tellme(question: String) = {
    ScriptResult(
        WriteSet([DataEntry("key", 100)]),
        TransferSet([ScriptTransfer(i.caller, 100500, unit)])
    )
}
`
	b64 := "AAIDAAAAAAAAAAAAAAAAAAAAAQAAAAFpAQAAAAZ0ZWxsbWUAAAABAAAACHF1ZXN0aW9uCQEAAAAMU2NyaXB0UmVzdWx0AAAAAgkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAADa2V5AAAAAAAAAABkBQAAAANuaWwJAQAAAAtUcmFuc2ZlclNldAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWkAAAAGY2FsbGVyAAAAAAAAAYiUBQAAAAR1bml0BQAAAANuaWwAAAAARKRntw=="
	r, err := reader.NewReaderFromBase64(b64)
	require.NoError(t, err)

	script, err := BuildScript(r)
	require.NoError(t, err)

	tx := byte_helpers.InvokeScriptV1.Transaction.Clone()
	invokeTxWithFunctionCall(tx, &proto.FunctionCall{
		Default:   false,
		Name:      "tellme",
		Arguments: proto.Arguments{proto.NewIntegerArgument(100)},
	})

	addr, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, tx.SenderPK)

	rs, err := script.CallFunction(proto.MainNetScheme, mockstate.State{}, tx, nil, nil)
	require.NoError(t, err)
	scriptTransfer := proto.ScriptResultTransfer{
		Recipient: proto.NewRecipientFromAddress(addr),
		Amount:    100500,
		Asset:     proto.OptionalAsset{Present: false},
	}
	require.Equal(t,
		&proto.ScriptResult{
			Writes:    []proto.DataEntry{&proto.IntegerDataEntry{Key: "key", Value: 100}},
			Transfers: []proto.ScriptResultTransfer{scriptTransfer},
		},
		rs,
	)
}

func TestMatchOverwrite(t *testing.T) {
	/*
		{-# STDLIB_VERSION 1 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		match tx {
		    case dt: DataTransaction =>
		    let a = extract(getInteger(dt.sender, "a"))
		    let x = if a == 0 then {
		        match getInteger(dt.sender, "x") {
		            case i: Int => i
		            case _ => 0
		        }
		    } else {
		        0
		    }
		    let xx = match getInteger(dt.data, "x") {
		        case i: Int => i
		        case _ => 0
		    }
		    x + xx == 3
		    case _ => false
		}
	*/
	code := "AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAACZHQFAAAAByRtYXRjaDAEAAAAAWEJAQAAAAdleHRyYWN0AAAAAQkABBoAAAACCAUAAAACZHQAAAAGc2VuZGVyAgAAAAFhBAAAAAF4AwkAAAAAAAACBQAAAAFhAAAAAAAAAAAABAAAAAckbWF0Y2gxCQAEGgAAAAIIBQAAAAJkdAAAAAZzZW5kZXICAAAAAXgDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAA0ludAQAAAABaQUAAAAHJG1hdGNoMQUAAAABaQAAAAAAAAAAAAAAAAAAAAAAAAQAAAACeHgEAAAAByRtYXRjaDEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAAAXgDCQAAAQAAAAIFAAAAByRtYXRjaDECAAAAA0ludAQAAAABaQUAAAAHJG1hdGNoMQUAAAABaQAAAAAAAAAAAAkAAAAAAAACCQAAZAAAAAIFAAAAAXgFAAAAAnh4AAAAAAAAAAADB2NbtyA="
	r, err := reader.NewReaderFromBase64(code)
	require.NoError(t, err)

	pk := crypto.PublicKey{}
	sig := crypto.Signature{}
	tx := proto.NewUnsignedData(pk, 1400000, 1539113093702)
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "x", Value: 2})
	tx.ID = &crypto.Digest{}
	tx.Proofs = proto.NewProofs()
	tx.Proofs.Proofs = append(tx.Proofs.Proofs, sig[:])

	tv, err := NewVariablesFromTransaction(proto.TestNetScheme, tx)
	require.NoError(t, err)

	dataEntries := map[string]proto.DataEntry{
		"a": &proto.IntegerDataEntry{Key: "a", Value: 0},
		"x": &proto.IntegerDataEntry{Key: "x", Value: 1},
	}
	state := mockstate.State{
		DataEntries: dataEntries,
	}
	scope := NewScope(1, proto.TestNetScheme, state)
	scope.SetTransaction(tv)
	scope.SetHeight(368430)

	script, err := BuildScript(r)
	require.NoError(t, err)

	rs, err := Eval(script.Verifier, scope)
	require.NoError(t, err)
	require.True(t, rs, rs)
}

func TestFailSript1(t *testing.T) {
	script := "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"
	r, err := reader.NewReaderFromBase64(script)
	require.NoError(t, err)

	pk, err := crypto.NewPublicKeyFromBase58("5ydncg624xM6LmJKWJ26iZoy7XBdGx9JxcgqKMNhJPaz")
	require.NoError(t, err)
	sig, err := crypto.NewSignatureFromBase58("JR8MP7AFSm5JY5UKYRHtTjJX7sVUEV7rnQaKAvLB7RjV9Ze8Cm1KeYQQiuYBp8gJZrcQqrC6gHiyYheKpheHVgk")
	require.NoError(t, err)
	id, err := crypto.NewDigestFromBase58("Eg5yoFwXcBrq3ik4JvbbhSg429b6HT2qdXTURAUMBTh9")
	require.NoError(t, err)

	tx := proto.NewUnsignedData(pk, 1400000, 1539113093702)
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "command", Value: 1})
	tx.Entries = append(tx.Entries, &proto.IntegerDataEntry{Key: "gameState", Value: 1})
	tx.Entries = append(tx.Entries, &proto.StringDataEntry{Key: "player1", Value: ""})
	tx.Entries = append(tx.Entries, &proto.StringDataEntry{Key: "player2", Value: ""})
	tx.ID = &id
	tx.Proofs = proto.NewProofs()
	tx.Proofs.Proofs = append(tx.Proofs.Proofs, sig[:])

	tv, err := NewVariablesFromTransaction(proto.TestNetScheme, tx)
	require.NoError(t, err)

	dataEntries := map[string]proto.DataEntry{
		"gameState": &proto.IntegerDataEntry{Key: "gameState", Value: 0},
	}
	state := mockstate.State{
		DataEntries: dataEntries,
	}
	scope := NewScope(1, proto.TestNetScheme, state)
	scope.SetTransaction(tv)
	scope.SetHeight(368430)

	scr, err := BuildScript(r)
	require.NoError(t, err)

	rs, err := Eval(scr.Verifier, scope)
	require.NoError(t, err)
	require.True(t, rs, rs)
}

func TestFailSript2(t *testing.T) {
	/* Script:
	{-# STDLIB_VERSION 2 #-}
	{-# CONTENT_TYPE EXPRESSION #-}
	let admin = Address(base58'3PEyLyxu4yGJAEmuVRy3G4FvEBUYV6ykQWF')
	match tx {
	    case tx: MassTransferTransaction|TransferTransaction =>
	        if tx.sender == admin then
	            true
	        else
	            throw("You're not allowed to transfer this asset")
	    case tx: BurnTransaction =>
	        throw("You're not allowed to burn this asset")
	    case tx: ExchangeTransaction =>
	        let amountAsset = match tx.sellOrder.assetPair.amountAsset {
	            case b: ByteVector => b
	            case _ => throw("Incorrect asset pair")
	        }
	        let priceAsset = match tx.sellOrder.assetPair.priceAsset {
	            case b: ByteVector => b
	            case _ => throw("Incorrect asset pair")
	        }
	        let pair1 = toBase58String(amountAsset) + "/" + toBase58String(priceAsset)
	        let pair2 = toBase58String(priceAsset) + "/" + toBase58String(amountAsset)
	        let checkPair1 = match getBoolean(admin, pair1) {
	            case b: Boolean => b
	            case _ => false
	        }
	        let checkPair2 = match getBoolean(admin, pair2) {
	            case b: Boolean => b
	            case _ => false
	        }
	        let status = match getString(admin, "status") {
	            case s: String => s
	            case _ => throw("The contest has not started yet")
	        }
	        if status == "finished" then
	            throw("The contest has already finished")
	        else
	            if status != "started" then
	                throw("The contest has not started yet")
	            else
	                if
	                    if checkPair1 then
	                        true
	                    else
	                        checkPair2
	                then
	                    true
	                else
	                    throw("Incorrect asset pair")
	   case tx: ReissueTransaction|SetAssetScriptTransaction =>
	       true
	   case _ =>
	       false
	}
	*/
	script := "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"
	transaction := `{"senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy","amount": 100000000,"fee": 1100000,"type": 7,"version": 2,"sellMatcherFee": 1100000,"sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3","feeAssetId": null,"proofs": ["DGxkASjpPaKxu8bAv3PJpF9hJ9KAiLsB7bLBTEZXYcWmmc65pHiq5ymJNAazRM2aoLCeTLXXNda5hR9LZNayB69"],"price": 790000,"id": "5aHKTDvWdVWmo9MPDPoYX83x6hyLJ5ji4eopmoUxELR2","order2": {"version": 2,"id": "CzBrJkpaWz2AHnT3U8baY3eTfRdymuC7dEqiGpas68tD","sender": "3PEjQH31dP2ipvrkouUs12ynKShpBcRQFAT","senderPublicKey": "BVtDAjf1MmUdPW2yRHEBiSP5yy7EnxzKgQWpajQM8FCx","matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy","assetPair": {"amountAsset": "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF","priceAsset": "CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr"},"orderType": "sell","amount": 100000000,"price": 790000,"timestamp": 1557995955609,"expiration": 1560501555609,"matcherFee": 1100000,"signature": "3Aw94WkF4PUeard435jtJTZLESRFMBuxYRYVVf3GrG48aAxLhbvcXdwsrtALLQ3LYbdNdhR1NUUzdMinU8pLiwWc","proofs": ["3Aw94WkF4PUeard435jtJTZLESRFMBuxYRYVVf3GrG48aAxLhbvcXdwsrtALLQ3LYbdNdhR1NUUzdMinU8pLiwWc"]},"order1": {"version": 2,"id": "APLf7qDhU5puSa5h1KChNBobF8VKoy37PcP7BnhoSPvi","sender": "3PEyLyxu4yGJAEmuVRy3G4FvEBUYV6ykQWF","senderPublicKey": "28sBbJ7pHNG4VFrvNN43sNsdWYyrTFVAwd98W892mxBQ","matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy","assetPair": {"amountAsset": "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF","priceAsset": "CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr"},"orderType": "buy","amount": 100000000,"price": 790000,"timestamp": 1557995158094,"expiration": 1560500758093,"matcherFee": 1100000,"signature": "5zUuSSJyv5NU11RPa91fpQaCXR3xvR1ctjQrfxnNREFhMmbXfACzhfFgV18rdvrvm4X3p3iYK3fxS1TXwgSV5m83","proofs": ["5zUuSSJyv5NU11RPa91fpQaCXR3xvR1ctjQrfxnNREFhMmbXfACzhfFgV18rdvrvm4X3p3iYK3fxS1TXwgSV5m83"]},"buyMatcherFee": 1100000,"timestamp": 1557995955923,"height": 1528811}`
	r, err := reader.NewReaderFromBase64(script)
	require.NoError(t, err)

	tx := new(proto.ExchangeV2)
	err = json.Unmarshal([]byte(transaction), tx)
	require.NoError(t, err)

	tv, err := NewVariablesFromTransaction(proto.MainNetScheme, tx)
	require.NoError(t, err)

	dataEntries := map[string]proto.DataEntry{
		"status": &proto.StringDataEntry{Key: "status", Value: "started"},
		"D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF/CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr": &proto.BooleanDataEntry{Key: "D796K7uVAeSPJcv29BN1KCuzrc6h7bAN1MSKPnrPPMfF/CAWKh6suz3jKw6PhzEh5FDCWLvLFJ6BZEpmxv6oZQSzr", Value: true},
	}
	state := mockstate.State{
		DataEntries: dataEntries,
	}
	scope := NewScope(2, proto.MainNetScheme, state)
	scope.SetTransaction(tv)
	scope.SetHeight(368430)

	scr, err := BuildScript(r)
	require.NoError(t, err)

	rs, err := Eval(scr.Verifier, scope)
	require.NoError(t, err)
	require.True(t, rs, rs)
}
