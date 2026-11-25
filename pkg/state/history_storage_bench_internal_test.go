package state

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Benchmarks for the state storage hot paths. They also run as tests with `-benchtime=1x`.

const (
	benchWavesBalanceDataSize = 32  // Approximate size of an encoded waves balance record.
	benchDataEntryDataSize    = 128 // Typical size of a data entry value.
	benchFormatterBlocks      = 300
	benchFormatterMinHeight   = 200
	benchStorageKeys          = 1000
	benchRangeBlocks          = 10
)

type benchHistoryEntity struct {
	name     string
	entity   blockchainEntity
	dataSize int
}

func benchHistoryEntities() []benchHistoryEntity {
	return []benchHistoryEntity{
		{name: "alias", entity: alias, dataSize: aliasRecordSize},
		{name: "ordersVolume", entity: ordersVolume, dataSize: orderVolumeRecordSize},
		{name: "wavesBalance", entity: wavesBalance, dataSize: benchWavesBalanceDataSize},
		{name: "dataEntry", entity: dataEntry, dataSize: benchDataEntryDataSize},
	}
}

// benchEntriesCounts returns the numbers of entries in the benchmarked history records.
func benchEntriesCounts() []int {
	return []int{1, 10, 100}
}

// silenceBenchLogs discards log output until the benchmark ends.
// Log lines printed between a benchmark name and its result break the benchstat input format.
func silenceBenchLogs(b *testing.B) {
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.DiscardHandler))
	b.Cleanup(func() { slog.SetDefault(prev) })
}

// createBenchStorageObjects creates storage objects for a benchmark with logging silenced.
func createBenchStorageObjects(b *testing.B, amend bool) *testStorageObjects {
	silenceBenchLogs(b) // Must be registered before the storage cleanup, so it is restored after closing.
	return createStorageObjects(b, amend)
}

func makeBenchHistory(entity blockchainEntity, dataSize, entriesCount int) *historyRecord {
	h := newHistoryRecord(entity)
	h.entries = make([]historyEntry, entriesCount)
	for i := range entriesCount {
		data := make([]byte, dataSize)
		for j := range data {
			data[j] = byte(i + j)
		}
		h.entries[i] = historyEntry{data: data, blockNum: uint32(i + 1)}
	}
	return h
}

func benchKey(i int) []byte {
	var addr proto.AddressID
	binary.BigEndian.PutUint64(addr[:8], uint64(i)+1)
	k := wavesBalanceKey{address: addr}
	return k.bytes()
}

func benchKeys(n int) [][]byte {
	keys := make([][]byte, n)
	for i := range n {
		keys[i] = benchKey(i)
	}
	return keys
}

func benchValue(i int) []byte {
	v := make([]byte, benchWavesBalanceDataSize)
	binary.BigEndian.PutUint64(v, uint64(i))
	return v
}

func BenchmarkHistoryRecordMarshal(b *testing.B) {
	for _, e := range benchHistoryEntities() {
		for _, n := range benchEntriesCounts() {
			b.Run(fmt.Sprintf("%s/entries=%d", e.name, n), func(b *testing.B) {
				h := makeBenchHistory(e.entity, e.dataSize, n)
				data, err := h.marshalBinary()
				require.NoError(b, err)
				b.ReportAllocs()
				for b.Loop() {
					if _, mErr := h.marshalBinary(); mErr != nil {
						b.Fatal(mErr)
					}
				}
				// The size of the encoded record, it shows the effect of the history encoding on the DB size.
				b.ReportMetric(float64(len(data)), "encoded-B")
			})
		}
	}
}

func BenchmarkHistoryRecordUnmarshal(b *testing.B) {
	for _, e := range benchHistoryEntities() {
		for _, n := range benchEntriesCounts() {
			b.Run(fmt.Sprintf("%s/entries=%d", e.name, n), func(b *testing.B) {
				h := makeBenchHistory(e.entity, e.dataSize, n)
				data, err := h.marshalBinary()
				require.NoError(b, err)
				decoded, err := newHistoryRecordFromBytes(data)
				require.NoError(b, err)
				require.Equal(b, h, decoded)
				b.ReportAllocs()
				for b.Loop() {
					if _, uErr := newHistoryRecordFromBytes(data); uErr != nil {
						b.Fatal(uErr)
					}
				}
			})
		}
	}
}

