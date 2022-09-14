package peer_manager

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestPeerManagerImpl_Suspend(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now()
	tcpAddr := proto.NewTCPAddrFromString("32.34.46.1:4535")
	reason := "some-reason"

	p := mock.NewMockPeer(ctrl)
	gomock.InOrder(
		p.EXPECT().ID(),
		p.EXPECT().Close(),
		p.EXPECT().RemoteAddr().Return(tcpAddr),
		p.EXPECT().ID(),
	)

	peerStorage := mock.NewMockPeerStorage(ctrl)
	peerStorage.EXPECT().AddSuspended([]storage.SuspendedPeer{{
		IP:                      storage.IpFromIpPort(tcpAddr.ToIpPort()),
		RestrictTimestampMillis: now.UnixMilli(),
		RestrictDuration:        suspendDuration,
		Reason:                  reason,
	}})

	manager := PeerManagerImpl{
		peerStorage: peerStorage,
	}

	manager.Suspend(p, now, reason)
}
