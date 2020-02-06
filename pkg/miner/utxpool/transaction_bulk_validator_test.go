package utxpool

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

func TestBulkValidator_Validate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	emptyBlock := &proto.Block{}
	mu := lock.NewRwMutex(&sync.RWMutex{})
	now := time.Now()

	m := NewMockstateWrapper(ctrl)
	m.EXPECT().Mutex().Return(mu)
	m.EXPECT().TopBlock().Return(emptyBlock, nil)
	m.EXPECT(). // first transaction returns err
			ValidateNextTx(byte_helpers.TransferV1.Transaction, gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("some err"))
	m.EXPECT(). // second returns ok
			ValidateNextTx(byte_helpers.BurnV1.Transaction, gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)
	m.EXPECT().ResetValidationList()

	utx := New(10000, NoOpValidator{})
	require.NoError(t, utx.AddWithBytes(byte_helpers.TransferV1.Transaction, byte_helpers.TransferV1.TransactionBytes))
	require.NoError(t, utx.AddWithBytes(byte_helpers.BurnV1.Transaction, byte_helpers.BurnV1.TransactionBytes))
	require.Equal(t, 2, utx.Len())

	validator := newBulkValidator(m, utx, tm(now))
	validator.Validate()

	require.Equal(t, 1, utx.Len())
}
