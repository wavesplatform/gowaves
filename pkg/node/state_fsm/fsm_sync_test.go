package state_fsm

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

//go:generate moq -pkg state_fsm -out time_moq.go ../../types Time:MockTime

func TestSyncFsm_Sync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockManager := mock.NewMockPeerManager(ctrl)
	mockPeer := mock.NewMockPeer(ctrl)
	mockState := mock.NewMockState(ctrl)

	mockState.EXPECT().Height().Return(proto.Height(0), nil)

	mockPeer.EXPECT().Handshake().Return(proto.Handshake{Version: proto.NewVersion(1, 2, 0)})
	mockPeer.EXPECT().SendMessage(gomock.Any())

	baseInfo := BaseInfo{
		peers:   mockManager,
		storage: mockState,
		tm: &MockTime{
			NowFunc: func() time.Time {
				return time.Now()
			},
		},
		skipMessageList: &messages.SkipMessageList{},
	}
	lastSignatures, err := signatures.LastSignaturesImpl{}.LastBlockIDs(baseInfo.storage)
	require.NoError(t, err)
	internal := sync_internal.InternalFromLastSignatures(extension.NewPeerExtension(mockPeer, baseInfo.scheme), lastSignatures)
	c := conf{
		peerSyncWith: mockPeer,
		timeout:      30 * time.Second,
	}
	fsm, async, err := NewSyncFsm(baseInfo, c.Now(baseInfo.tm), internal)

	require.NoError(t, err)
	require.NotEmpty(t, fsm)
	require.Empty(t, async)
}

func TestSyncFsm_SignaturesTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p := mock.NewMockPeer(ctrl)
	p.EXPECT().ID()

	conf := conf{peerSyncWith: p}
	fsm, async, err := NewSyncFsm(BaseInfo{tm: ntptime.Stub{}, skipMessageList: &messages.SkipMessageList{}}, conf, sync_internal.Internal{})
	require.NoError(t, err)
	require.Len(t, async, 0)
	require.NotNil(t, fsm)

	fsm, _, _ = fsm.Task(tasks.AsyncTask{
		TaskType: tasks.Ping,
	})

	require.IsType(t, &IdleFsm{}, fsm)
}
