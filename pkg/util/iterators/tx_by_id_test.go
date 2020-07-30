package iterators

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func TestTxByIdIterator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ids := [][]byte{{'a'}}

	stateMock := mock.NewMockState(ctrl)
	stateMock.EXPECT().
		TransactionByIDWithStatus([]byte{'a'}).
		Return(byte_helpers.BurnWithProofs.Transaction, true, nil)

	iter := NewTxByIdIterator(stateMock, ids)
	require.True(t, iter.Next())
	tx, _, _ := iter.Transaction()
	require.Equal(t, byte_helpers.BurnWithProofs.Transaction, tx)

	require.False(t, iter.Next())

	iter.Release()
	require.NoError(t, iter.Error())
}
