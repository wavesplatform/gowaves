package peer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

type mockID struct {
	id string
}

func (m *mockID) String() string {
	return m.id
}

func TestHandleStopContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(1 * time.Millisecond)
		cancel()
	}()
	parent := NewParent()
	remote := NewRemote()
	peer := &mockPeer{CloseFunc: func() error { return nil }}
	err := Handle(ctx, peer, parent, remote)
	assert.NoError(t, err)
	assert.Len(t, peer.CloseCalls(), 1)
	require.Len(t, parent.NotificationsCh, 1)
	connected, ok := (<-parent.NotificationsCh).(ConnectedNotification)
	require.True(t, ok)
	connectedPeer := connected.Peer.(*peerOnceCloser).Peer
	assert.Equal(t, peer, connectedPeer)
}

func TestHandleReceive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	remote := NewRemote()
	parent := NewParent()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		peer := &mockPeer{
			CloseFunc: func() error { return nil },
			IDFunc:    func() ID { return &mockID{id: "test-peer-id"} },
		}
		_ = Handle(ctx, peer, parent, remote)
		assert.Len(t, peer.CloseCalls(), 1)
		wg.Done()
	}()
	_ = (<-parent.NotificationsCh).(ConnectedNotification).Peer.(*peerOnceCloser).Peer // fist message should be notification about connection
	bb := bytebufferpool.Get()
	_, err := bb.Write(byte_helpers.TransferWithSig.MessageBytes)
	require.NoError(t, err)
	remote.FromCh <- bb
	assert.IsType(t, &proto.TransactionMessage{}, (<-parent.NodeMessagesCh).Message)
	cancel()
	wg.Wait()
}

func TestHandleError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	remote := NewRemote()
	parent := NewParent()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		peer := &mockPeer{CloseFunc: func() error { return nil }}
		_ = Handle(ctx, peer, parent, remote)
		assert.Len(t, peer.CloseCalls(), 1)
		wg.Done()
	}()
	_ = (<-parent.NotificationsCh).(ConnectedNotification).Peer.(*peerOnceCloser).Peer // fist message should be notification about connection
	err := errors.New("error")
	remote.ErrCh <- err
	actualErr := (<-parent.NotificationsCh).(DisconnectedNotification).Err
	assert.Equal(t, err, actualErr)
	cancel()
	wg.Wait()
}
