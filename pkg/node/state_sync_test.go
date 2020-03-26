package node

import (
	"context"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type mockHistoryBlockApplier struct {
	sync.Mutex
	bts []*proto.Block
	err error
}

func (a *mockHistoryBlockApplier) Apply(blocks []*proto.Block) error {
	a.Lock()
	a.bts = blocks
	a.Unlock()
	return a.err
}

func TestApplyHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan []*proto.Block, 1)
	mock := &mockHistoryBlockApplier{}
	require.Equal(t, 0, len(mock.bts))

	// sending no bytes, expect function succ exit
	ch <- nil
	require.NoError(t, applyWorker(ctx, 50, ch, mock))
	require.Equal(t, 0, len(mock.bts))

	// sending less than minimum value(50), expect exit and apply bytes
	ch <- make([]*proto.Block, 49)
	require.NoError(t, applyWorker(ctx, 50, ch, mock))
	require.Equal(t, 49, len(mock.bts))

	// applier returns err, expect err
	ch <- make([]*proto.Block, 50)
	mock.err = errors.New("some err")
	require.Equal(t, mock.err, applyWorker(ctx, 50, ch, mock))

	// check context exit
	mock.err = errors.New("some err")
	cancel()
	require.Equal(t, nil, applyWorker(ctx, 50, ch, mock))
}

func TestCreateBulkHandler(t *testing.T) {
	t.Skip()
	ctx, cancel := context.WithCancel(context.Background())
	receivedBlocksCh := make(chan blockBytes, 10)
	blocksBulk := make(chan []*proto.Block, 10)

	block, err := proto.CreateBlock(
		proto.Transactions(nil),
		100,
		proto.BlockID{},
		crypto.PublicKey{},
		proto.NxtConsensus{},
		1,
		nil,
		100500,
		proto.TestNetScheme)
	require.NoError(t, err)
	bts, err := block.MarshalBinary()
	require.NoError(t, err)

	receivedBlocksCh <- blockBytes{bts, false}
	receivedBlocksCh <- blockBytes{bts, false}
	//receivedBlocksCh <- nil

	require.NoError(t, createBulkWorker(ctx, 2, receivedBlocksCh, blocksBulk, proto.MainNetScheme))
	require.Equal(t, [][]byte{{1}, {1}}, <-blocksBulk)
	require.True(t, 0 == len(<-blocksBulk))

	cancel()
	require.Nil(t, createBulkWorker(ctx, 2, receivedBlocksCh, blocksBulk, proto.MainNetScheme))
}
