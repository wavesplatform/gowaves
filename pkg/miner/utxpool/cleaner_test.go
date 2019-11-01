package utxpool

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

type mockStateWrapper struct {
	index       int
	_lastHeader *proto.BlockHeader
	mu          *sync.RWMutex
}

func newMockStateWrapper(lastHeader *proto.BlockHeader) *mockStateWrapper {
	return &mockStateWrapper{
		index:       0,
		_lastHeader: lastHeader,
		mu:          &sync.RWMutex{},
	}
}

func (a mockStateWrapper) lastHeader() (*proto.BlockHeader, error) {
	return a._lastHeader, nil
}

func (a mockStateWrapper) Height() (uint64, error) {
	return 1, nil
}

func (a mockStateWrapper) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	return a._lastHeader, nil
}

func (a *mockStateWrapper) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, version proto.BlockVersion) error {
	a.index += 1
	if a.index%2 == 0 {
		return nil
	}
	return errors.New("invalid tx")
}

func (a mockStateWrapper) ResetValidationList() {
}

func (a *mockStateWrapper) Mutex() *lock.RwMutex {
	return lock.NewRwMutex(a.mu)
}

func TestInner_Handle(t *testing.T) {
	header := proto.BlockHeader{}

	utx := New(10000)

	require.True(t, utx.AddWithBytes(byte_helpers.TransferV1.Transaction, byte_helpers.TransferV1.TransactionBytes))
	require.True(t, utx.AddWithBytes(byte_helpers.IssueV1.Transaction, byte_helpers.IssueV1.TransactionBytes))

	inner := newInner(newMockStateWrapper(&header), utx)
	inner.Handle()

	require.Equal(t, 1, utx.Len())
}

func TestNewCleaner(t *testing.T) {
	require.NotNil(t, NewCleaner(services.Services{}))
}

func TestCleaner_work(t *testing.T) {
	block := &proto.Block{BlockHeader: proto.BlockHeader{
		BlockSignature: crypto.Signature{},
	}}
	m, err := node.NewMockStateManager(block)
	require.NoError(t, err)
	c := NewCleaner(services.Services{State: m, UtxPool: New(1)})
	c.work()
}

func TestCleaner_Handle(t *testing.T) {
	block := &proto.Block{BlockHeader: proto.BlockHeader{
		BlockSignature: crypto.Signature{},
	}}
	m, err := node.NewMockStateManager(block)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	c := NewCleaner(services.Services{State: m, UtxPool: New(1)})
	c.Run(ctx)
	c.Handle()

	cancel()
	c.Handle()
}

func TestStateWrapperImpl(t *testing.T) {
	block := &proto.Block{BlockHeader: proto.BlockHeader{
		BlockSignature: crypto.Signature{},
	}}
	m, err := node.NewMockStateManager(block)
	require.NoError(t, err)
	w := stateWrapperImpl{state: m}

	last, err := w.lastHeader()
	require.NoError(t, err)
	require.NotNil(t, last)

	require.NotNil(t, w.Mutex())
}
