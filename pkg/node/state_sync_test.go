package node

import (
	"context"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type mockHistoryBlockApplier struct {
	sync.Mutex
	bts [][]byte
	err error
}

func (a *mockHistoryBlockApplier) ApplyBlocksBytes(blocks [][]byte) error {
	a.Lock()
	a.bts = blocks
	a.Unlock()
	return a.err
}

func TestApplyHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan [][]byte, 1)
	mock := &mockHistoryBlockApplier{}
	require.Equal(t, 0, len(mock.bts))

	// sending no bytes, expect function succ exit
	ch <- nil
	require.NoError(t, applyWorker(ctx, 50, ch, mock))
	require.Equal(t, 0, len(mock.bts))

	// sending less than minimum value(50), expect exit and apply bytes
	ch <- make([][]byte, 49)
	require.NoError(t, applyWorker(ctx, 50, ch, mock))
	require.Equal(t, 49, len(mock.bts))

	// applier returns err, expect err
	ch <- make([][]byte, 50)
	mock.err = errors.New("some err")
	require.Equal(t, mock.err, applyWorker(ctx, 50, ch, mock))

	// check context exit
	mock.err = errors.New("some err")
	cancel()
	require.Equal(t, nil, applyWorker(ctx, 50, ch, mock))
}

func TestCreateBulkHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	receivedBlocksCh := make(chan blockBytes, 10)
	blocksBulk := make(chan []blockBytes, 10)

	receivedBlocksCh <- []byte{1}
	receivedBlocksCh <- []byte{1}
	receivedBlocksCh <- nil

	require.NoError(t, createBulkWorker(ctx, 2, receivedBlocksCh, blocksBulk))
	require.Equal(t, [][]byte{{1}, {1}}, <-blocksBulk)
	require.True(t, 0 == len(<-blocksBulk))

	cancel()
	require.Nil(t, createBulkWorker(ctx, 2, receivedBlocksCh, blocksBulk))
}
