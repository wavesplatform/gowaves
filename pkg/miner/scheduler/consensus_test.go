package scheduler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/mock"
)

func TestMinerConsensus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock.NewMockPeerManager(ctrl)

	m.EXPECT().ConnectedCount().Return(1)
	a := NewMinerConsensus(m, 1)
	assert.True(t, a.IsMiningAllowed())

	m.EXPECT().ConnectedCount().Return(0)
	a = NewMinerConsensus(m, 1)
	assert.False(t, a.IsMiningAllowed())
}
