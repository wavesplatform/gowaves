package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

// referenceHistoryRecordEncoding encodes the history record in the straightforward way, it is used to check
// that the on-disk format of state version 28 is produced by the optimized encoder.
func referenceHistoryRecordEncoding(entity blockchainEntity, fixed bool, entries []historyEntry) []byte {
	res := []byte{byte(entity)}
	for _, e := range entries {
		if !fixed {
			res = binary.AppendUvarint(res, uint64(len(e.data)+4)) // #nosec G115 // test data.
		}
		res = append(res, e.data...)
		res = binary.BigEndian.AppendUint32(res, e.blockNum)
	}
	return res
}

func allBlockchainEntities() []blockchainEntity {
	var res []blockchainEntity
	for e := alias; e <= generators; e++ {
		res = append(res, e)
	}
	return res
}

func TestEntityPropertiesDefinedForAllEntities(t *testing.T) {
	for _, e := range allBlockchainEntities() {
		p, ok := entityProperties(e)
		require.True(t, ok, "entity %d", e)
		assert.True(t, p.needToFilter, "entity %d", e)
		if p.fixedSize {
			assert.Greater(t, p.recordSize, historyBlockNumSize, "entity %d", e)
		} else {
			assert.Zero(t, p.recordSize, "entity %d", e)
		}
	}
}

func TestEntityPropertiesRejectsUnknownEntities(t *testing.T) {
	for _, e := range []blockchainEntity{0, generators + 1, 100, 255} {
		_, ok := entityProperties(e)
		assert.False(t, ok, "entity %d", e)

		_, err := newHistoryRecordFromBytes([]byte{byte(e), 4, 0, 0, 0, 1})
		assert.ErrorContains(t, err, "bad entity type", "entity %d", e)

		hr := newHistoryRecord(e)
		hr.entries = []historyEntry{{data: []byte{1}, blockNum: 1}}
		_, err = hr.marshalBinary()
		assert.ErrorContains(t, err, "bad entity type", "entity %d", e)
		_, err = hr.countTotalSize()
		assert.Error(t, err, "entity %d", e)

		hf := newHistoryFormatter(nil) // DB is not reached for unknown entities.
		_, err = hf.filter(hr)
		assert.ErrorContains(t, err, "bad entity type", "entity %d", e)
		_, err = hf.cut(hr)
		assert.ErrorContains(t, err, "bad entity type", "entity %d", e)
	}
}

func testHistoryEntries(entity blockchainEntity, p blockchainEntityProperties, n int) []historyEntry {
	entries := make([]historyEntry, n)
	for i := range entries {
		size := p.recordSize - historyBlockNumSize
		if !p.fixedSize {
			size = i % 7 // includes empty data
		}
		data := make([]byte, size)
		for j := range data {
			data[j] = byte(int(entity) + i + j)
		}
		entries[i] = historyEntry{data: data, blockNum: uint32(1000 + i)} // #nosec G115 // test data.
	}
	return entries
}

func TestHistoryRecordRoundTripAllEntities(t *testing.T) {
	for _, e := range allBlockchainEntities() {
		p, ok := entityProperties(e)
		require.True(t, ok)
		for _, n := range []int{1, 2, 10, 100} {
			t.Run(fmt.Sprintf("entity_%d_entries_%d", e, n), func(t *testing.T) {
				entries := testHistoryEntries(e, p, n)
				hr := newHistoryRecord(e)
				hr.entries = entries
				data, err := hr.marshalBinary()
				require.NoError(t, err)
				assert.Equal(t, referenceHistoryRecordEncoding(e, p.fixedSize, entries), data)
				size, err := hr.countTotalSize()
				require.NoError(t, err)
				assert.Equal(t, len(data), size)

				decoded, err := newHistoryRecordFromBytes(data)
				require.NoError(t, err)
				assert.Equal(t, e, decoded.entityType)
				require.Len(t, decoded.entries, n)
				assert.Equal(t, n, cap(decoded.entries))
				for i := range entries {
					assert.Equal(t, entries[i].blockNum, decoded.entries[i].blockNum)
					assert.True(t, bytes.Equal(entries[i].data, decoded.entries[i].data))
					// Capacity is limited, so appending to entry data never overwrites the next entry.
					assert.Equal(t, len(decoded.entries[i].data), cap(decoded.entries[i].data))
				}
				again, err := decoded.marshalBinary()
				require.NoError(t, err)
				assert.Equal(t, data, again)
			})
		}
	}
}

