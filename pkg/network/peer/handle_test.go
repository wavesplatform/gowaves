package peer

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-errors/errors"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
)

type mockConnection struct {
	closeCalledTimes int
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
	handle(handlerParams{
		ctx:        ctx,
		connection: conn,
	})

	assert.Equal(t, 1, conn.closeCalledTimes)
}

func TestHandleReceive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := int32(0)
	c := &mockConnection{}
	remote := newRemote()
	go handle(handlerParams{
		ctx:        ctx,
		connection: c,
		receiveFromRemoteCallback: func(b []byte, address string, resendTo chan ProtoMessage, pool conn.Pool) {
			atomic.AddInt32(&called, 1)
		},
		remote: remote,
	})
	remote.fromCh <- []byte{}
	<-time.After(5 * time.Millisecond)
	assert.EqualValues(t, 1, atomic.LoadInt32(&called))
}

func TestHandleError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	remote := newRemote()
	parent := NewParent()
	go handle(handlerParams{
		ctx:        ctx,
		connection: &mockConnection{},
		remote:     remote,
		parent:     parent,
	})
	err := errors.New("error")
	remote.errCh <- err
	<-time.After(5 * time.Millisecond)
	assert.Equal(t, err, (<-parent.InfoCh).Value)
}