// formatterBenchSetup creates a state with benchFormatterBlocks blocks and the rollback minimal height set to
// benchFormatterMinHeight. It returns the storage objects and the block numbers of all added blocks in order.
func formatterBenchSetup(b *testing.B, amend bool) (*testStorageObjects, []uint32) {
	to := createBenchStorageObjects(b, amend)
	ids := genRandBlockIDs(b, benchFormatterBlocks)
	for _, id := range ids {
		to.addBlock(b, id)
	}
	to.flush(b)
	require.NoError(b, to.stateDB.setRollbackMinHeight(benchFormatterMinHeight))
	require.NoError(b, to.stateDB.flushBatch())
	nums := make([]uint32, len(ids))
	for i, id := range ids {
		num, err := to.stateDB.blockIdToNum(id)
		require.NoError(b, err)
		nums[i] = num
	}
	return to, nums
}

func BenchmarkHistoryFormatterNormalize(b *testing.B) {
	for _, amend := range []bool{true, false} {
		to, nums := formatterBenchSetup(b, amend)
		hf := newHistoryFormatter(to.stateDB)
		minNum, err := hf.db.getMinAcceptableBlockNum()
		require.NoError(b, err)

		// History that spans the whole chain, so its oldest entries have to be cut.
		cutEntries := make([]historyEntry, 0, len(nums)/3+1)
		// History that has only entries above the minimal acceptable block, so nothing changes.
		keepEntries := make([]historyEntry, 0, len(nums))
		for i, num := range nums {
			e := historyEntry{data: benchValue(i), blockNum: num}
			if i%3 == 0 {
				cutEntries = append(cutEntries, e)
			}
			if num > minNum {
				keepEntries = append(keepEntries, e)
			}
		}
		cases := []struct {
			name    string
			entries []historyEntry
			changed bool
		}{
			{name: "cut", entries: cutEntries, changed: true},
			{name: "nochange", entries: keepEntries, changed: false},
		}
		for _, tc := range cases {
			b.Run(fmt.Sprintf("%s/amend=%t", tc.name, amend), func(b *testing.B) {
				h := newHistoryRecord(wavesBalance)
				h.entries = tc.entries
				changed, nErr := hf.normalize(h, amend)
				require.NoError(b, nErr)
				require.Equal(b, tc.changed, changed)
				b.ReportAllocs()
				for b.Loop() {
					h.entries = tc.entries
					if _, lErr := hf.normalize(h, amend); lErr != nil {
						b.Fatal(lErr)
					}
				}
			})
		}
	}
}

func historyStorageBenchSetup(b *testing.B) (*testStorageObjects, proto.BlockID) {
	to := createBenchStorageObjects(b, true)
	blockID := genRandBlockID(b)
	to.addBlock(b, blockID)
	to.flush(b)
	return to, blockID
}

func addBenchEntries(b *testing.B, to *testStorageObjects, blockID proto.BlockID, keys [][]byte) {
	for i, key := range keys {
		require.NoError(b, to.hs.addNewEntry(wavesBalance, key, benchValue(i), blockID))
	}
}

func BenchmarkHistoryStorageAddNewEntry(b *testing.B) {
	to, blockID := historyStorageBenchSetup(b)
	keys := benchKeys(benchStorageKeys)
	val := benchValue(1)
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		if err := to.hs.addNewEntry(wavesBalance, keys[i%len(keys)], val, blockID); err != nil {
			b.Fatal(err)
		}
		i++
	}
}

