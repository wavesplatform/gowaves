package state_fsm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSyncFsm_Sync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManager := mock.NewMockPeerManager(ctrl)
	mockPeer := mock.NewMockPeer(ctrl)
	mockState := mock.NewMockState(ctrl)

	mockState.EXPECT().Height().Return(proto.Height(0), nil)

	mockPeer.EXPECT().Handshake().Return(proto.Handshake{Version: proto.NewVersion(1, 2, 0)})
	mockPeer.EXPECT().SendMessage(gomock.Any())

	fsm, async, err := NewIdleToSyncTransition(
		BaseInfo{
			peers:   mockManager,
			storage: mockState},
		mockPeer)
	require.NoError(t, err)
	require.NotEmpty(t, fsm)
	require.Empty(t, async)
}

func TestSyncFsm_SignaturesTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p := mock.NewMockPeer(ctrl)

	conf := conf{peerSyncWith: p}
	fsm, async, err := NewSyncFsm(BaseInfo{tm: ntptime.Stub{}}, conf, sync_internal.Internal{})
	require.NoError(t, err)
	require.Len(t, async, 0)
	require.NotNil(t, fsm)

	fsm, _, _ = fsm.Task(AsyncTask{
		TaskType: Ping,
	})

	require.IsType(t, &IdleFsm{}, fsm)
}
