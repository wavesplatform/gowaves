package state_fsm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSyncFsm_Sync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManager := mock.NewMockPeerManager(ctrl)
	mockPeer := mock.NewMockPeer(ctrl)
	mockState := mock.NewMockState(ctrl)

	///
	mockManager.EXPECT().KnownPeers().Return([]proto.TCPAddr(nil), nil)
	mockPeer.EXPECT().SendMessage(gomock.Any())

	fsm, async, err := NewIdleToSyncTransition(
		BaseInfo{
			peers:   mockManager,
			storage: mockState},
		mockPeer)
	require.NoError(t, err)
	require.NotEmpty(t, fsm)
	require.NotEmpty(t, async)

}
