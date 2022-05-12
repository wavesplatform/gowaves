package ntptime

import (
	"context"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNtpTimeImpl_Run(t *testing.T) {
	st := stub{
		resp: &ntp.Response{
			ClockOffset: 1 * time.Second,
		},
		err: nil,
	}
	tm, err := newNTPTime("pool.ntp.org", st)
	require.NoError(t, err)
	rs := tm.Now()
	require.NotEmpty(t, rs)

	ctx, cancel := context.WithCancel(context.Background())
	go tm.Run(ctx, 0)
	<-time.After(10 * time.Millisecond)
	cancel()
}

func TestTryNewSuccessful(t *testing.T) {
	st := stub{
		resp: &ntp.Response{
			ClockOffset: 1 * time.Second,
		},
		err: nil,
	}
	tm, err := tryNew("pool.ntp.org", 2, st)
	require.NoError(t, err)
	require.NotEmpty(t, tm)
}

func TestTryNewFailure(t *testing.T) {
	st := stub{
		err: errors.New("some error"),
	}
	tm, err := tryNew("pool.ntp.org", 1, st)
	require.Nil(t, tm)
	require.Error(t, err)
}
