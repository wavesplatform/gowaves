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
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn", "020c0000000b", c(rideInt(1))},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", "020c0000000b", c(rideString("abc"))},
		{`V3: func A() = 1; func B() = 2; true`, "AwoBAAAAAUEAAAAAAAAAAAAAAAABCgEAAAABQgAAAAAAAAAAAAAAAAIG+N0aQQ==",
			"020c0000010b0000000b", c(rideInt(1), rideInt(2))},
		{`V3: func A() = 1; func B() = 2; A() != B()`, "AwoBAAAAAUEAAAAAAAAAAAAAAAABCgEAAAABQgAAAAAAAAAAAAAAAAIJAQAAAAIhPQAAAAIJAQAAAAFBAAAAAAkBAAAAAUIAAAAAv/Pmkg==",
			"07000e07000a0801020c0000010b0000000b", c(rideInt(1), rideInt(2))},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=",
			"09001108270109000d0803020c0000010b0000000b", c(rideInt(1), rideString("string"))},
		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E",
			"0205000b0109001004000f010900120c020b030b", nil},
		{`V3: if (let a = 1; a == 0) then {let a = 2; a == 0} else {let a = 0; a == 0}`, "AwMEAAAAAWEAAAAAAAAAAAEJAAAAAAAAAgUAAAABYQAAAAAAAAAAAAQAAAABYQAAAAAAAAAAAgkAAAAAAAACBQAAAAFhAAAAAAAAAAAABAAAAAFhAAAAAAAAAAAACQAAAAAAAAIFAAAAAWEAAAAAAAAAAAB3u9Yb",
			"090024000001080302050019010900280000030803020400230109002c0000050803020c0000000b0000020b0000040b", c(rideInt(1), rideInt(0), rideInt(2), rideInt(0), rideInt(0), rideInt(0))},
		{`let a = 1; let b = a; let c = b; a == c`,
			"AwQAAAABYQAAAAAAAAAAAQQAAAABYgUAAAABYQQAAAABYwUAAAABYgkAAAAAAAACBQAAAAFhBQAAAAFjUFI1Og==",
			"09001209000a0803020c09000e0b0900120b0000000b", c(rideInt(1))},
		{`let x = addressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3"); let a = x; let b = a; let c = b; let d = c; let e = d; let f = e; f == e`,
			"AQQAAAABeAkBAAAAEWFkZHJlc3NGcm9tU3RyaW5nAAAAAQIAAAAjM1BKYUR5cHJ2ZWt2UFhQdUF0eHJhcGFjdURKb3BnSlJhVTMEAAAAAWEFAAAAAXgEAAAAAWIFAAAAAWEEAAAAAWMFAAAAAWIEAAAAAWQFAAAAAWMEAAAAAWUFAAAAAWQEAAAAAWYFAAAAAWUJAAAAAAAAAgUAAAABZgUAAAABZS5FHzs=",
			"09000a09000e0803020c09000e0b0900120b0900160b09001a0b09001e0b0900220b0000000837010b", c(rideString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3"))},
		{`V3: let x = { let y = 1; y == 0 }; let y = { let z = 2; z == 0 } x == y`,
			"AwQAAAABeAQAAAABeQAAAAAAAAAAAQkAAAAAAAACBQAAAAF5AAAAAAAAAAAABAAAAAF5BAAAAAF6AAAAAAAAAAACCQAAAAAAAAIFAAAAAXoAAAAAAAAAAAAJAAAAAAAAAgUAAAABeAUAAAABedn8HVg=",
			"09001c0900120803020c0000000b0000020b09000e0000030803020b09000a0000010803020b", c(rideInt(1), rideInt(0), rideInt(2), rideInt(0))},
		{`V3: let z = 0; let a = {let b = 1; b == z}; let b = {let c = 2; c == z}; a == b`,
			"AwQAAAABegAAAAAAAAAAAAQAAAABYQQAAAABYgAAAAAAAAAAAQkAAAAAAAACBQAAAAFiBQAAAAF6BAAAAAFiBAAAAAFjAAAAAAAAAAACCQAAAAAAAAIFAAAAAWMFAAAAAXoJAAAAAAAAAgUAAAABYQUAAAABYnau3I8=",
			"09001c0900120803020c0000010b0000020b09000e0900260803020b09000a0900260803020b0000000b", c(rideInt(0), rideInt(1), rideInt(2))},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==",
			"00000107000d0000020803020c0a0000000000080d02050020010a0000040027010a00000802010b", c(rideInt(0), rideInt(-10), rideInt(10))},
		{`V3: if (true) then {if (false) then {func XX() = true; XX()} else {func XX() = false; XX()}} else {if (true) then {let x = false; x} else {let x = true; x}}`,
			"AwMGAwcKAQAAAAJYWAAAAAAGCQEAAAACWFgAAAAACgEAAAACWFgAAAAABwkBAAAAAlhYAAAAAAMGBAAAAAF4BwUAAAABeAQAAAABeAYFAAAAAXgYYeMi",
			"020500170103050010010700280400140107002a04002701020500230109002c0400270109002e0c020b030b030b020b", nil},
		{`tx.sender == Address(base58'11111111111111111')`, "AwkAAAAAAAACCAUAAAACdHgAAAAGc2VuZGVyCQEAAAAHQWRkcmVzcwAAAAEBAAAAEQAAAAAAAAAAAAAAAAAAAAAAWc7d/w==",
			"0d170600000000010851010803020c", c(rideString("sender"), rideBytes{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})},
		{`func b(x: Int) = {func a(y: Int) = x + y; a(1) + a(2)}; b(2) + b(3) == 0`, "AwoBAAAAAWIAAAABAAAAAXgKAQAAAAFhAAAAAQAAAAF5CQAAZAAAAAIFAAAAAXgFAAAAAXkJAABkAAAAAgkBAAAAAWEAAAABAAAAAAAAAAABCQEAAAABYQAAAAEAAAAAAAAAAAIJAAAAAAAAAgkAAGQAAAACCQEAAAABYgAAAAEAAAAAAAAAAAIJAQAAAAFiAAAAAQAAAAAAAAAAAwAAAAAAAAAAAPsZlhQ=",
			"000002 070020 000003 070020 080502 000004 080302 0c 0a0000 0a0000 080502 0b 000000 070016 000001 070016 080502 0b", c(rideInt(1), rideInt(2), rideInt(2), rideInt(3), rideInt(0))},
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
