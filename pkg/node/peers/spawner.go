package peers

import (
	"context"
	"log/slog"
	"net"
	"slices"

	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/incoming"
	"github.com/wavesplatform/gowaves/pkg/p2p/outgoing"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type DuplicateChecker interface {
	Add([]byte) bool
}

func NewSkipFilter(list *messages.SkipMessageList) conn.SkipFilter {
	return func(header proto.Header) bool {
		return func(h proto.Header, l *messages.SkipMessageList) bool {
			return slices.Contains(l.List(), h.ContentID)
		}(header, list)
	}
}

type PeerSpawner interface {
	SpawnOutgoing(ctx context.Context, addr proto.TCPAddr) error
	SpawnIncoming(ctx context.Context, c net.Conn) error
}

type PeerSpawnerImpl struct {
	parent       peer.Parent
	wavesNetwork string
	declAddr     proto.TCPAddr
	skipFunc     conn.SkipFilter
	nodeName     string
	nodeNonce    uint64
	version      proto.Version
	logger       *slog.Logger
	dataLogger   *slog.Logger
}

func NewPeerSpawner(
	parent peer.Parent, wavesNetwork string, declAddr proto.TCPAddr, nodeName string, nodeNonce uint64,
	version proto.Version, logger, dl *slog.Logger,
) *PeerSpawnerImpl {
	return &PeerSpawnerImpl{
		skipFunc:     NewSkipFilter(parent.SkipMessageList),
		parent:       parent,
		wavesNetwork: wavesNetwork,
		declAddr:     declAddr,
		nodeName:     nodeName,
		nodeNonce:    nodeNonce,
		version:      version,
		logger:       logger,
		dataLogger:   dl,
	}
}

func (a *PeerSpawnerImpl) SpawnOutgoing(ctx context.Context, address proto.TCPAddr) error {
	params := outgoing.EstablishParams{
		Address:      address,
		WavesNetwork: a.wavesNetwork,
		Parent:       a.parent,
		DeclAddr:     a.declAddr,
		Skip:         a.skipFunc,
		NodeName:     a.nodeName,
		NodeNonce:    a.nodeNonce,
	}

	return outgoing.EstablishConnection(ctx, params, a.version, a.logger, a.dataLogger)
}

func (a *PeerSpawnerImpl) SpawnIncoming(ctx context.Context, c net.Conn) error {
	params := incoming.PeerParams{
		WavesNetwork: a.wavesNetwork,
		Conn:         c,
		Skip:         a.skipFunc,
		Parent:       a.parent,
		DeclAddr:     a.declAddr,
		NodeName:     a.nodeName,
		NodeNonce:    a.nodeNonce,
		Version:      a.version,
	}

	return incoming.RunIncomingPeer(ctx, params, a.logger, a.dataLogger)
}
