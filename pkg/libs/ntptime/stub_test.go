package ntptime

import (
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/require"
)

func TestStub_Query(t *testing.T) {
	s := stub{
		resp: &ntp.Response{
			ClockOffset: 1 * time.Second,
		},
		err: nil,
	}

	rs, err := s.Query("")

	require.Equal(t, &ntp.Response{
		ClockOffset: 1 * time.Second,
	}, rs)
	require.NoError(t, err)
}

func TestStub_Now(t *testing.T) {
	require.NotEmpty(t, Stub{}.Now())
}
