package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUTF16Size(t *testing.T) {
	for _, test := range []struct {
		s    string
		size int
	}{
		{"Hello", 5},
		{"Привет", 6},
		{"世界", 2},
		{"x冬x", 4},
		{"", 0},
	} {
		r := UTF16Size(test.s)
		assert.Equal(t, test.size, r)
	}
}
