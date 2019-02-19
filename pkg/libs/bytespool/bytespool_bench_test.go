package bytespool

import "testing"

func BenchmarkBytesPool_Get_Put(b *testing.B) {
	b.ReportAllocs()

	pool := NewBytesPool(1, 1024)
	pool.Put(pool.Get())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b := pool.Get()
		pool.Put(b)
	}
}
