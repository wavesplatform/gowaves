package state

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestLocalStor(t *testing.T) {
	stor := newLocalHistoryStorage()
	history0 := &historyRecord{entityType: wavesBalance}
	err := stor.set([]byte("key0"), history0)
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

func TestSortEntries(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 11)) // Deterministic data for test.
	entries := make([]history, 0, 300)
	for i := range 300 {
		key := make([]byte, 1+r.IntN(8))
		for j := range key {
			key[j] = byte(r.UintN(4)) // Small alphabet to get common prefixes.
		}
		entries = append(entries, history{key: key, value: &historyRecord{entityType: blockchainEntity(i % 3)}})
	}
	entries = append(entries, history{key: []byte{}}, history{key: []byte{0}}, history{key: []byte{0, 0}})
	r.Shuffle(len(entries), func(i, j int) { entries[i], entries[j] = entries[j], entries[i] })
	values := make(map[*historyRecord][]byte, len(entries))
	for _, e := range entries {
		if e.value != nil {
			values[e.value] = e.key
		}
	}

	sortEntries(entries)

	for i := 1; i < len(entries); i++ {
		assert.LessOrEqual(t, bytes.Compare(entries[i-1].key, entries[i].key), 0, "entries %d and %d out of order", i-1, i)
	}
	for _, e := range entries { // Values must move together with their keys.
		if e.value != nil {
			assert.Equal(t, values[e.value], e.key)
		}
	}
	assert.True(t, slices.IsSortedFunc(entries, compareHistoryKeys))
}

func TestLocalStorResetReuse(t *testing.T) {
	stor := newLocalHistoryStorage()
	const n = 100
	for i := range n {
		require.NoError(t, stor.set(fmt.Appendf(nil, "key%d", i), &historyRecord{entityType: wavesBalance}))
	}
	oldEntries := stor.getEntries()
	capBefore := cap(oldEntries)

	stor.reset()

	assert.Empty(t, stor.getEntries())
	assert.Empty(t, stor.index)
	assert.Equal(t, capBefore, cap(stor.getEntries()), "entries slice must be reused")
	for _, e := range oldEntries[:n] { // Old references are dropped to let GC collect them.
		assert.Nil(t, e.key)
		assert.Nil(t, e.value)
	}
	for i := range n {
		_, err := stor.get(fmt.Appendf(nil, "key%d", i))
		assert.ErrorIs(t, err, errNotFound, "stale entry %d after reset", i)
	}

	// Refill with fewer different keys: no stale entries must be visible.
	h := &historyRecord{entityType: dataEntry}
	require.NoError(t, stor.set([]byte("new0"), h))
	require.NoError(t, stor.set([]byte("new1"), h))
	require.NoError(t, stor.set([]byte("new0"), h))
	entries := stor.getEntries()
	require.Len(t, entries, 2)
	assert.Equal(t, []byte("new0"), entries[0].key)
	assert.Equal(t, []byte("new1"), entries[1].key)
	_, err := stor.get([]byte("key0"))
	assert.ErrorIs(t, err, errNotFound)
	got, err := stor.get([]byte("new1"))
	require.NoError(t, err)
	assert.Same(t, h, got)
}

func TestMarshalHistoryIntoReusesBuffer(t *testing.T) {
	small := makeBenchHistory(dataEntry, 10, 2)
	large := makeBenchHistory(dataEntry, 100, 5)
	expectedSmall, err := small.marshalBinary()
	require.NoError(t, err)
	expectedLarge, err := large.marshalBinary()
	require.NoError(t, err)

	buf, err := marshalHistoryInto(nil, small.entityType, small.entries, nil)
	require.NoError(t, err)
	assert.Equal(t, expectedSmall, buf)
	buf, err = marshalHistoryInto(buf, large.entityType, large.entries, nil) // Grows.
	require.NoError(t, err)
	assert.Equal(t, expectedLarge, buf)
	reused, err := marshalHistoryInto(buf, small.entityType, small.entries, nil) // Shrinks, reusing memory.
	require.NoError(t, err)
	assert.Equal(t, expectedSmall, reused)
	assert.Same(t, &buf[0], &reused[0])

	_, err = marshalHistoryInto(buf, blockchainEntity(0xff), nil, nil)
	assert.Error(t, err)
}

// TestHistoryStorageFlushOutput checks that flush with a reused encoding buffer writes
// exactly what the per-record encoding of the combined histories gives.
func TestHistoryStorageFlushOutput(t *testing.T) {
	to := createStorageObjects(t, true)
	ids := []proto.BlockID{genRandBlockID(t), genRandBlockID(t)}
	type item struct {
		entity blockchainEntity
		key    []byte
	}
	var items []item
	for i := range 40 {
		// Mixed entity types and value sizes, so the buffer both grows and shrinks between records.
		var it item
		switch i % 4 {
		case 0:
			it.entity = dataEntry
		case 1:
			it.entity = wavesBalance
		case 2:
			it.entity = alias
		default:
			it.entity = accountScript
		}
		it.key = fmt.Appendf(nil, "flush-key-%02d", 39-i)
		items = append(items, it)
	}
	value := func(it item, i, block int) []byte {
		size := 1 + (i*37+block*11)%200
		if p, ok := entityProperties(it.entity); ok && p.fixedSize {
			size = p.recordSize - historyBlockNumSize
		}
		return bytes.Repeat([]byte{byte(i + block)}, size)
	}
	for block, id := range ids {
		to.addBlock(t, id)
		for i, it := range items {
			if block == 1 && i%2 == 0 {
				continue // Some keys get only one entry, others are combined with the DB history.
			}
			require.NoError(t, to.hs.addNewEntry(it.entity, it.key, value(it, i, block), id))
		}
		// Compute the expected records with the allocating encoder before flushing.
		entries := slices.Clone(to.hs.stor.getEntries())
		sortEntries(entries)
		expected := make(map[string][]byte, len(entries))
		for _, e := range entries {
			combined, err := to.hs.combineHistories(e.key, e.value)
			require.NoError(t, err)
			data, err := combined.marshalBinary()
			require.NoError(t, err)
			expected[string(e.key)] = data
		}
		to.flush(t)
		assert.Empty(t, to.hs.stor.getEntries())
		for k, want := range expected {
			got, err := to.db.Get([]byte(k))
			require.NoError(t, err)
			assert.Equal(t, want, got, "key %q", k)
		}
	}
}
