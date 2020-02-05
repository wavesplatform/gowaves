package utxpool

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

type mockStateWrapper struct {
	index      int
	_lastBlock *proto.Block
	mu         *sync.RWMutex
}

func newMockStateWrapper(lastHeader *proto.Block) *mockStateWrapper {
	return &mockStateWrapper{
		index:      0,
		_lastBlock: lastHeader,
		mu:         &sync.RWMutex{},
	}
}

func (a mockStateWrapper) TopBlock() (*proto.Block, error) {
	return a._lastBlock, nil
}

func (a mockStateWrapper) Height() (uint64, error) {
	return 1, nil
}

func (a mockStateWrapper) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	return &a._lastBlock.BlockHeader, nil
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
	header := proto.Block{}

	utx := New(10000, NoOpValidator{})

	require.NoError(t, utx.AddWithBytes(byte_helpers.TransferV1.Transaction, byte_helpers.TransferV1.TransactionBytes))
	require.NoError(t, utx.AddWithBytes(byte_helpers.IssueV1.Transaction, byte_helpers.IssueV1.TransactionBytes))

	inner := newInner(newMockStateWrapper(&header), utx, ntptime.Stub{})
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
	c := NewCleaner(services.Services{State: m, UtxPool: New(1000, NoOpValidator{}), Time: ntptime.Stub{}})
	c.work()
}

func TestCleaner_Handle(t *testing.T) {
	block := &proto.Block{BlockHeader: proto.BlockHeader{
		BlockSignature: crypto.Signature{},
	}}
	m, err := node.NewMockStateManager(block)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	c := NewCleaner(services.Services{State: m, UtxPool: nil})
	go c.Run(ctx)
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

	last, err := w.TopBlock()
	require.NoError(t, err)
	require.NotNil(t, last)

	require.NotNil(t, w.Mutex())
}
