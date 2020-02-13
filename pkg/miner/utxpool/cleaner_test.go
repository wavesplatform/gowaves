package utxpool

import (
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

func TestNewCleaner(t *testing.T) {
	require.NotNil(t, NewCleaner(services.Services{}))
}

func TestCleaner_work(t *testing.T) {
	block := &proto.Block{BlockHeader: proto.BlockHeader{
		BlockSignature: crypto.Signature{},
	}}
	m, err := node.NewMockStateManager(block)
	require.NoError(t, err)
	c := NewCleaner(services.Services{State: m, UtxPool: New(1000, NoOpValidator{}), Time: ntptime.Stub{}})
	c.work()
}

func TestCleaner_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mu := lock.NewRwMutex(&sync.RWMutex{})

	m := NewMockstateWrapper(ctrl)
	m.EXPECT().Mutex().Return(mu)
	m.EXPECT().Height().Return(uint64(0), errors.New("some err"))

	c := newCleaner(m, noOnBulkValidator{})
	c.Handle()
}
