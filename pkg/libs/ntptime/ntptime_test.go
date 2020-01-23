package ntptime

import (
	"context"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	require.NotEmpty(t, New("pool.ntp.org"))
}

func TestNtpTimeImpl_Run(t *testing.T) {
	st := stub{
		resp: &ntp.Response{
			ClockOffset: 1 * time.Second,
		},
		err: nil,
	}
	tm := new("pool.ntp.org", st)
	rs, err := tm.Now()
	require.NoError(t, err)
	require.NotEmpty(t, rs)

	ctx, cancel := context.WithCancel(context.Background())
	go tm.Run(ctx, 0)
	<-time.After(10 * time.Millisecond)
	cancel()
}
