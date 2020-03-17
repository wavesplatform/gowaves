package state_fsm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
)

func mapAsync(a Async) []int {
	var out []int
	for _, t := range a {
		out = append(out, t.Type())
	}
	return out
}

func TestNewFsm(t *testing.T) {
	fsm, async, err := NewFsm(
		nil, nil, nil, 0, 'W', nil, nil, nil)

	require.NoError(t, err)
	require.Equal(t, []int{tasks.ASK_PEERS, tasks.PING}, mapAsync(async))

	require.NotNil(t, fsm)
}
