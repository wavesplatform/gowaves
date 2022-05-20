package state_fsm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func mapAsync(a Async) []int {
	var out []int
	for _, t := range a {
		out = append(out, t.Type())
	}
	return out
}

type noopReschedule struct {
}

func (noopReschedule) Reschedule() {
}

func TestNewFsm(t *testing.T) {
	fakeCh := make(chan []uint8, 1)
	defer close(fakeCh)
	fsm, async, err := NewFsm(services.Services{Scheduler: noopReschedule{}, ListOfExcludedCh: fakeCh}, 1000)

	require.NoError(t, err)
	require.Equal(t, []int{tasks.AskPeers, tasks.Ping}, mapAsync(async))

	require.NotNil(t, fsm)
}
