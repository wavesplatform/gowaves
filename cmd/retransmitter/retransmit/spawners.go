package retransmit

import (
	"context"
	"net"

	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/network"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerSpawner interface {
	SpawnOutgoing(ctx context.Context, address string)
	SpawnIncoming(ctx context.Context, c net.Conn)
}

type PeerOutgoingSpawnerImpl struct {
	parent       peer.Parent
	wavesNetwork string
	declAddr     proto.TCPAddr
	skipFunc     conn.SkipFilter
}

func NewPeerSpawner(skipFunc conn.SkipFilter, parent peer.Parent, WavesNetwork string, declAddr proto.TCPAddr) *PeerOutgoingSpawnerImpl {
	return &PeerOutgoingSpawnerImpl{
		skipFunc:     skipFunc,
		parent:       parent,
		wavesNetwork: WavesNetwork,
		declAddr:     declAddr,
	}
}

func (a *PeerOutgoingSpawnerImpl) SpawnOutgoing(ctx context.Context, address string) {
	params := network.OutgoingPeerParams{
		Address:      address,
		WavesNetwork: a.wavesNetwork,
		Parent:       a.parent,
		Skip:         a.skipFunc,
		DeclAddr:     a.declAddr,
	}

	network.RunOutgoingPeer(ctx, params)
}

func (a *PeerOutgoingSpawnerImpl) SpawnIncoming(ctx context.Context, c net.Conn) {
	params := network.IncomingPeerParams{
		WavesNetwork: a.wavesNetwork,
		Conn:         c,
		Skip:         a.skipFunc,
		Parent:       a.parent,
		DeclAddr:     a.declAddr,
	}

	network.RunIncomingPeer(ctx, params)
}
