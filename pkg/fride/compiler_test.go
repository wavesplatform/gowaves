package fride

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func c(values ...interface{}) []interface{} {
	return values
}

func TestCompilation(t *testing.T) {
	for _, test := range []struct {
		comment   string
		source    string
		code      string
		constants []interface{}
	}{
		{`V1: true`, "AQa3b8tH", "02", nil},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn",
			"00000008000102", c(int64(1), "x")},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=",
			"00000008000102", c("abc", "x")},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=",
			"000000080001000002080003090004070000090005070001", c(int64(1), "i", "string", "s", "i", "s")},
		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E",
			"0205000b01020800000900010400080103080002090003", c("r", "r", "r", "r")},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==",
			"09000000000107000005000701090002040007010900030700010a000004070002000005070003", c(int64(0), int64(-10), int64(10), "i", "i", "i")},
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
		assert.ElementsMatch(t, test.constants, program.Constants)
	}
}
