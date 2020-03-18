package state_fsm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/mock"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
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

type lastSignaturesMock struct {
}

func (lastSignaturesMock) LastSignatures(state state.State) (*signatures.Signatures, error) {
	return signatures.NewSignatures(), nil
}

func TestSyncFsm_SignaturesTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p := mock.NewMockPeer(ctrl)
	p.EXPECT().SendMessage(gomock.Any())

	fsm, async, err := NewSyncFsmExtended(BaseInfo{tm: ntptime.Stub{}}, p, lastSignaturesMock{}, 0)
	require.NoError(t, err)
	require.Len(t, async, 0)
	require.NotNil(t, fsm)

	fsm, _, _ = fsm.Task(AsyncTask{
		TaskType: PING,
	})

	require.IsType(t, &IdleFsm{}, fsm)

}
