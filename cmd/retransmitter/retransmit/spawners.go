package retransmit

import (
	"context"
	"net"

	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerSpawner interface {
	SpawnOutgoing(ctx context.Context, address string)
	SpawnIncoming(ctx context.Context, c net.Conn)
}

type PeerOutgoingSpawnerImpl struct {
	pool                      conn.Pool
	receiveFromRemoteCallback peer.ReceiveFromRemoteCallback
	parent                    peer.Parent
	wavesNetwork              string
	declAddr                  proto.PeerInfo
}

func NewPeerSpawner(pool conn.Pool, ReceiveFromRemoteCallback peer.ReceiveFromRemoteCallback, parent peer.Parent, WavesNetwork string, declAddr proto.PeerInfo) *PeerOutgoingSpawnerImpl {
	return &PeerOutgoingSpawnerImpl{
		pool:                      pool,
		receiveFromRemoteCallback: ReceiveFromRemoteCallback,
		parent:                    parent,
		wavesNetwork:              WavesNetwork,
		declAddr:                  declAddr,
	}
}

func (a *PeerOutgoingSpawnerImpl) SpawnOutgoing(ctx context.Context, address string) {
	params := peer.OutgoingPeerParams{
		Address:                   address,
		WavesNetwork:              a.wavesNetwork,
		Parent:                    a.parent,
		ReceiveFromRemoteCallback: a.receiveFromRemoteCallback,
		Pool:                      a.pool,
		DeclAddr:                  a.declAddr,
	}

	peer.RunOutgoingPeer(ctx, params)
}

func (a *PeerOutgoingSpawnerImpl) SpawnIncoming(ctx context.Context, c net.Conn) {
	params := peer.IncomingPeerParams{
		WavesNetwork:              a.wavesNetwork,
		Conn:                      c,
		ReceiveFromRemoteCallback: a.receiveFromRemoteCallback,
		Parent:                    a.parent,
		DeclAddr:                  a.declAddr,
		Pool:                      a.pool,
	}

	peer.RunIncomingPeer(ctx, params)
}
