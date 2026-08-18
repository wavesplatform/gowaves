package peers

import (
	"log/slog"
	"testing"
	"time"

	"github.com/wavesplatform/gowaves/pkg/node/peers/storage"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestPeerManagerImpl_BlackList(t *testing.T) {
	now := time.Now()
	tcpAddr := proto.NewTCPAddrFromString("32.34.46.1:4535")
	reason := "some-reason"

	p := peer.NewMockPeer(t)
	p.EXPECT().ID().Return(nil)
	p.EXPECT().Close().Return(nil)
	p.EXPECT().RemoteAddr().Return(tcpAddr)
	p.EXPECT().ID().Return(nil)

	peerStorage := NewMockPeerStorage(t)
	peerStorage.EXPECT().AddToBlackList([]storage.BlackListedPeer{{
		IP:                      storage.IpFromIpPort(tcpAddr.ToIpPort()),
		RestrictTimestampMillis: now.UnixMilli(),
		RestrictDuration:        restrictionDuration,
		Reason:                  reason,
	}}).Return(nil)

	manager := PeerManagerImpl{
		peerStorage:       peerStorage,
		logger:            slog.New(slog.DiscardHandler),
		blackListDuration: restrictionDuration,
	}

	manager.AddToBlackList(p, now, reason)
}
