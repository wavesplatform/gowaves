package utxpool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

type tm time.Time

func (t tm) Now() time.Time {
	return time.Time(t)
}

func TestValidatorImpl_Validate(t *testing.T) {
	emptyBlock := &proto.Block{}
	emptyBlock.Timestamp = proto.NewTimestampFromTime(time.Now())
	now := time.Now()

	m := state.NewMockState(t)
	v, err := NewValidator(tm(now), 24*time.Hour)
	require.NoError(t, err)

	m.EXPECT().TopBlock().Return(emptyBlock)
	m.EXPECT().TxValidation(mock.Anything).Return(nil).Times(1)

	err = v.Validate(m, byte_helpers.BurnWithSig.Transaction)
	require.NoError(t, err)
}
