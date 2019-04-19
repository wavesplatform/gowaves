package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/p2p/incoming"
	"github.com/wavesplatform/gowaves/pkg/p2p/outgoing"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
)

type PeerSpawner interface {
	SpawnOutgoing(ctx context.Context, addr proto.TCPAddr) error
	SpawnIncoming(ctx context.Context, c net.Conn)
}

type PeerSpawnerImpl struct {
	pool         bytespool.Pool
	parent       peer.Parent
	wavesNetwork string
	declAddr     proto.TCPAddr
	skipFunc     conn.SkipFilter
	nodeName     string
	nodeNonce    uint64
	version      proto.Version
}

func NewPeerSpawner(pool bytespool.Pool, skipFunc conn.SkipFilter, parent peer.Parent, WavesNetwork string, declAddr proto.TCPAddr, nodeName string, nodeNonce uint64, version proto.Version) *PeerSpawnerImpl {
	return &PeerSpawnerImpl{
		pool:         pool,
		skipFunc:     skipFunc,
		parent:       parent,
		wavesNetwork: WavesNetwork,
		declAddr:     declAddr,
		nodeName:     nodeName,
		nodeNonce:    nodeNonce,
		version:      version,
	}
}

func (a *PeerSpawnerImpl) SpawnOutgoing(ctx context.Context, address proto.TCPAddr) error {
	params := outgoing.EstablishParams{
		Address:      address,
		WavesNetwork: a.wavesNetwork,
		Parent:       a.parent,
		Pool:         a.pool,
		DeclAddr:     a.declAddr,
		Skip:         a.skipFunc,
		NodeName:     a.nodeName,
		NodeNonce:    a.nodeNonce,
	}

	return outgoing.EstablishConnection(ctx, params, a.version)
}

func (a *PeerSpawnerImpl) SpawnIncoming(ctx context.Context, c net.Conn) {
	params := incoming.IncomingPeerParams{
		WavesNetwork: a.wavesNetwork,
		Conn:         c,
		Skip:         a.skipFunc,
		Parent:       a.parent,
		DeclAddr:     a.declAddr,
		Pool:         a.pool,

		NodeName:  a.nodeName,
		NodeNonce: a.nodeNonce,
		Version:   a.version,
	}

	incoming.RunIncomingPeer(ctx, params)
}
