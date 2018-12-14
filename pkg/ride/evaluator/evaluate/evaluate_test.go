package evaluate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/parser"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

const seed = "test test"

func newTransferTransaction(seed string) *proto.TransferV2 {

	js := `{"type":4,"version":2,"id":"CqjGMbrd5bFmLAv2mUSdphEJSgVWkWa6ZtcMkKmgH2ax","proofs":["5W7hjPpgmmhxevCt4A7y9F8oNJ4V9w2g8jhQgx2qGmBTNsP1p1MpQeKF3cvZULwJ7vQthZfSx2BhL6TWkHSVLzvq"],"senderPublicKey":"14ovLL9a6xbBfftyxGNLKMdbnzGgnaFQjmgUJGdho6nY","assetId":null,"feeAssetId":null,"timestamp":1544715621,"amount":15,"fee":10000,"recipient":"3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"}`

	tv2 := &proto.TransferV2{}

	err := json.Unmarshal([]byte(js), tv2)
	if err != nil {
		panic(err)
	}
	return tv2

	//secret, public := crypto.GenerateKeyPair([]byte(seed))
	//
	//addr, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, public)
	//
	//t, err := proto.NewUnsignedTransferV2(
	//	public,
	//	proto.OptionalAsset{},
	//	proto.OptionalAsset{},
	//	uint64(1544715621),
	//	15,
	//	10000,
	//	addr,
	//	"")
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = t.Sign(secret)
	//if err != nil {
	//	panic(err)
	//}
	//return t
}

func defaultScope() Scope {
	predefObject := make(map[string]Expr)
	t := newTransferTransaction("test test")

	vars, err := NewVariablesFromTransaction(proto.MainNetScheme, t)
	if err != nil {
		panic(err)
	}

	js, _ := json.Marshal(t)
	fmt.Println(string(js))

	predefObject["tx"] = NewObject(vars)

	predefObject["height"] = NewLong(5)
	return NewScope(proto.MainNetScheme, state.MockState{}, NewFuncScope(), predefObject)
}

func decode(s string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return decoded
}

var longScript = `match tx {
  case t : TransferTransaction | MassTransferTransaction | ExchangeTransaction => true
  case _ => false
}`

func TestEval(t *testing.T) {

	conds := []struct {
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
		{`tx.proofs[0] == base58'a'`, `AQkAAAAAAAACCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAABAAAAASEdRfFh`, true},
		{longScript, `AQQAAAAHJG1hdGNoMAUAAAACdHgDAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNFeGNoYW5nZVRyYW5zYWN0aW9uBgMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24GCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGB6Ilvok=`, true},
	}

	for _, c := range conds {

		reader, err := reader.NewReaderFromBase64(c.Base64)
		require.NoError(t, err)

		exprs, err := BuildAst(reader)
		require.NoError(t, err)

		rs, err := Eval(exprs, defaultScope())
		require.NoError(t, err)
		assert.Equal(t, c.Result, rs, fmt.Sprintf("script: %s", c.Name))
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
		e, _ := BuildAst(r)
		b.StartTimer()
		_, _ = Eval(e, s)
	}
}

func TestFunctions(t *testing.T) {
	conds := []struct {
		FuncCode int
		FuncName string
		Code     string
		Base64   string
		Result   bool
	}{
		{0, "EQ", `5 == 5`, `AQkAAAAAAAACAAAAAAAAAAAFAAAAAAAAAAAFqWG0Fw==`, true},
		{1,
			"ISINSTANCEOF",
			`match tx {case t : MassTransferTransaction => true case _  => false}`,
			`AQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAF01hc3NUcmFuc2ZlclRyYW5zYWN0aW9uBAAAAAF0BQAAAAckbWF0Y2gwBgcVsAaK`, true},
		{2, `THROW`, `true && throw("mess")`, `AQMGCQAAAgAAAAECAAAABG1lc3MH7PDwAQ==`, false},
		{100, `SUM_LONG`, `1 + 1 > 0`, `AQkAAGYAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAABiJjSk`, true},
		{101, `SUB_LONG`, `2 - 1 > 0`, `AQkAAGYAAAACCQAAZQAAAAIAAAAAAAAAAAIAAAAAAAAAAAEAAAAAAAAAAABqsps1`, true},
		{102, `GT_LONG`, `1 > 0`, `AQkAAGYAAAACAAAAAAAAAAABAAAAAAAAAAAAyAIM4w==`, true},
		{103, `GE_LONG`, `1 >= 0`, `AQkAAGcAAAACAAAAAAAAAAABAAAAAAAAAAAAm30DnQ==`, true},
		{104, `MUL_LONG`, `2 * 2>0`, `AQkAAGYAAAACCQAAaAAAAAIAAAAAAAAAAAIAAAAAAAAAAAIAAAAAAAAAAABCMM5o`, true},
		{105, `DIV_LONG`, `4 / 2>0`, `AQkAAGYAAAACCQAAaQAAAAIAAAAAAAAAAAQAAAAAAAAAAAIAAAAAAAAAAAAadVma`, true},
		{106, `MOD_LONG`, `-10 % 6>0`, `AQkAAGYAAAACCQAAagAAAAIA//////////YAAAAAAAAAAAYAAAAAAAAAAAB5rBSH`, true},
		{107, `FRACTION`, `fraction(10, 5, 2)>0`, `AQkAAGYAAAACCQAAawAAAAMAAAAAAAAAAAoAAAAAAAAAAAUAAAAAAAAAAAIAAAAAAAAAAACRyFu2`, true},

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

		{500, `SIGVERIFY`, `toString(true) == "true"`, `AQkAAAAAAAACCQABpQAAAAEGAgAAAAR0cnVlL6ZrWg==`, true},
	}

	for _, c := range conds {
		reader, err := reader.NewReaderFromBase64(c.Base64)
		require.NoError(t, err)

		exprs, err := BuildAst(reader)
		require.NoError(t, err)

		rs, err := Eval(exprs, defaultScope())
		assert.NoError(t, err)
		assert.Equal(t, c.Result, rs, fmt.Sprintf("func name: %s, code: %d, script: %s", c.FuncName, c.FuncCode, c.Code))
	}
}
