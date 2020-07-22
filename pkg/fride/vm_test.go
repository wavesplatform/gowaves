package fride

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecution(t *testing.T) {
	for _, test := range []struct {
		comment string
		source  string
		res     bool
	}{
		{`V1: true`, "AQa3b8tH", true},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn", true},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", true},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=", false},
		{`V3: let i = 12345; let s = "12345"; toString(i) == s`, "AwQAAAABaQAAAAAAAAAwOQQAAAABcwIAAAAFMTIzNDUJAAAAAAAAAgkAAaQAAAABBQAAAAFpBQAAAAFz1B1iCw==", true},
		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E", true},
		{`V3: if (false) then {let r = true; r} else {let r = false; r}`, "AwMHBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXI+tfo1", false},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==", true},
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

		program, err := Compile(tree)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, program, test.comment)

		res, err := Run(program)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, res, test.comment)
		r, ok := res.(ScriptResult)
		assert.True(t, ok, test.comment)
		assert.Equal(t, test.res, bool(r), test.comment)
	}
}

func BenchmarkSimplestScript(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		src, err := base64.StdEncoding.DecodeString("AwZd0cYf") // V3: true
		require.NoError(b, err)
		tree, err := Parse(src)
		require.NoError(b, err)
		prg, err := Compile(tree)
		require.NoError(b, err)
		assert.NotNil(b, prg)
		res, err := Run(prg)
		require.NoError(b, err)
		r := res.(ScriptResult)
		assert.True(b, bool(r))
	}
}

func BenchmarkEval(b *testing.B) {
	//let x = addressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	//
	//let a = x
	//let b = a
	//let c = b
	//let d = c
	//let e = d
	//let f = e
	//
	//f == e

	code := "AQQAAAABeAkBAAAAEWFkZHJlc3NGcm9tU3RyaW5nAAAAAQIAAAAjM1BKYUR5cHJ2ZWt2UFhQdUF0eHJhcGFjdURKb3BnSlJhVTMEAAAAAWEFAAAAAXgEAAAAAWIFAAAAAWEEAAAAAWMFAAAAAWIEAAAAAWQFAAAAAWMEAAAAAWUFAAAAAWQEAAAAAWYFAAAAAWUJAAAAAAAAAgUAAAABZgUAAAABZS5FHzs="
	b.ReportAllocs()
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(b, err)
	tree, err := Parse(src)
	require.NoError(b, err)
	prg, err := Compile(tree)
	require.NoError(b, err)
	assert.NotNil(b, prg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := Run(prg)
		require.NoError(b, err)
		r := res.(ScriptResult)
		assert.True(b, bool(r))
	}
}
