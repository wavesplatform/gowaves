package evaluate

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/parser"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
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

func defaultScope() Scope {
	variables := VariablesV3()

	t := newTransferTransaction()
	vars, err := NewVariablesFromTransaction(proto.MainNetScheme, t)
	if err != nil {
		panic(err)
	}
	variables["tx"] = NewObject(vars)
	variables["height"] = NewLong(5)

	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, t.SenderPK)
	if err != nil {
		panic(err)
	}

	am := mockstate.MockAccount{
		Assets: map[string]uint64{"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD": 5},
	}

	s := mockstate.MockStateImpl{
		//TransactionsHeightByID: map[string]uint64{},
		//AssetsByID: map[string]uint64{addr.String() + "BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD": 5},
		Accounts: map[string]mockstate.Account{addr.String(): &am},
	}

	return NewScope(proto.MainNetScheme, s, FunctionsV3(), variables)
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
		Name   string
		Base64 string
		Result bool
	}{
		{`let x = 5; 6 > 4`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGAAAAAAAAAAAEYSW6XA==`, true},
		{`let x = 5; 6 > x`, `AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGBQAAAAF4Gh24hw==`, true},
		{`let x = 5; 6 >= x`, `AQQAAAABeAAAAAAAAAAABQkAAGcAAAACAAAAAAAAAAAGBQAAAAF4jlxXHA==`, true},
		{`true`, `AQa3b8tH`, true},
		{`false`, `AQfeYll6`, false},
		{`let x =  throw(); true`, `AQQAAAABeAkBAAAABXRocm93AAAAAAa7bgf4`, true},
		{`let x =  throw();true || x`, `AQQAAAABeAkBAAAABXRocm93AAAAAAMGBgUAAAABeKRnLds=`, true},
		{`tx.id == base58''`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAAJBtD70=`, false},
		{`tx.id == base58'CqjGMbrd5bFmLAv2mUSdphEJSgVWkWa6ZtcMkKmgH2ax'`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAIK/sOVMfQLb6FHT+QbJpYq4m7jlQoC3GPCMpxfHPeT5F5CUKdw==`, true},
		{`let x = tx.id == base58'a';true`, `AQQAAAABeAkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAASEGjR0kcA==`, true},
		{`tx.proofs[0] == base58'5W7hjPpgmmhxevCt4A7y9F8oNJ4V9w2g8jhQgx2qGmBTNsP1p1MpQeKF3cvZULwJ7vQthZfSx2BhL6TWkHSVLzvq'`, `AQkAAAAAAAACCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAABAAAAQOEtF8V5p+9JHReO90FmBf+yKZW1lLJGBsnkZww94TJ8bNcxWIKfohMXm4BsKKIBUTXLaS6Vcgyw1UTNN5iICQ719Fxf`, true},
		{longScript, `AQQAAAAHJG1hdGNoMAUAAAACdHgDAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNFeGNoYW5nZVRyYW5zYWN0aW9uBgMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24GCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB6Ilvok=`, true},
		{`match transactionById(tx.id) {case  t: Unit => true case _ => false }`, `AQQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAAAXQFAAAAByRtYXRjaDAGB1+iIek=`, true},
		//TODO: The RIDE compiler is broken, test after repair
		// {`Ceiling() == CEILING`, ``, true},
		// {`Floor() == FLOOR`, ``, true},
		// {`HalfEven() == HALFEVEN`, ``, true},
		{`Down() == DOWN`, `AgkAAAAAAAACCQEAAAAERG93bgAAAAAFAAAABERPV052K6LQ`, true},
		{`Up() == UP`, `AwkAAAAAAAACCQEAAAACVXAAAAAABQAAAAJVUPGUxeg=`, true},
		{`HalfUp() == HALFUP`, `AwkAAAAAAAACCQEAAAAGSGFsZlVwAAAAAAUAAAAGSEFMRlVQbUfpTQ==`, true},
		//TODO: Test after implementation of this rounding mode in decimal library {`HalfDown() == HALFDOWN`, `AgkAAAAAAAACCQEAAAAERG93bgAAAAAFAAAABERPV052K6LQ`, true},
		{hashes, `AwQAAAACYTAJAAAAAAAAAgkBAAAABU5vQWxnAAAAAAUAAAAFTk9BTEcEAAAAAmExCQAAAAAAAAIJAQAAAANNZDUAAAAABQAAAANNRDUEAAAAAmEyCQAAAAAAAAIJAQAAAARTaGExAAAAAAUAAAAEU0hBMQQAAAACYTMJAAAAAAAAAgkBAAAABlNoYTIyNAAAAAAFAAAABlNIQTIyNAQAAAACYTQJAAAAAAAAAgkBAAAABlNoYTI1NgAAAAAFAAAABlNIQTI1NgQAAAACYTUJAAAAAAAAAgkBAAAABlNoYTM4NAAAAAAFAAAABlNIQTM4NAQAAAACYTYJAAAAAAAAAgkBAAAABlNoYTUxMgAAAAAFAAAABlNIQTUxMgQAAAACYTcJAAAAAAAAAgkBAAAAB1NoYTMyMjQAAAAABQAAAAdTSEEzMjI0BAAAAAJhOAkAAAAAAAACCQEAAAAHU2hhMzI1NgAAAAAFAAAAB1NIQTMyNTYEAAAAAmE5CQAAAAAAAAIJAQAAAAdTaGEzMzg0AAAAAAUAAAAHU0hBMzM4NAQAAAADYTEwCQAAAAAAAAIJAQAAAAdTaGEzNTEyAAAAAAUAAAAHU0hBMzUxMgMDAwMDAwMDAwMFAAAAAmEwBQAAAAJhMQcFAAAAAmEyBwUAAAACYTMHBQAAAAJhNAcFAAAAAmE1BwUAAAACYTYHBQAAAAJhNwcFAAAAAmE4BwUAAAACYTkHBQAAAANhMTAHRc/wAA==`, true},
	} {
		r, err := reader.NewReaderFromBase64(test.Base64)
		require.NoError(t, err)

		script, err := BuildAst(r)
		require.NoError(t, err)

		rs, err := Eval(script.Verifier, defaultScope())
		require.NoError(t, err)
		assert.Equal(t, test.Result, rs, fmt.Sprintf("script: %s", test.Name))
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

	s := defaultScope()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r, _ := reader.NewReaderFromBase64(base64)
		script, _ := BuildAst(r)
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
		FuncCode int
		FuncName string
		Code     string
		Base64   string
		Result   bool
	}{
		{0, "EQ", `5 == 5`, `AQkAAAAAAAACAAAAAAAAAAAFAAAAAAAAAAAFqWG0Fw==`, true},
		{1,
			"ISINSTANCEOF",
			`match tx {case t : TransferTransaction => true case _  => false}`,
			`AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB5yQ/+k=`, true},
		{2, `THROW`, `true && throw("mess")`, `AQMGCQAAAgAAAAECAAAABG1lc3MH7PDwAQ==`, false},
		{100, `SUM_LONG`, `1 + 1 > 0`, `AQkAAGYAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAABiJjSk`, true},
		{101, `SUB_LONG`, `2 - 1 > 0`, `AQkAAGYAAAACCQAAZQAAAAIAAAAAAAAAAAIAAAAAAAAAAAEAAAAAAAAAAABqsps1`, true},
		{102, `GT_LONG`, `1 > 0`, `AQkAAGYAAAACAAAAAAAAAAABAAAAAAAAAAAAyAIM4w==`, true},
		{103, `GE_LONG`, `1 >= 0`, `AQkAAGcAAAACAAAAAAAAAAABAAAAAAAAAAAAm30DnQ==`, true},
		{104, `MUL_LONG`, `2 * 2>0`, `AQkAAGYAAAACCQAAaAAAAAIAAAAAAAAAAAIAAAAAAAAAAAIAAAAAAAAAAABCMM5o`, true},
		{105, `DIV_LONG`, `4 / 2>0`, `AQkAAGYAAAACCQAAaQAAAAIAAAAAAAAAAAQAAAAAAAAAAAIAAAAAAAAAAAAadVma`, true},
		{106, `MOD_LONG`, `-10 % 6>0`, `AQkAAGYAAAACCQAAagAAAAIA//////////YAAAAAAAAAAAYAAAAAAAAAAAB5rBSH`, true},
		{107, `FRACTION`, `fraction(10, 5, 2)>0`, `AQkAAGYAAAACCQAAawAAAAMAAAAAAAAAAAoAAAAAAAAAAAUAAAAAAAAAAAIAAAAAAAAAAACRyFu2`, true},
		{108, `POW`, `pow(12, 1, 3456, 3, 2, Down()) == 187`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIJAQAAAAREb3duAAAAAAAAAAAAAAAAu9llw2M=`, true},
		{108, `POW`, `pow(12, 1, 3456, 3, 2, UP) == 187`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIFAAAAAlVQAAAAAAAAAAC7FUMwCQ==`, false},
		{108, `POW`, `pow(12, 1, 3456, 3, 2, UP) == 188`, `AwkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAAwAAAAAAAAAAAEAAAAAAAAADYAAAAAAAAAAAAMAAAAAAAAAAAIFAAAAAlVQAAAAAAAAAAC8evjDQQ==`, true},
		{109, `LOG`, `log(16, 0, 2, 0, 0, CEILING) == 4`, `AwkAAAAAAAACCQAAbQAAAAYAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAAAFAAAAB0NFSUxJTkcAAAAAAAAAAARh6Dy6`, true},
		{109, `LOG`, `log(100, 0, 10, 0, 0, CEILING) == 2`, `AwkAAAAAAAACCQAAbQAAAAYAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAFAAAAB0NFSUxJTkcAAAAAAAAAAAJ7Op42`, true},

		{200, `SIZE_BYTES`, `size(base58'abcd') > 0`, `AQkAAGYAAAACCQAAyAAAAAEBAAAAA2QGAgAAAAAAAAAAACMcdM4=`, true},
		{201, `TAKE_BYTES`, `size(take(base58'abcd', 2)) == 2`, `AQkAAAAAAAACCQAAyAAAAAEJAADJAAAAAgEAAAADZAYCAAAAAAAAAAACAAAAAAAAAAACccrCZg==`, true},
		{202, `DROP_BYTES`, `size(drop(base58'abcd', 2)) > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAADKAAAAAgEAAAADZAYCAAAAAAAAAAACAAAAAAAAAAAA+srbUQ==`, true},
		{203, `SUM_BYTES`, `size(base58'ab' + base58'cd') > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAADLAAAAAgEAAAACB5wBAAAAAggSAAAAAAAAAAAAo+LRIA==`, true},

		{300, `SUM_STRING`, `"ab"+"cd" == "abcd"`, `AQkAAAAAAAACCQABLAAAAAICAAAAAmFiAgAAAAJjZAIAAAAEYWJjZMBJvls=`, true},
		{303, `TAKE_STRING`, `take("abcd", 2) == "ab"`, `AQkAAAAAAAACCQABLwAAAAICAAAABGFiY2QAAAAAAAAAAAICAAAAAmFiiXc+oQ==`, true},
		{304, `DROP_STRING`, `drop("abcd", 2) == "cd"`, `AQkAAAAAAAACCQABMAAAAAICAAAABGFiY2QAAAAAAAAAAAICAAAAAmNkZQdjWQ==`, true},
		{305, `SIZE_STRING`, `size("abcd") == 4`, `AQkAAAAAAAACCQABMQAAAAECAAAABGFiY2QAAAAAAAAAAAScZzsq`, true},

		{400, `SIZE_LIST`, `size(tx.proofs) == 1`, `AQkAAAAAAAACCQABkAAAAAEIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAGGGXM4`, true},
		{401, `GET_LIST`, `size(tx.proofs[0]) > 0`, `AQkAAGYAAAACCQAAyAAAAAEJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAAAAAAAAAAAAFF6iVo=`, true},
		{410, `LONG_TO_BYTES`, `toBytes(1) == base58'11111112'`, `AQkAAAAAAAACCQABmgAAAAEAAAAAAAAAAAEBAAAACAAAAAAAAAABm8cc1g==`, true},
		{411, `STRING_TO_BYTES`, `toBytes("привет") == base58'4wUjatAwfVDjaHQVX'`, `AQkAAAAAAAACCQABmwAAAAECAAAADNC/0YDQuNCy0LXRggEAAAAM0L/RgNC40LLQtdGCuUGFxw==`, true},
		{412, `BOOLEAN_TO_BYTES`, `toBytes(true) == base58'2'`, `AQkAAAAAAAACCQABnAAAAAEGAQAAAAEBJRrQbw==`, true},
		{420, `LONG_TO_STRING`, `toString(5) == "5"`, `AQkAAAAAAAACCQABpAAAAAEAAAAAAAAAAAUCAAAAATXPb5tR`, true},
		{421, `BOOLEAN_TO_STRING`, `toString(true) == "true"`, `AQkAAAAAAAACCQABpQAAAAEGAgAAAAR0cnVlL6ZrWg==`, true},

		{500, `SIGVERIFY`, `sigVerify(tx.bodyBytes, tx.proofs[0], base58'14ovLL9a6xbBfftyxGNLKMdbnzGgnaFQjmgUJGdho6nY')`, `AQkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAABAAAAIAD5y2Wf7zxfv7l+9tcWxyLAbktd9nCbdvFMnxmREqV1igWi3A==`, true},
		{501, `KECCAK256`, `keccak256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfUAAAABAQAAAAEhAQAAAAEhKeR77g==`, true},
		{502, `BLAKE256`, `blake2b256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfYAAAABAQAAAAEhAQAAAAEh50D2WA==`, true},
		{503, `SHA256`, `sha256(base58'a') != base58'a'`, `AQkBAAAAAiE9AAAAAgkAAfcAAAABAQAAAAEhAQAAAAEhVojmeg==`, true},
		{504, `RSAVERIFY`, rsaVerify, `AwQAAAACcGsJAAJbAAAAAQIAAAGITUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUFrRGc4bTBiQ0RYN2ZUYkJsSFptK0JaSUhWT2ZDMkk0a2xSYmpTcXdGaS9lQ2RmaEdqWVJZdnUvZnJwU08wTEltMGJlS09VdndhdDZEWTRkRWhOdDJQVzNVZVF2VDJ1ZFJROVZCY3B3YUpsTHJlQ3I4MzdzbjRmYTlVRzlGUUZhR29mU3d3MU85ZUJCandNWGVacjFqT3pSOVJCSXdvTDFUUWtJa1pHYURYUmx0RWFNeHRObnpvdFBmRjN2R0laWnVaWDRDamlpdEhhU0MwemxtUXJFTDNCRHFxb0x3bzNqcThVM1p6OFhVTXlRRWx3dWZHUmJacWRGQ2VpSXMvRW9IaUptOHE4Q1ZFeFJveEIwSC92RTJ1REZLL09YTEdUZ2Z3bkRsckNhL3FHdDlac2I4cmFVU3o5SUlIeDcyWEIra09YVHQvR091Vzd4MmRKdlRKSXFLVHdJREFRQUIEAAAAA21zZwkAAlsAAAABAgAAAFBSRUlpTjJoRFFVeElKVlF6ZGsxelFTcFhjbFJSZWxFeFZXZCtZR1FvT3l4MEtIZHVQekZtY1U4elVXb3NXaUE3YUZsb09XcGxjbEF4UENVPQQAAAADc2lnCQACWwAAAAECAAABWE9YVktKd3RTb2VuUm13aXpQdHBqaDNzQ05tT3BVMXRuWFVueXpsK1BFSTFQOVJ4MjBHa3hrSVhseXNGVDJXZGJQbi9Ic2ZHTXdHSlc3WWhyVmtEWHk0dUFReFV4U2dRb3V2ZlpvcUdTUHAxTnRNOGlWSk9HeUtpZXBnQjNHeFJ6UXNldjJHOElrNDdlTmtFRFZRYTQ3Y3Q5ajE5OFd2bmtmODh5alNrSzBLeFIwNTdNV0FpMjBpcE5MaXJXNFpIREFmMWdpdjY4bW5pS2ZLeHNQV2FoT0EvN0pZa3YxOHN4Y3NJU1FxUlhNOG5HSTFVdVNMdDlFUjdrSXp5QWsybWdQQ2lWbGowaG9QR1V5dG1iaVVxdkVNNFFhSmZDcFIwd1ZPNGYvZm9iNmp3S2tHVDZ3YnRpYSs1eENEN2JFU0lISDhJU0RyZGV4WjAxUXlOUDJyNGVudz09CQAB+AAAAAQFAAAAB1NIQTMyNTYFAAAAA21zZwUAAAADc2lnBQAAAAJwa8wcz28=`, true},

		{600, `TOBASE58`, `toBase58String(base58'a') == "a"`, `AQkAAAAAAAACCQACWAAAAAEBAAAAASECAAAAAWFcT4nY`, true},
		{601, `FROMBASE58`, `fromBase58String("a") == base58'a'`, `AQkAAAAAAAACCQACWQAAAAECAAAAAWEBAAAAASEB1Qmd`, true},
		{602, `TOBASE64`, `toBase64String(base16'544553547465737454455354') == "VEVTVHRlc3RURVNU"`, `AwkAAAAAAAACCQACWgAAAAEBAAAADFRFU1R0ZXN0VEVTVAIAAAAQVkVWVFZIUmxjM1JVUlZOVd6DVfc=`, true},
		{603, `FROMBASE64`, `base16'544553547465737454455354' == fromBase64String("VEVTVHRlc3RURVNU")`, `AwkAAAAAAAACAQAAAAxURVNUdGVzdFRFU1QJAAJbAAAAAQIAAAAQVkVWVFZIUmxjM1JVUlZOVV+c29Q=`, true},
		{604, `TOBASE16`, `toBase16String(base64'VEVTVHRlc3RURVNU') == "544553547465737454455354"`, `AwkAAAAAAAACCQACXAAAAAEBAAAADFRFU1R0ZXN0VEVTVAIAAAAYNTQ0NTUzNTQ3NDY1NzM3NDU0NDU1MzU07NMrMQ==`, true},
		{605, `FROMBASE16`, `fromBase16String("544553547465737454455354") == base64'VEVTVHRlc3RURVNU'`, `AwkAAAAAAAACCQACXQAAAAECAAAAGDU0NDU1MzU0NzQ2NTczNzQ1NDQ1NTM1NAEAAAAMVEVTVHRlc3RURVNUFBEa5A==`, true},

		{700, `CHECKMERKLEPROOF`, merkle, `AwQAAAAIcm9vdEhhc2gBAAAAIHofX5tx3h2d1wP1HzKQvR0sC1TMgS4JACS9DCFQSKGJBAAAAAhsZWFmRGF0YQEAAAAEAAAm+wQAAAALbWVya2xlUHJvb2YBAAAA7gAgUrNnYuq2PvTd5q6UFUTWSxgHUV9lgQ+l5Kzl7oYeNl8BICoQdwIoo+o6i0bkIa8Kr+ntOZK/PbcsZLhWMFLjAkAfACCi4afwBUJKJE3vzTEJyhS+XkzLw5RXowKZicT2OkafFwAgFq3YAfA8LzwPA9OBPR4FI4SoIQJ+WqrWZ7inJqSx87UAIHdjhteVmM/dK9VoqYF8I3avVobzCALbxpZuhN1mGNdqACCVzz1SHW8Dk3VCbwoDCOImqXtoL0eeb7KztjpiSbWVDQEgMmtt2MgZfhKSsrS/fu3D3ZHjDFko5NAXNBjZ6iIrKH0JAAK8AAAAAwUAAAAIcm9vdEhhc2gFAAAAC21lcmtsZVByb29mBQAAAAhsZWFmRGF0YXe8Icg=`, true},

		{1000, `GETTRANSACTIONBYID`, `match transactionById(tx.id) {case  t: Unit => true case _ => false }`, `AQQAAAAHJG1hdGNoMAkAA+gAAAABCAUAAAACdHgAAAACaWQDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAABFVuaXQEAAAAAXQFAAAAByRtYXRjaDAGB1+iIek=`, true},
		{1001, `TRANSACTIONHEIGHTBYID`, `transactionHeightById(base58'aaaa') == 5`, `AQkAAAAAAAACCQAD6QAAAAEBAAAAA2P4ZwAAAAAAAAAABSLhRM4=`, false},
		{1003, `ACCOUNTASSETBALANCE`, `assetBalance(tx.sender, base58'BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD') == 5`, `AQkAAAAAAAACCQAD6wAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIBAAAAIJxQIls8iGUc1935JolBz6bYc37eoPDtScOAM0lTNhY0AAAAAAAAAAAFjp6PBg==`, true},
		{1061, `ADDRESSTOSTRING`, `toString(Address(base58'3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg')) == "3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg"`, `AwkAAAAAAAACCQAEJQAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVcMIZxOsk2Gw5Avd0ztqi+phtb1Bb83MiUCAAAAIzNQMzMzNnJOU1NVOGJEQXFEYjZTNWpOczhESmIyYmZObXBnkXj7Cg==`, true},
		{1061, `ADDRESSTOSTRING`, `toString(Address(base58'3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg')) == "3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpf"`, `AwkAAAAAAAACCQAEJQAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVcMIZxOsk2Gw5Avd0ztqi+phtb1Bb83MiUCAAAAIzNQMzMzNnJOU1NVOGJEQXFEYjZTNWpOczhESmIyYmZObXBmb/6mcg==`, false},
	} {
		r, err := reader.NewReaderFromBase64(test.Base64)
		require.NoError(t, err)

		script, err := BuildAst(r)
		require.NoError(t, err)

		rs, err := Eval(script.Verifier, defaultScope())
		assert.NoError(t, err)
		assert.Equal(t, test.Result, rs, fmt.Sprintf("func name: %s, code: %d, script: %s", test.FuncName, test.FuncCode, test.Code))
	}
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

	variables := VariablesV3()
	variables["tx"] = NewObject(vars)

	scope := NewScope(proto.MainNetScheme, mockstate.MockStateImpl{}, FunctionsV3(), variables)

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

		script, err := BuildAst(reader)
		require.NoError(t, err)

		rs, err := Eval(script.Verifier, scope)
		assert.NoError(t, err)
		assert.Equal(t, test.Result, rs, fmt.Sprintf("func name: %s, code: %d, script: %s", test.FuncName, test.FuncCode, test.Code))
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
