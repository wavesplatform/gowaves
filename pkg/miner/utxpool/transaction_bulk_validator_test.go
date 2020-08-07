package utxpool

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func TestBulkValidator_Validate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	emptyBlock := &proto.Block{}
	now := time.Now()

	m := NewMockstateWrapper(ctrl)
	m.EXPECT().TopBlock().Return(emptyBlock)
	m.EXPECT().Map(gomock.Any()).Return(nil)
	utx := New(10000, NoOpValidator{}, settings.MainNetSettings)
	require.NoError(t, utx.AddWithBytes(byte_helpers.TransferWithSig.Transaction, byte_helpers.TransferWithSig.TransactionBytes))

	validator := newBulkValidator(m, utx, tm(now))
	validator.Validate()
}
