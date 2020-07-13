package fride

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompilation(t *testing.T) {
	for _, test := range []struct {
		comment string
		source  string
		code    string
		longs   []int64
		bytes   [][]byte
		strings []string
	}{
		//{`V1: true`, "AQa3b8tH", "02", nil, nil, nil},
		//{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn",
		//	"00000008000002", []int64{1}, nil, []string{"x"}},
		//{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=",
		//	"00000008000102", nil, nil, []string{"abc", "x"}},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=",
			"000000080000000001080002090002070003090001070004 ", []int64{1}, nil, []string{"i", "string", "s", "", ""}},
		//{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E",
		//	"", nil, nil, nil},
		//{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==",
		//	"", nil, nil, nil},
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

		program, err := Compile(tree)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, program, test.comment)

		code := hex.EncodeToString(program.Code)
		assert.Equal(t, test.code, code, test.comment)

		assert.ElementsMatch(t, test.longs, program.LongConstants)
		assert.ElementsMatch(t, test.bytes, program.ByteConstants)
		assert.ElementsMatch(t, test.strings, program.StringConstants)
	}
}