func TestNewHistoryRecordFromBytesInvalidData(t *testing.T) {
	_, err := newHistoryRecordFromBytes(nil)
	assert.ErrorIs(t, err, errInvalidDataSize)

	// Fixed size record without a single complete entry.
	p, _ := entityProperties(alias)
	_, err = newHistoryRecordFromBytes(append([]byte{byte(alias)}, make([]byte, p.recordSize-1)...))
	assert.ErrorIs(t, err, errInvalidDataSize)

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"entry size exceeding the data", []byte{byte(dataEntry), 10, 1, 2, 3, 4}},
		{"entry too short to hold the block number", []byte{byte(dataEntry), 3, 1, 2, 3}},
		{"zero entry size", []byte{byte(dataEntry), 0, 0, 0, 0, 5, 1, 2, 3, 4}},
		{"truncated entry size", []byte{byte(dataEntry), 5, 1, 2, 3, 4, 5, 0x80}},
		{"non-canonical entry size", []byte{byte(dataEntry), 0x85, 0x00, 1, 2, 3, 4, 5}},
		{"trailing bytes", []byte{byte(dataEntry), 4, 0, 0, 0, 1, 0}},
		{"size of version 27", []byte{byte(dataEntry), 0, 0, 0, 5, 1, 0, 0, 0, 1}},
	} {
		_, err = newHistoryRecordFromBytes(tc.data)
		assert.ErrorIs(t, err, errInvalidDataSize, tc.name)
	}

	// Variable size record without entries.
	hr, err := newHistoryRecordFromBytes([]byte{byte(dataEntry)})
	require.NoError(t, err)
	assert.Empty(t, hr.entries)
}

func TestVariableSizeHistoryEntrySizes(t *testing.T) {
	// Entry sizes around the boundaries of the uvarint encoding.
	for _, dataSize := range []int{0, 123, 124, 125, 16379, 16380, 16381, 70000} {
		entries := []historyEntry{
			{data: bytes.Repeat([]byte{1}, dataSize), blockNum: 1},
			{data: []byte{2}, blockNum: 2},
		}
		hr := &historyRecord{entityType: dataEntry, entries: entries}
		data, err := hr.marshalBinary()
		require.NoError(t, err)
		assert.Equal(t, referenceHistoryRecordEncoding(dataEntry, false, entries), data)
		size, err := hr.countTotalSize()
		require.NoError(t, err)
		assert.Equal(t, len(data), size)
		decoded, err := newHistoryRecordFromBytes(data)
		require.NoError(t, err)
		require.Len(t, decoded.entries, 2)
		assert.Equal(t, entries[0].data, decoded.entries[0].data)
		assert.Equal(t, entries[1], decoded.entries[1])
	}
}

func TestVariableSizeHistoryLongEntrySizeInTheMiddle(t *testing.T) {
	// Short sizes followed by a long one: the decoding continues with the generic loop from the long size.
	for _, sizes := range [][]int{
		{1, 2, 16381, 3},
		{1, 70000},
		{200, 16380, 16381, 5, 70000, 7},
	} {
		entries := make([]historyEntry, len(sizes))
		for i, size := range sizes {
			entries[i] = historyEntry{data: bytes.Repeat([]byte{byte(i + 1)}, size), blockNum: uint32(i)} // #nosec G115
		}
		hr := &historyRecord{entityType: dataEntry, entries: entries}
		data, err := hr.marshalBinary()
		require.NoError(t, err)
		assert.Equal(t, referenceHistoryRecordEncoding(dataEntry, false, entries), data)
		decoded, err := newHistoryRecordFromBytes(data)
		require.NoError(t, err)
		assert.Equal(t, entries, decoded.entries, "sizes %v", sizes)
	}
}

