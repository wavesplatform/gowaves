package wallet

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCryptKey(t *testing.T) {

	word := "bla bla bla"

	a := NewCrypt([]byte("1"))
	b, err := a.Encrypt([]byte(word))
	require.NoError(t, err)

	word2, err := a.Decrypt(b)
	require.NoError(t, err)

	assert.Equal(t, word, string(word2))
}
