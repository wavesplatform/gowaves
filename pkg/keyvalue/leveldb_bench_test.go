package keyvalue //nolint:testpackage // White-box benchmarks that reuse the unexported DB parameters.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	benchKeysCount  = 1000
	benchKeySize    = 27
	benchValueSize  = 64
	benchCacheSize  = 16 * 1024 * 1024
	benchBloomItems = 100000
)

func newBenchKeyVal(b *testing.B) *KeyVal {
	params := KeyValParams{
		CacheParams:         CacheParams{benchCacheSize},
		BloomFilterParams:   NewBloomFilterParams(benchBloomItems, 0.01, NoOpStore{}),
		WriteBuffer:         writeBuffer,
		CompactionTableSize: sstableSize,
		CompactionTotalSize: compactionTotalSize,
	}
	// Log lines printed between a benchmark name and its result break the benchstat input format.
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.DiscardHandler))
	b.Cleanup(func() { slog.SetDefault(prev) })
	kv, err := NewKeyVal(b.TempDir(), params)
	require.NoError(b, err)
	b.Cleanup(func() {
		require.NoError(b, kv.Close())
	})
	return kv
}

func benchKVKey(prefix byte, i int) []byte {
	k := make([]byte, benchKeySize)
	k[0] = prefix
	binary.BigEndian.PutUint64(k[1:], uint64(i))
	return k
}

func benchKVValue(i int) []byte {
	v := make([]byte, benchValueSize)
	binary.BigEndian.PutUint64(v, uint64(i))
	return v
}

func benchKVPairs(n int) ([][]byte, [][]byte) {
	keys := make([][]byte, n)
	values := make([][]byte, n)
	for i := range n {
		keys[i] = benchKVKey(1, i)
		values[i] = benchKVValue(i)
	}
	return keys, values
}

// fillBenchKeyVal writes the pairs into the DB through a batch.
func fillBenchKeyVal(b *testing.B, kv *KeyVal, keys, values [][]byte) {
	batch, err := kv.NewBatch()
	require.NoError(b, err)
	for i := range keys {
		batch.Put(keys[i], values[i])
	}
	require.NoError(b, kv.Flush(batch))
}

func BenchmarkBatchPutFlush(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("pairs=%d", n), func(b *testing.B) {
			kv := newBenchKeyVal(b)
			keys, values := benchKVPairs(n)
			batch, err := kv.NewBatch()
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				for i := range keys {
					batch.Put(keys[i], values[i])
				}
				if fErr := kv.Flush(batch); fErr != nil {
					b.Fatal(fErr)
				}
			}
		})
	}
}

func BenchmarkKeyValGet(b *testing.B) {
	b.Run("cache-hit", func(b *testing.B) {
		kv := newBenchKeyVal(b)
		keys, values := benchKVPairs(benchKeysCount)
		fillBenchKeyVal(b, kv, keys, values)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			if _, err := kv.Get(keys[i%len(keys)]); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
	b.Run("cache-miss-db-hit", func(b *testing.B) {
		kv := newBenchKeyVal(b)
		keys, values := benchKVPairs(benchKeysCount)
		fillBenchKeyVal(b, kv, keys, values)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			key := keys[i%len(keys)]
			b.StopTimer()
			kv.cache.Del(key) // Force the DB lookup.
			b.StartTimer()
			if _, err := kv.Get(key); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
	b.Run("bloom-negative", func(b *testing.B) {
		kv := newBenchKeyVal(b)
		keys, values := benchKVPairs(benchKeysCount)
		fillBenchKeyVal(b, kv, keys, values)
		missing := make([][]byte, benchKeysCount)
		for i := range missing {
			missing[i] = benchKVKey(2, i)
		}
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			// False positives of the bloom filter lead to a DB lookup, which also returns ErrNotFound.
			if _, err := kv.Get(missing[i%len(missing)]); !errors.Is(err, ErrNotFound) {
				b.Fatalf("unexpected error: %v", err)
			}
			i++
		}
	})
}

func BenchmarkKeyValHas(b *testing.B) {
	kv := newBenchKeyVal(b)
	keys, values := benchKVPairs(benchKeysCount)
	fillBenchKeyVal(b, kv, keys, values)
	b.Run("present", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			ok, err := kv.Has(keys[i%len(keys)])
			if err != nil || !ok {
				b.Fatalf("unexpected result: %t, %v", ok, err)
			}
			i++
		}
	})
	b.Run("absent", func(b *testing.B) {
		missing := benchKVKey(2, 0)
		b.ReportAllocs()
		for b.Loop() {
			ok, err := kv.Has(missing)
			if err != nil || ok {
				b.Fatalf("unexpected result: %t, %v", ok, err)
			}
		}
	})
}
