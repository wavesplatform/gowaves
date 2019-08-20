package reader

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

func decode(s string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return decoded
}

func TestBytesIterator_Next(t *testing.T) {

	v := decode("AQQAAAABeAAAAAAAAAAABQkAAGYAAAACAAAAAAAAAAAGAAAAAAAAAAAEYSW6XA==")
	iter := NewBytesReader(v)

	assert.Equal(t, E_BYTES, iter.Next())
	assert.Equal(t, E_BLOCK, iter.Next())
	assert.Equal(t, "x", iter.ReadString())
	assert.Equal(t, E_LONG, iter.Next())
	assert.EqualValues(t, 5, iter.ReadLong())
}

func TestBytesIterator_Eof(t *testing.T) {
	iter := NewBytesReader([]byte{})
	assert.True(t, iter.Eof())

	iter = NewBytesReader([]byte{2, 120})
	assert.False(t, iter.Eof())
	_, _ = iter.ReadByte()
	assert.False(t, iter.Eof())
	_, _ = iter.ReadByte()
	assert.True(t, iter.Eof())
}

func TestBytesReader_ReadInt(t *testing.T) {
	r := NewBytesReader([]byte{0, 0, 0, 5})
	assert.EqualValues(t, 5, r.ReadInt())
}
