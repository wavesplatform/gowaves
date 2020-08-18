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
	}{
		{`V1: true`, "AQa3b8tH", "020c", nil},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn", "0e020c0000000b", c(rideInt(1))},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", "0e020c0000000b", c(rideString("abc"))},
		{`V3: func A() = 1; func B() = 2; true`, "AwoBAAAAAUEAAAAAAAAAAAAAAAABCgEAAAABQgAAAAAAAAAAAAAAAAIG+N0aQQ==",
			"0e0e020c0000010b0000000b", c(rideInt(1), rideInt(2))},
		{`V3: func A() = 1; func B() = 2; A() != B()`, "AwoBAAAAAUEAAAAAAAAAAAAAAAABCgEAAAABQgAAAAAAAAAAAAAAAAIJAQAAAAIhPQAAAAIJAQAAAAFBAAAAAAkBAAAAAUIAAAAAv/Pmkg==",
			"0e0e07001007000c0801020c0000010b0000000b", c(rideInt(1), rideInt(2))},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=",
			"0e0e09001308270109000f0803020c0000010b0000000b", c(rideInt(1), rideString("string"))},
		{`V3: if true then if true then true else false else false`, "AwMGAwYGBwdYjCji",
			"02050013010205000e0102040010010304001501030c", nil},
		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E",
			"0205000c010e090012040011010e0900140c020b030b", nil},
		{`V3: if (let a = 1; a == 0) then {let a = 2; a == 0} else {let a = 0; a == 0}`, "AwMEAAAAAWEAAAAAAAAAAAEJAAAAAAAAAgUAAAABYQAAAAAAAAAAAAQAAAABYQAAAAAAAAAAAgkAAAAAAAACBQAAAAFhAAAAAAAAAAAABAAAAAFhAAAAAAAAAAAACQAAAAAAAAIFAAAAAWEAAAAAAAAAAAB3u9Yb",
			"0e09002700000108030205001b010e09002b000003080302040026010e09002f0000050803020c0000000b0000020b0000040b", c(rideInt(1), rideInt(0), rideInt(2), rideInt(0), rideInt(0), rideInt(0))},
		{`let a = 1; let b = a; let c = b; a == c`,
			"AwQAAAABYQAAAAAAAAAAAQQAAAABYgUAAAABYQQAAAABYwUAAAABYgkAAAAAAAACBQAAAAFhBQAAAAFjUFI1Og==",
			"0e0e0e09001509000d0803020c0900110b0900150b0000000b", c(rideInt(1))},
		{`let x = addressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3"); let a = x; let b = a; let c = b; let d = c; let e = d; let f = e; f == e`,
			"AQQAAAABeAkBAAAAEWFkZHJlc3NGcm9tU3RyaW5nAAAAAQIAAAAjM1BKYUR5cHJ2ZWt2UFhQdUF0eHJhcGFjdURKb3BnSlJhVTMEAAAAAWEFAAAAAXgEAAAAAWIFAAAAAWEEAAAAAWMFAAAAAWIEAAAAAWQFAAAAAWMEAAAAAWUFAAAAAWQEAAAAAWYFAAAAAWUJAAAAAAAAAgUAAAABZgUAAAABZS5FHzs=",
			"0e0e0e0e0e0e0e0900110900150803020c0900150b0900190b09001d0b0900210b0900250b0900290b0000000837010b", c(rideString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3"))},
		{`V3: let x = { let y = 1; y == 0 }; let y = { let z = 2; z == 0 } x == y`,
			"AwQAAAABeAQAAAABeQAAAAAAAAAAAQkAAAAAAAACBQAAAAF5AAAAAAAAAAAABAAAAAF5BAAAAAF6AAAAAAAAAAACCQAAAAAAAAIFAAAAAXoAAAAAAAAAAAAJAAAAAAAAAgUAAAABeAUAAAABedn8HVg=",
			"0e0e09001f0900140803020c0000000b0000020b0e0900100000030803020b0e09000c0000010803020b", c(rideInt(1), rideInt(0), rideInt(2), rideInt(0))},
		{`V3: let z = 0; let a = {let b = 1; b == z}; let b = {let c = 2; c == z}; a == b`,
			"AwQAAAABegAAAAAAAAAAAAQAAAABYQQAAAABYgAAAAAAAAAAAQkAAAAAAAACBQAAAAFiBQAAAAF6BAAAAAFiBAAAAAFjAAAAAAAAAAACCQAAAAAAAAIFAAAAAWMFAAAAAXoJAAAAAAAAAgUAAAABYQUAAAABYnau3I8=",
			"0e0e0e0900200900150803020c0000010b0000020b0e09001109002b0803020b0e09000d09002b0803020b0000000b", c(rideInt(0), rideInt(1), rideInt(2))},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==",
			"0e00000107000e0000020803020c0a0000000000080d02050021010a0000040028010a00000802010b", c(rideInt(0), rideInt(-10), rideInt(10))},
		{`V3: if (true) then {if (false) then {func XX() = true; XX()} else {func XX() = false; XX()}} else {if (true) then {let x = false; x} else {let x = true; x}}`,
			"AwMGAwcKAQAAAAJYWAAAAAAGCQEAAAACWFgAAAAACgEAAAACWFgAAAAABwkBAAAAAlhYAAAAAAMGBAAAAAF4BwUAAAABeAQAAAABeAYFAAAAAXgYYeMi",
			"020500190103050011010e07002c040016010e07002e04002b0102050026010e09003004002b010e0900320c020b030b030b020b", nil},
		{`tx.sender == Address(base58'11111111111111111')`, "AwkAAAAAAAACCAUAAAACdHgAAAAGc2VuZGVyCQEAAAAHQWRkcmVzcwAAAAEBAAAAEQAAAAAAAAAAAAAAAAAAAAAAWc7d/w==",
			"0d180600000000010851010803020c", c(rideString("sender"), rideBytes{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})},
		{`func b(x: Int) = {func a(y: Int) = x + y; a(1) + a(2)}; b(2) + b(3) == 0`, "AwoBAAAAAWIAAAABAAAAAXgKAQAAAAFhAAAAAQAAAAF5CQAAZAAAAAIFAAAAAXgFAAAAAXkJAABkAAAAAgkBAAAAAWEAAAABAAAAAAAAAAABCQEAAAABYQAAAAEAAAAAAAAAAAIJAAAAAAAAAgkAAGQAAAACCQEAAAABYgAAAAEAAAAAAAAAAAIJAQAAAAFiAAAAAQAAAAAAAAAAAwAAAAAAAAAAAPsZlhQ=",
			"0e0000020700210000030700210805020000040803020c0a00000a00000805020b0e0000000700170000010700170805020b", c(rideInt(1), rideInt(2), rideInt(2), rideInt(3), rideInt(0))},
		{`func first(a: Int, b: Int) = {let x = a + b; x}; first(1, 2) == 0`, "AwoBAAAABWZpcnN0AAAAAgAAAAFhAAAAAWIEAAAAAXgJAABkAAAAAgUAAAABYQUAAAABYgUAAAABeAkAAAAAAAACCQEAAAAFZmlyc3QAAAACAAAAAAAAAAABAAAAAAAAAAACAAAAAAAAAAAAm+QHtw==",
			"", c(rideInt(1), rideInt(2), rideInt(0))},
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
