package utxpool

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type transaction struct {
	fee uint64
	id  []byte
}

func (a transaction) GetID() ([]byte, error) {
	return a.id, nil
}

func (transaction) Valid() (bool, error) {
	panic("implement me")
}

func (transaction) MarshalBinary() ([]byte, error) {
	panic("implement me")
}

func (transaction) UnmarshalBinary([]byte) error {
	panic("implement me")
}

func (transaction) GetTimestamp() uint64 {
	return 0
}

func (transaction) GenerateID() {}
func (transaction) GetTypeVersion() proto.TransactionTypeVersion {
	panic("implement me")
}

func (a transaction) GetFee() uint64 {
	return a.fee
}

func tr(fee uint64) *transaction {
	return &transaction{fee: fee}
}

func id(b []byte, fee uint64) *transaction {
	return &transaction{fee: fee, id: b}
}

func TestTransactionPool(t *testing.T) {
	a := New(10000)

	// add unique by id transactions, then check them sorted
	a.AddWithBytes(id([]byte{}, 4), []byte{1})
	a.AddWithBytes(id([]byte{1}, 1), []byte{1})
	a.AddWithBytes(id([]byte{1, 2}, 10), []byte{1})
	a.AddWithBytes(id([]byte{1, 2, 3}, 8), []byte{1})

	require.EqualValues(t, 10, a.Pop().T.GetFee())
	require.EqualValues(t, 8, a.Pop().T.GetFee())
	require.EqualValues(t, 4, a.Pop().T.GetFee())
	require.EqualValues(t, 1, a.Pop().T.GetFee())
	require.Nil(t, a.Pop())
}

func BenchmarkTransactionPool(b *testing.B) {
	b.ReportAllocs()
	rand.Seed(time.Now().Unix())
	a := New(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		n := rand.Intn(1000000)
		b.StartTimer()
		a.AddWithBytes(tr(uint64(n)), []byte{1})
	}

	if a.Len() != b.N {
		b.Fatal("not all elements were pushed")
	}

	for i := 0; i < b.N; i++ {
		a.Pop()
	}

	if a.Len() != 0 {
		b.Fatal("size should be equal 0")
	}
}

func TestTransactionPool_Exists(t *testing.T) {
	a := New(10000)

	require.False(t, a.Exists(id([]byte{1, 2, 3}, 0)))

	a.AddWithBytes(id([]byte{1, 2, 3}, 10), []byte{1})
	require.True(t, a.Exists(id([]byte{1, 2, 3}, 0)))

	a.Pop()
	require.False(t, a.Exists(id([]byte{1, 2, 3}, 0)))
}

// check transaction not added when limit
func TestUtxPool_Limit(t *testing.T) {
	a := New(10)
	require.Equal(t, 0, a.Len())

	// added
	added := a.AddWithBytes(id([]byte{1, 2, 3}, 10), bytes.Repeat([]byte{1, 2}, 5))
	require.Equal(t, 1, a.Len())
	require.True(t, added)

	// not added
	added = a.AddWithBytes(id([]byte{1, 2, 3, 4}, 10), bytes.Repeat([]byte{1, 2}, 5))
	require.Equal(t, 1, a.Len())
	require.False(t, added)
}
