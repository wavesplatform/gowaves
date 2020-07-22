package fride

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func c(values ...rideType) []rideType {
	return values
}

func TestCompilation(t *testing.T) {
	for _, test := range []struct {
		comment   string
		source    string
		code      string
		constants []rideType
		entry     int
	}{
		{`V1: true`, "AQa3b8tH", "02", nil, 0},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn",
			"00000008000102", c(rideLong(1), rideString("x")), 0},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=",
			"00000008000102", c(rideString("abc"), rideString("x")), 0},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=",
			"000000080001000002080003090001070004090003070005",
			c(rideLong(1), rideString("i"), rideString("string"), rideString("s"), rideCall{"420", 1}, rideCall{"0", 2}), 0},
		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E",
			"0205000b01020800000900000400080103080000090000",
			c(rideString("r")), 0},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==",
			"0a08000009000000000107000205000701090000040007010900000700030b000004070005000006070007",
			c(rideLong(0), rideLong(-10), rideLong(10), rideString("i"), rideCall{"103", 2}, rideCall{"-", 1}, rideCall{"abs", 1}, rideCall{"0", 2}), 31},
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
		assert.Equal(t, test.entry, program.EntryPoint, test.comment)
	}
}
