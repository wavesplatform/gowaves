package evaluate

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/parser"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

func defaultScope() Scope {
	predefObject := make(map[string]Expr)
	predefObject["tx"] = NewObject(map[string]Expr{
		"id":              NewBytes([]byte{33}),
		"proofs":          Exprs([]Expr{NewBytes([]byte{33})}),
		InstanceFieldName: NewString("MassTransferTransaction"),
	})
	predefObject["height"] = NewLong(5)
	return NewScope(NewFuncScope(), predefObject)
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
		{`tx.id == base58'a'`, `AQkAAAAAAAACCAUAAAACdHgAAAACaWQBAAAAASHuubwr`, true},
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
		assert.NoError(t, err)
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
