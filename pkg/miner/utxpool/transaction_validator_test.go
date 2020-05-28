package utxpool

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

type tm time.Time

func (t tm) Now() time.Time {
	return time.Time(t)
}

func TestValidatorImpl_Validate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	emptyBlock := &proto.Block{}
	emptyBlock.Timestamp = proto.NewTimestampFromTime(time.Now())
	now := time.Now()

	m := NewMockstateWrapper(ctrl)
	v := NewValidator(m, tm(now))

	m.EXPECT().TopBlock().Return(emptyBlock)
	m.EXPECT().IsActivated(gomock.Any()).Return(false, nil)
	m.EXPECT().
		TxValidation(gomock.Any())

	err := v.Validate(byte_helpers.BurnWithSig.Transaction)
	require.NoError(t, err)
}
