package api

const base58BTCAlphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var base58Alphabet map[rune]struct{}

func init() {
	base58Alphabet = make(map[rune]struct{})
	for _, r := range base58BTCAlphabet {
		base58Alphabet[r] = struct{}{}
	}
}
