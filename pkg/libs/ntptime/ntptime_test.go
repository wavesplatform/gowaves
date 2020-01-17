package ntptime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNtpTimeImpl_Run(t *testing.T) {
	tm := New("0.ru.pool.ntp.org")
	rs, err := tm.Now()
	require.NoError(t, err)
	require.NotEmpty(t, rs)

	ctx, cancel := context.WithCancel(context.Background())
	go tm.Run(ctx, 0)
	<-time.After(10 * time.Millisecond)
	cancel()
}
