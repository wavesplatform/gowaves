package peer_manager

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
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
		p.EXPECT().ID().Return(peer.PeerID("some-id")),
		p.EXPECT().Close(),
		p.EXPECT().RemoteAddr().Return(tcpAddr),
		p.EXPECT().ID().Return(peer.PeerID("some-id")),
	)

	peerStorage := mock.NewMockPeerStorage(ctrl)
	peerStorage.EXPECT().AddSuspended([]storage.SuspendedPeer{{
		IP:                     storage.IpFromIpPort(tcpAddr.ToIpPort()),
		SuspendTimestampMillis: now.UnixMilli(),
		SuspendDuration:        suspendDuration,
		Reason:                 reason,
	}})

	manager := PeerManagerImpl{
		peerStorage: peerStorage,
	}

	manager.Suspend(p, now, reason)
}
