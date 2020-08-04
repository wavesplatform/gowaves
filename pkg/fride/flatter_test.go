package fride

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlatter(t *testing.T) {
	for _, test := range []struct {
		comment string
		source  string
		code    string
	}{
		{`V1: true`, "AQa3b8tH", "TRUE RET"},
		{`V3: let x = 1; true`, "AwQAAAABeAAAAAAAAAAAAQbtAkXn", "TRUE RET [0] LONG(1) RET"},
		{`V3: let x = "abc"; true`, "AwQAAAABeAIAAAADYWJjBrpUkE4=", "TRUE RET [0] STRING(abc) RET"},
		{`V3: func A() = 1; func B() = 2; true`, "AwoBAAAAAUEAAAAAAAAAAAAAAAABCgEAAAABQgAAAAAAAAAAAAAAAAIG+N0aQQ==", "TRUE RET [0] LONG(1) RET [1] LONG(2) RET"},
		{`V3: func A() = 1; func B() = 2; A() != B()`, "AwoBAAAAAUEAAAAAAAAAAAAAAAABCgEAAAABQgAAAAAAAAAAAAAAAAIJAQAAAAIhPQAAAAIJAQAAAAFBAAAAAAkBAAAAAUIAAAAAv/Pmkg==",
			"LCALL(A, 0) LCALL(B, 1) CALL(!=) RET [0] LONG(1) RET [1] LONG(2) RET"},
		{`V1: let i = 1; let s = "string"; toString(i) == s`, "AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=",
			"REF(i, 0) CALL(420) REF(s, 1) CALL(0) RET [0] LONG(1) RET [1] STRING(string) RET"},

		{`V3: if (true) then {let r = true; r} else {let r = false; r}`, "AwMGBAAAAAFyBgUAAAABcgQAAAABcgcFAAAAAXJ/ok0E",
			"TRUE ? REF(r, 1) : REF(r, 0) RET [0] FALSE RET [1] TRUE RET"},
		{`V3: if (let a = 1; a == 0) then {let a = 2; a == 0} else {let a = 0; a == 0}`, "AwMEAAAAAWEAAAAAAAAAAAEJAAAAAAAAAgUAAAABYQAAAAAAAAAAAAQAAAABYQAAAAAAAAAAAgkAAAAAAAACBQAAAAFhAAAAAAAAAAAABAAAAAFhAAAAAAAAAAAACQAAAAAAAAIFAAAAAWEAAAAAAAAAAAB3u9Yb",
			"REF(a, 2) LONG(0) CALL(0) ? REF(a, 1) LONG(0) CALL(0) : REF(a, 0) LONG(0) CALL(0) RET [0] LONG(0) RET [1] LONG(2) RET [2] LONG(1) RET"},

		{`let x = addressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3"); let a = x; let b = a; let c = b; let d = c; let e = d; let f = e; f == e`,
			"AQQAAAABeAkBAAAAEWFkZHJlc3NGcm9tU3RyaW5nAAAAAQIAAAAjM1BKYUR5cHJ2ZWt2UFhQdUF0eHJhcGFjdURKb3BnSlJhVTMEAAAAAWEFAAAAAXgEAAAAAWIFAAAAAWEEAAAAAWMFAAAAAWIEAAAAAWQFAAAAAWMEAAAAAWUFAAAAAWQEAAAAAWYFAAAAAWUJAAAAAAAAAgUAAAABZgUAAAABZS5FHzs=",
			"REF(f, 6) REF(e, 5) CALL(0) RET [0] STRING(3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3) CALL(addressFromString) RET [1] REF(x, 0) RET [2] REF(a, 1) RET [3] REF(b, 2) RET [4] REF(c, 3) RET [5] REF(d, 4) RET [6] REF(e, 5) RET"},
		{`V3: let x = { let y = 1; y == 0 }; let y = { let z = 2; z == 0 } x == y`,
			"AwQAAAABeAQAAAABeQAAAAAAAAAAAQkAAAAAAAACBQAAAAF5AAAAAAAAAAAABAAAAAF5BAAAAAF6AAAAAAAAAAACCQAAAAAAAAIFAAAAAXoAAAAAAAAAAAAJAAAAAAAAAgUAAAABeAUAAAABedn8HVg=",
			"REF(x, 0) REF(y, 1) CALL(0) RET [0] REF(y, 3) LONG(0) CALL(0) RET [1] REF(z, 2) LONG(0) CALL(0) RET [2] LONG(2) RET [3] LONG(1) RET"},
		{`V3: let z = 0; let a = {let b = 1; b == z}; let b = {let c = 2; c == z}; a == b`,
			"AwQAAAABegAAAAAAAAAAAAQAAAABYQQAAAABYgAAAAAAAAAAAQkAAAAAAAACBQAAAAFiBQAAAAF6BAAAAAFiBAAAAAFjAAAAAAAAAAACCQAAAAAAAAIFAAAAAWMFAAAAAXoJAAAAAAAAAgUAAAABYQUAAAABYnau3I8=",
			"REF(a, 1) REF(b, 2) CALL(0) RET [0] LONG(0) RET [1] REF(b, 4) REF(z, 0) CALL(0) RET [2] REF(c, 3) REF(z, 0) CALL(0) RET [3] LONG(2) RET [4] LONG(1) RET"},
		{`V3: func abs(i:Int) = if (i >= 0) then i else -i; abs(-10) == 10`, "AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJAQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==",
			"LONG(-10) LCALL(abs, 0) LONG(10) CALL(0) RET [0] LSTORE(i) LREF(i) LONG(0) CALL(103) ? LREF(i) : LREF(i) CALL(-) RET"},
		{`V3: if (true) then {if (false) then {func XX() = true; XX()} else {func XX() = false; XX()}} else {if (true) then {let x = false; x} else {let x = true; x}}`,
			"AwMGAwcKAQAAAAJYWAAAAAAGCQEAAAACWFgAAAAACgEAAAACWFgAAAAABwkBAAAAAlhYAAAAAAMGBAAAAAF4BwUAAAABeAQAAAABeAYFAAAAAXgYYeMi",
			"TRUE ? FALSE ? LCALL(XX, 3) : LCALL(XX, 2) : TRUE ? REF(x, 1) : REF(x, 0) RET [0] TRUE RET [1] FALSE RET [2] FALSE RET [3] TRUE RET"},
		{`V3: if (true) then {if (false) then {func A() = 1; A() == 0} else {func A() = false; A()}} else {if (true) then {let x = false; x} else {let x = true; x}}`,
			"AwMGAwcKAQAAAAFBAAAAAAAAAAAAAAAAAQkAAAAAAAACCQEAAAABQQAAAAAAAAAAAAAAAAAKAQAAAAFBAAAAAAcJAQAAAAFBAAAAAAMGBAAAAAF4BwUAAAABeAQAAAABeAYFAAAAAXiOtKj8",
			"TRUE ? FALSE ? LCALL(A, 3) LONG(0) CALL(0) : LCALL(A, 2) : TRUE ? REF(x, 1) : REF(x, 0) RET [0] TRUE RET [1] FALSE RET [2] FALSE RET [3] LONG(1) RET"},
		{`V3: func A() = true; let a = if A() then {func B() = !A(); if B() then {let a = A(); if a then 1 else 2} else 3} else 4; a == 3`,
			"AwoBAAAAAUEAAAAABgQAAAABYQMJAQAAAAFBAAAAAAoBAAAAAUIAAAAACQEAAAABIQAAAAEJAQAAAAFBAAAAAAMJAQAAAAFCAAAAAAQAAAABYQkBAAAAAUEAAAAAAwUAAAABYQAAAAAAAAAAAQAAAAAAAAAAAgAAAAAAAAAAAwAAAAAAAAAABAkAAAAAAAACBQAAAAFhAAAAAAAAAAAD3Gkqxg==",
			"REF(a, 1) LONG(3) CALL(0) RET [0] TRUE RET [1] LCALL(A, 0) ? LCALL(B, 2) ? REF(a, 3) ? LONG(1) : LONG(2) : LONG(3) : LONG(4) RET [2] LCALL(A, 0) CALL(!) RET [3] LCALL(A, 0) RET"},
	} {
		src, err := base64.StdEncoding.DecodeString(test.source)
		require.NoError(t, err, test.comment)

		tree, err := Parse(src)
		require.NoError(t, err, test.comment)
		assert.NotNil(t, tree, test.comment)

		res, err := flatten(tree)
		require.NoError(t, err, test.comment)
		assert.Equal(t, test.code, res, test.comment)
	}
}
