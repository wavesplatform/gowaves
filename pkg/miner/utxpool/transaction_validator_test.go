package utxpool

import (
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
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
	mu := lock.NewRwMutex(&sync.RWMutex{})
	now := time.Now()

	m := NewMockstateWrapper(ctrl)
	v := NewValidator(m, tm(now))

	m.EXPECT().Mutex().Return(mu)
	m.EXPECT().TopBlock().Return(emptyBlock)
	m.EXPECT().
		ValidateNextTx(byte_helpers.BurnWithSig.Transaction, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	m.EXPECT().ResetValidationList()

	err := v.Validate(byte_helpers.BurnWithSig.Transaction)
	require.NoError(t, err)
}
