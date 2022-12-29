package state_fsm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
)

func TestDefaultImpl_PeerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := DefaultImpl{}

	peer := mock.NewMockPeer(ctrl)

	t.Run("has more connections", func(t *testing.T) {
		manager := mock.NewMockPeerManager(ctrl)
		manager.EXPECT().Disconnect(peer)
		manager.EXPECT().ConnectedCount().Return(1)
		fsm, async, err := d.PeerError(nil, peer, BaseInfo{peers: manager}, nil)
		require.NoError(t, err)
		require.Nil(t, fsm)
		require.Nil(t, async)
	})

	t.Run("has no connections", func(t *testing.T) {
		manager := mock.NewMockPeerManager(ctrl)
		manager.EXPECT().Disconnect(peer)
		manager.EXPECT().ConnectedCount().Return(0)
		fsm, async, err := d.PeerError(nil, peer, BaseInfo{peers: manager, skipMessageList: &messages.SkipMessageList{}}, nil)
		require.NoError(t, err)
		require.IsType(t, &IdleFsm{}, fsm)
		require.Nil(t, async)
	})
}

func TestDefaultImpl_Noop(t *testing.T) {
	fsm, async, err := DefaultImpl{}.Noop(nil)
	require.Nil(t, fsm)
	require.Nil(t, async)
	require.Nil(t, err)
}

func TestAskPeersInterval(t *testing.T) {
	require.LessOrEqual(t, askPeersInterval, conn.MaxConnIdleIODuration)
}
