package utxpool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func TestBulkValidator_Validate(t *testing.T) {
	emptyBlock := &proto.Block{}
	now := time.Now()

	m := state.NewMockState(t)
	m.EXPECT().TopBlock().Return(emptyBlock)
	m.EXPECT().TxValidation(mock.Anything).Return(nil).Times(1)
	utx := New(10000, NoOpValidator{}, settings.MustMainNetSettings())
	require.NoError(t, utx.AddWithBytesRaw(byte_helpers.TransferWithSig.Transaction,
		byte_helpers.TransferWithSig.TransactionBytes))
	validator := newBulkValidator(m, utx, tm(now), proto.TestNetScheme)
	ctx := context.Background()
	validator.Validate(ctx)
}
