package retransmit

import (
	"context"
	"log/slog"
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
	logger       *slog.Logger
}

func NewPeerSpawner(
	skipFunc conn.SkipFilter, parent peer.Parent, wavesNetwork string, declAddr proto.TCPAddr, logger *slog.Logger,
) *PeerOutgoingSpawnerImpl {
	return &PeerOutgoingSpawnerImpl{
		skipFunc:     skipFunc,
		parent:       parent,
		wavesNetwork: wavesNetwork,
		declAddr:     declAddr,
		logger:       logger,
	}
}

func (a *PeerOutgoingSpawnerImpl) SpawnOutgoing(ctx context.Context, address string) {
	params := network.OutgoingPeerParams{
		Address:      address,
		WavesNetwork: a.wavesNetwork,
		Parent:       a.parent,
		Skip:         a.skipFunc,
		DeclAddr:     a.declAddr,
		Logger:       a.logger,
		DataLogger:   a.logger,
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
		Logger:       a.logger,
		DataLogger:   a.logger,
	}

	network.RunIncomingPeer(ctx, params)
}
