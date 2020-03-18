package sync_internal_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/mock"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
)

var sig1 = crypto.MustSignatureFromBase58("5syuWANDSgk8KyPxq2yQs2CYV23QfnrBoZMSv2LaciycxDYfBw6cLA2SqVnonnh1nFiFumzTgy2cPETnE7ZaZg5P")
var sig2 = crypto.MustSignatureFromBase58("3kvbjSovZWLg1zdMyW5vGsCj1DR1jkHY3ALtu5VxoqscrXQq3nH2vS2V5dhVo6ff9bxtbFAkUkVQQqCFUAHmwnpX")

func TestSigFSM_Signatures(t *testing.T) {
	or := ordered_blocks.NewOrderedBlocks()
	sigs := signatures.NewSignatures()

	t.Run("error on receive unexpected signatures", func(t *testing.T) {
		fsm := NewSigFSM(or, sigs, NoSignaturesExpected, false)
		rs2, err := fsm.Signatures(nil, []crypto.Signature{sig1, sig2})
		require.Equal(t, NoSignaturesExpectedErr, err)
		require.NotNil(t, rs2)
	})

	t.Run("successful receive signatures", func(t *testing.T) {
		fsm := NewSigFSM(or, sigs, WaitingForSignatures, false)
		rs2, err := fsm.Signatures(mock.NoOpPeer{}, []crypto.Signature{sig1, sig2})
		require.NoError(t, err)
		require.NotNil(t, rs2)
		require.True(t, rs2.NearEnd())
	})
}