func TestVariableSizeHistoryInvalidLongEntrySizeInTheMiddle(t *testing.T) {
	valid := []byte{byte(dataEntry), 5, 1, 0, 0, 0, 1} // One entry of one data byte and the block number.
	for name, tail := range map[string][]byte{
		"truncated":     {0x80, 0x80},
		"non-canonical": {0x85, 0x80, 0x00},
		"too long":      {0x80, 0x80, 0x01},
	} {
		_, err := newHistoryRecordFromBytes(append(bytes.Clone(valid), tail...))
		assert.ErrorIs(t, err, errInvalidDataSize, name)
	}
}

func TestUvarintSize(t *testing.T) {
	for _, v := range []uint64{0, 1, 127, 128, 16383, 16384, 1<<21 - 1, 1 << 21, 1<<63 - 1, 1<<64 - 1} {
		assert.Equal(t, len(binary.AppendUvarint(nil, v)), uvarintSize(v), "value %d", v)
	}
}

func TestHistoryRecordAppendEntry(t *testing.T) {
	hr := newHistoryRecord(dataEntry)
	require.NoError(t, hr.appendEntry(historyEntry{data: []byte{1}, blockNum: 1}))
	assert.Equal(t, []historyEntry{{data: []byte{1}, blockNum: 1}}, hr.entries)
	require.NoError(t, hr.appendEntry(historyEntry{data: []byte{2}, blockNum: 1}))
	assert.Equal(t, []historyEntry{{data: []byte{2}, blockNum: 1}}, hr.entries)
	require.NoError(t, hr.appendEntry(historyEntry{data: []byte{3}, blockNum: 2}))
	assert.Equal(t, []historyEntry{{data: []byte{2}, blockNum: 1}, {data: []byte{3}, blockNum: 2}}, hr.entries)
}

// reusingIterator mimics the LevelDB iterator, which reuses its value buffer: every move overwrites the
// previously returned value.
type reusingIterator struct {
	keyvalue.Iterator
	buf []byte
}

func (it *reusingIterator) scribble() {
	for i := range it.buf {
		it.buf[i] = 0xee
	}
}

func (it *reusingIterator) Next() bool {
	it.scribble()
	return it.Iterator.Next()
}

func (it *reusingIterator) Value() []byte {
	v := it.Iterator.Value()
	it.buf = append(it.buf[:0], v...)
	return it.buf[:len(v):len(v)]
}

func (it *reusingIterator) Release() {
	it.scribble()
	it.Iterator.Release()
}

func TestTopEntryIteratorCopiesReusedBuffer(t *testing.T) {
	to := createStorageObjects(t, true)
	to.addBlock(t, blockID0)
	const n = 10
	expected := make(map[string][]byte, n)
	for i := range n {
		key := []byte{accountsDataStorKeyPrefix, byte(i)}
		val := bytes.Repeat([]byte{byte(i + 1)}, 10+i)
		require.NoError(t, to.hs.addNewEntry(dataEntry, key, val, blockID0))
		expected[string(key)] = val
	}
	to.flush(t)

	dbIter, err := to.hs.db.NewKeyIterator([]byte{accountsDataStorKeyPrefix})
	require.NoError(t, err)
	iter := &topEntryIterator{dbIter: &reusingIterator{Iterator: dbIter}, fmt: to.hs.fmt, amend: to.hs.amend}
	got := make(map[string][]byte, n)
	for iter.Next() {
		got[string(iter.Key())] = iter.Value() // Values are kept without copying.
	}
	require.NoError(t, iter.Error())
	iter.Release()
	assert.Equal(t, expected, got)
}