func BenchmarkHistoryStorageNewestTopEntryData(b *testing.B) {
	b.Run("local", func(b *testing.B) {
		to, blockID := historyStorageBenchSetup(b)
		keys := benchKeys(benchStorageKeys)
		addBenchEntries(b, to, blockID, keys)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			if _, err := to.hs.newestTopEntryData(keys[i%len(keys)]); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
	b.Run("db", func(b *testing.B) {
		to, blockID := historyStorageBenchSetup(b)
		keys := benchKeys(benchStorageKeys)
		addBenchEntries(b, to, blockID, keys)
		to.flush(b)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			if _, err := to.hs.newestTopEntryData(keys[i%len(keys)]); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkHistoryStorageTopEntryData(b *testing.B) {
	to, blockID := historyStorageBenchSetup(b)
	keys := benchKeys(benchStorageKeys)
	addBenchEntries(b, to, blockID, keys)
	to.flush(b)
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		if _, err := to.hs.topEntryData(keys[i%len(keys)]); err != nil {
			b.Fatal(err)
		}
		i++
	}
}

// BenchmarkHistoryStorageFlush measures flushing of the local history buffer into the batch.
// Every key already has a history in DB, so flush has to read, combine and re-encode it.
// The batch is reset after every iteration, so the DB content stays the same.
func BenchmarkHistoryStorageFlush(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("keys=%d", n), func(b *testing.B) {
			to, blockID := historyStorageBenchSetup(b)
			keys := benchKeys(n)
			addBenchEntries(b, to, blockID, keys)
			to.flush(b)
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				addBenchEntries(b, to, blockID, keys)
				b.StartTimer()
				if err := to.hs.flush(); err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				to.dbBatch.Reset()
				b.StartTimer()
			}
		})
	}
}

func BenchmarkHistoryStorageEntriesDataInHeightRange(b *testing.B) {
	to := createBenchStorageObjects(b, true)
	key := benchKey(0)
	ids := genRandBlockIDs(b, benchRangeBlocks)
	for i, id := range ids {
		to.addBlock(b, id)
		require.NoError(b, to.hs.addNewEntry(wavesBalance, key, benchValue(i), id))
		to.flush(b)
	}
	const start, end = 2, benchRangeBlocks - 2
	data, err := to.hs.entriesDataInHeightRange(key, start, end)
	require.NoError(b, err)
	require.NotEmpty(b, data)
	b.ReportAllocs()
	for b.Loop() {
		if _, rErr := to.hs.entriesDataInHeightRange(key, start, end); rErr != nil {
			b.Fatal(rErr)
		}
	}
}

func BenchmarkHistoryStorageTopEntryIterator(b *testing.B) {
	to, blockID := historyStorageBenchSetup(b)
	keys := benchKeys(benchStorageKeys)
	addBenchEntries(b, to, blockID, keys)
	to.flush(b)
	scan := func() int {
		iter, err := to.hs.newTopEntryIterator(wavesBalance)
		if err != nil {
			b.Fatal(err)
		}
		cnt := 0
		for iter.Next() {
			cnt++
		}
		iter.Release()
		if iErr := iter.Error(); iErr != nil {
			b.Fatal(iErr)
		}
		return cnt
	}
	require.Equal(b, len(keys), scan())
	b.ReportAllocs()
	for b.Loop() {
		scan()
	}
}

func BenchmarkLocalHistoryStorage(b *testing.B) {
	keys := benchKeys(benchStorageKeys)
	records := make([]*historyRecord, len(keys))
	for i := range keys {
		records[i] = makeBenchHistory(wavesBalance, benchWavesBalanceDataSize, 1)
	}
	fill := func(l *localHistoryStorage) {
		for i, k := range keys {
			if err := l.set(k, records[i]); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.Run("set", func(b *testing.B) {
		l := newLocalHistoryStorage()
		b.ReportAllocs()
		for b.Loop() {
			fill(l)
			l.reset()
		}
	})
	b.Run("get", func(b *testing.B) {
		l := newLocalHistoryStorage()
		fill(l)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			if _, err := l.get(keys[i%len(keys)]); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
	b.Run("getEntries", func(b *testing.B) {
		l := newLocalHistoryStorage()
		fill(l)
		b.ReportAllocs()
		for b.Loop() {
			if len(l.getEntries()) != len(keys) {
				b.Fatal("unexpected entries count")
			}
		}
	})
	b.Run("sort", func(b *testing.B) {
		l := newLocalHistoryStorage()
		fill(l)
		shuffled := make([]history, len(l.getEntries()))
		copy(shuffled, l.getEntries())
		r := rand.New(rand.NewPCG(1, 2)) // Deterministic shuffle for benchmark.
		r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		buf := make([]history, len(shuffled))
		b.ReportAllocs()
		for b.Loop() {
			copy(buf, shuffled)
			sortEntries(buf)
		}
	})
}
