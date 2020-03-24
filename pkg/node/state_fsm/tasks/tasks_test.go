package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func mkch() chan AsyncTask {
	return make(chan AsyncTask, 1)
}

func TestAskPeersTask_Run(t *testing.T) {

}

func TestMineMicroTask_Run(t *testing.T) {
	sig := *byte_helpers.BurnWithSig.Transaction.Signature
	task := NewMineMicroTask(0, sig)
	require.Equal(t, MINE_MICRO, task.Type())

	ch := mkch()
	task.Run(context.Background(), ch)

	require.Equal(t, sig, (<-ch).Data.(crypto.Signature))
}