func TestNewHistoryRecordWithLen(t *testing.T) {
	for _, n := range []int{0, 1, 2, 5} {
		hr := newHistoryRecordWithLen(dataEntry, n)
		assert.Equal(t, dataEntry, hr.entityType)
		assert.Len(t, hr.entries, n)
		assert.Equal(t, max(n, 1), cap(hr.entries), "n=%d", n)
	}

	// Appending beyond the inline entry reallocates and leaves the earlier view intact.
	hr := newHistoryRecord(dataEntry)
	require.NoError(t, hr.appendEntry(historyEntry{data: []byte{1}, blockNum: 1}))
	first := hr.entries
	require.NoError(t, hr.appendEntry(historyEntry{data: []byte{2}, blockNum: 2}))
	assert.Equal(t, []historyEntry{{data: []byte{1}, blockNum: 1}}, first)
	assert.Equal(t, []historyEntry{{data: []byte{1}, blockNum: 1}, {data: []byte{2}, blockNum: 2}}, hr.entries)

	// Combining a decoded single-entry history must not overwrite memory shared with other records.
	data, err := (&historyRecord{entityType: dataEntry, entries: []historyEntry{{data: []byte{7}, blockNum: 3}}}).
		marshalBinary()
	require.NoError(t, err)
	decoded, err := newHistoryRecordFromBytes(data)
	require.NoError(t, err)
	view := decoded.entries
	decoded.entries = append(decoded.entries, historyEntry{data: []byte{8}, blockNum: 4})
	assert.Equal(t, []historyEntry{{data: []byte{7}, blockNum: 3}}, view)
}

func TestNewHistoryRecordFromBytesSingleAllocation(t *testing.T) {
	for _, e := range []blockchainEntity{alias, wavesBalance, dataEntry} {
		p, ok := entityProperties(e)
		require.True(t, ok)
		hr := newHistoryRecord(e)
		hr.entries = testHistoryEntries(e, p, 1)
		data, err := hr.marshalBinary()
		require.NoError(t, err)
		allocs := testing.AllocsPerRun(100, func() {
			_, err = newHistoryRecordFromBytes(data)
		})
		require.NoError(t, err)
		assert.Equal(t, 1.0, allocs, "entity %d", e)
	}
}

func TestMarshalHistoryInto(t *testing.T) {
	for _, e := range []blockchainEntity{alias, ordersVolume, wavesBalance, dataEntry} {
		p, ok := entityProperties(e)
		require.True(t, ok)
		all := testHistoryEntries(e, p, 6)
		for split := 0; split <= len(all); split++ {
			head, tail := all[:split], all[split:]
			combined := &historyRecord{entityType: e, entries: all}
			want, err := combined.marshalBinary()
			require.NoError(t, err)
			got, err := marshalHistoryInto(nil, e, head, tail)
			require.NoError(t, err)
			assert.Equal(t, want, got, "entity %d, split %d", e, split)
			assert.Equal(t, referenceHistoryRecordEncoding(e, p.fixedSize, all), got)

			buf := make([]byte, 0, len(want)+10)
			reused, err := marshalHistoryInto(buf, e, head, tail)
			require.NoError(t, err)
			assert.Equal(t, want, reused)
			assert.Same(t, &buf[:1][0], &reused[0])
		}
	}
	_, err := marshalHistoryInto(nil, blockchainEntity(0xff), nil, nil)
	assert.ErrorContains(t, err, "bad entity type")
}

func TestReadEntrySize(t *testing.T) {
	for _, v := range []uint64{0, 1, 4, 127, 128, 132, 16383, 16384, 1<<21 - 1, 1 << 21, 1<<35 + 7, 1<<64 - 1} {
		enc := binary.AppendUvarint(nil, v)
		got, n := readEntrySize(append(enc, 0xff))
		assert.Equal(t, v, got, "value %d", v)
		assert.Equal(t, len(enc), n, "value %d", v)
	}
	for _, b := range [][]byte{
		nil, {0x80}, {0xff, 0xff}, // truncated
		{0x85, 0x00}, {0x80, 0x80, 0x00}, // non-canonical
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x02}, // overflow
	} {
		_, n := readEntrySize(b)
		assert.Zero(t, n, "bytes %x", b)
	}
}
