package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/node/peers/outgoing"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerSpawner interface {
	SpawnOutgoing(ctx context.Context, addr proto.NodeAddr) error
	//SpawnIncoming(ctx context.Context, c net.Conn)
}

type PeerSpawnerImpl struct {
	pool         bytespool.Pool
	parent       peer.Parent
	wavesNetwork string
	declAddr     proto.PeerInfo
	skipFunc     conn.SkipFilter
	nodeName     string
	nodeNonce    uint64
	version      proto.Version
}

func NewPeerSpawner(pool bytespool.Pool, skipFunc conn.SkipFilter, parent peer.Parent, WavesNetwork string, declAddr proto.PeerInfo, nodeName string, nodeNonce uint64, version proto.Version) *PeerSpawnerImpl {
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

func (a *PeerSpawnerImpl) SpawnOutgoing(ctx context.Context, address proto.NodeAddr) error {
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

//
//func (a *PeerSpawnerImpl) SpawnIncoming(ctx context.Context, c net.Conn) {
//	params := peer.IncomingPeerParams{
//		WavesNetwork: a.wavesNetwork,
//		Conn:         c,
//		Skip:         a.skipFunc,
//		Parent:       a.parent,
//		DeclAddr:     a.declAddr,
//		Pool:         a.pool,
//	}
//
//	peer.RunIncomingPeer(ctx, params)
//}
