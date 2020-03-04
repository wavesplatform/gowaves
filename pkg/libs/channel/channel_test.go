package channel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChannelMultipleClose(t *testing.T) {
	ch := NewChannel(10)
	ch.Close()
	ch.Close()
}

func TestChannelUnlockOnClose(t *testing.T) {
	ch := NewChannel(1)
	go func() {
		ch.Close()
	}()
	ch.Send(1)
	ch.Send(2)
}

func TestChannel_ReceiveDeadlock(t *testing.T) {
	ch := NewChannel(10)
	go func() {
		ch.Send(1)
	}()
	ch.Receive()
}

func TestChannel_Receive(t *testing.T) {
	ch := NewChannel(1)
	ch.Send(1)
	ch.Close()

	rs1, ok1 := ch.Receive()
	require.Equal(t, 1, rs1)
	require.True(t, ok1)

	rs2, ok2 := ch.Receive()
	require.Equal(t, nil, rs2)
	require.False(t, ok2)
}

func TestNewChannel(t *testing.T) {
	ch := NewChannel(1)
	ch.Send(1)
	go func() {
		<-time.After(1 * time.Second)
		ch.Close()
	}()
	require.False(t, ch.Send(2))
	require.False(t, ch.Send(3))
}
