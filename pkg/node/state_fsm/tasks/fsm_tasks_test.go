package tasks

import (
	"testing"
)

func TestAskPeersTask_Run(t *testing.T) {

}

//func TestGetSignaturesTimoutTask_Run(t *testing.T) {
//	task := NewGetSignaturesTimoutTask(1 * time.Microsecond)
//	output := make(chan AsyncTask, 1)
//	task.Run(context.Background(), output)
//	require.Equal(t, SYNC_WAIT_SIGNATURES_TIMEOUT, (<-output).TaskType)
//}
