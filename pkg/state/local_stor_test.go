package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalStor(t *testing.T) {
	stor, err := newLocalStorage()
	assert.NoError(t, err, "newLocalStorage() failed")
	err = stor.set(alias, []byte("key0"), []byte("val0"))
	assert.NoError(t, err, "stor.set() failed")
	val, err := stor.get([]byte("key0"))
	assert.NoError(t, err, "stor.get() failed")
	assert.Equal(t, []byte("val0"), val)
	err = stor.set(lease, []byte("key0"), []byte("val1"))
	assert.NoError(t, err, "stor.set() failed")
	val, err = stor.get([]byte("key0"))
	assert.NoError(t, err, "stor.get() failed")
	assert.Equal(t, []byte("val1"), val)
	entries := stor.getEntries()
	assert.Equal(t, 1, len(entries))
	entry := keyValueEntry{[]byte("key0"), []byte("val1"), lease}
	assert.Equal(t, entry, entries[0])
	stor.reset()
	entries = stor.getEntries()
	assert.Equal(t, 0, len(entries))
}
