package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func mkch() chan AsyncTask {
	return make(chan AsyncTask, 1)
}

func TestMineMicroTask_Run(t *testing.T) {
	task := MineMicroTask{}
	require.Equal(t, MINE_MICRO, task.Type())

	ch := mkch()
	_ = task.Run(context.Background(), ch)

	require.IsType(t, MineMicroTaskData{}, (<-ch).Data.(MineMicroTaskData))
}
