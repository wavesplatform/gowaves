package peer

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/p2p/common"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

type mockConnection struct {
	closeCalledTimes int
}

func (a *mockConnection) SendClosed() bool {
	panic("implement me")
}

func (a *mockConnection) ReceiveClosed() bool {
	panic("implement me")
}

func (a *mockConnection) Close() error {
	a.closeCalledTimes += 1
	return nil
}

func (a *mockConnection) Conn() net.Conn {
	return nil
}

func TestHHandleStopContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(1 * time.Millisecond)
		cancel()
	}()
	conn := &mockConnection{}
	err := Handle(HandlerParams{
		Ctx:        ctx,
		Connection: conn,
	})
	assert.Error(t, err)

	assert.Equal(t, 1, conn.closeCalledTimes)
}

func TestHandleReceive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &mockConnection{}
	remote := NewRemote()
	parent := NewParent()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = Handle(HandlerParams{
			Ctx:              ctx,
			Connection:       c,
			Parent:           parent,
			Remote:           remote,
			Pool:             bytespool.NewBytesPool(1, 15*1024),
			DuplicateChecker: common.NewDuplicateChecker(),
		})
		wg.Done()
	}()
	remote.FromCh <- byte_helpers.TransferWithSig.MessageBytes
	assert.IsType(t, &proto.TransactionMessage{}, (<-parent.MessageCh).Message)
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
		_ = Handle(HandlerParams{
			Ctx:        ctx,
			Connection: &mockConnection{},
			Remote:     remote,
			Parent:     parent,
		})
		wg.Done()
	}()
	err := errors.New("error")
	remote.ErrCh <- err
	assert.Equal(t, err, (<-parent.InfoCh).Value)
	cancel()
	wg.Wait()
}
