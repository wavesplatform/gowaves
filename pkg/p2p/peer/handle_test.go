package peer

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
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
	Handle(HandlerParams{
		Ctx:        ctx,
		Connection: conn,
	})

	assert.Equal(t, 1, conn.closeCalledTimes)
}

func TestHandleReceive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &mockConnection{}
	remote := NewRemote()
	parent := NewParent()
	go Handle(HandlerParams{
		Ctx:        ctx,
		Connection: c,
		Parent:     parent,
		Remote:     remote,
		Pool:       bytespool.NewBytesPool(1, 15*1024),
	})
	remote.FromCh <- byte_helpers.TransferV1.MessageBytes
	assert.IsType(t, &proto.TransactionMessage{}, (<-parent.MessageCh).Message)
}

func TestHandleError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	remote := NewRemote()
	parent := NewParent()
	go Handle(HandlerParams{
		Ctx:        ctx,
		Connection: &mockConnection{},
		Remote:     remote,
		Parent:     parent,
	})
	err := errors.New("error")
	remote.ErrCh <- err
	<-time.After(5 * time.Millisecond)
	assert.Equal(t, err, (<-parent.InfoCh).Value)
}
