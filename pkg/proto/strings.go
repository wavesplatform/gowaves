package proto

import "unicode/utf16"

func UTF16Size(s string) int {
	return len(utf16.Encode([]rune(s)))
}
