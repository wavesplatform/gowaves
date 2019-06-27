package utxpool

import (
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

func (a transaction) GetTypeVersion() proto.TransactionTypeVersion {
	panic("implement me")
}

func (a transaction) GetID() []byte {
	return a.id
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

func (a transaction) GetFee() uint64 {
	return a.fee
}

func (a transaction) GenerateID() {
	panic("not implemented")
}

func tr(fee uint64) *transaction {
	return &transaction{fee: fee}
}

func id(b []byte, fee uint64) *transaction {
	return &transaction{fee: fee, id: b}
}

func TestTransactionPool(t *testing.T) {
	a := New(10000)

	a.Add(tr(4))
	a.Add(tr(1))
	a.Add(tr(10))
	a.Add(tr(8))

	require.EqualValues(t, 10, a.Pop().GetFee())
	require.EqualValues(t, 8, a.Pop().GetFee())
	require.EqualValues(t, 4, a.Pop().GetFee())
	require.EqualValues(t, 1, a.Pop().GetFee())
	require.Equal(t, nil, a.Pop())
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
		a.Add(tr(uint64(n)))
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

	a.Add(id([]byte{1, 2, 3}, 10))
	require.True(t, a.Exists(id([]byte{1, 2, 3}, 0)))

	a.Pop()
	require.False(t, a.Exists(id([]byte{1, 2, 3}, 0)))
}
