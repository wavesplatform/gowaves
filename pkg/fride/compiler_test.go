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
		{`V1: true`, "AQa3b8tH", "020b", nil, 0},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn",
			"0c00000007020b0000010b", c(rideString("x"), rideInt(1)), 0},
		//{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=",
		//	"020c0000000c", c(rideString("abc")), nil},
		//{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=",
		//	"0000000c0000010c090002070003090004070005",
		//	c(rideInt(1), rideString("i"), rideString("string"), rideString("s")),
		//	[]externalCall{{"420", 1}, {"0", 2}}},
		//{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E",
		//	"0205ffff01020800000900000400080103080000090000",
		//	c(rideString("r")), nil},
		//{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==",
		//	"0a08000009000000000107000205000701090000040007010900000700030b000004070005000006070007",
		//	c(rideInt(0), rideInt(-10), rideInt(10), rideString("i")),
		//	[]externalCall{{"103", 2}, {"-", 1}, {"0", 2}}},
		//{`V3: if (true) then {if (false) then {func XX() = true; XX()} else {func XX() = false; XX()}} else {if (true) then {let x = false; x} else {let x = true; x}}`,
		//	"AwMGAwcKAQAAAAJYWAAAAAAGCQEAAAACWFgAAAAACgEAAAACWFgAAAAABwkBAAAAAlhYAAAAAAMGBAAAAAF4BwUAAAABeAQAAAABeAYFAAAAAXgYYeMi",
		//	"0a08000009000000000107000205000701090000040007010900000700030b000004070005000006070007",
		//	c(rideString("XX")), []externalCall{{"XX", 0}}},
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
