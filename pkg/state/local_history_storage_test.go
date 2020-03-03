package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalStor(t *testing.T) {
	stor, err := newLocalHistoryStorage()
	assert.NoError(t, err, "newLocalHistoryStorage() failed")
	history0 := &historyRecord{entityType: wavesBalance}
	err = stor.set([]byte("key0"), history0)
	assert.NoError(t, err, "stor.set() failed")
	val, err := stor.get([]byte("key0"))
	assert.NoError(t, err, "stor.get() failed")
	assert.Equal(t, history0, val)
	history1 := &historyRecord{entityType: dataEntry}
	err = stor.set([]byte("key0"), history1)
	assert.NoError(t, err, "stor.set() failed")
	val, err = stor.get([]byte("key0"))
	assert.NoError(t, err, "stor.get() failed")
	assert.Equal(t, history1, val)
	entries := stor.getEntries()
	assert.Equal(t, 1, len(entries))
	entry := history{[]byte("key0"), history1}
	assert.Equal(t, entry, entries[0])
	stor.reset()
	entries = stor.getEntries()
	assert.Equal(t, 0, len(entries))
}
