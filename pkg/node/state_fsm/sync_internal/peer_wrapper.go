package sync_internal

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerWrapper interface {
	AskBlocksIDs(id []proto.BlockID)
	AskBlock(id proto.BlockID)
}

type PeerWrapperImpl struct {
	p peer.Peer
}

func NewPeerWrapper(p peer.Peer) PeerWrapper {
	return PeerWrapperImpl{p: p}
}

func (a PeerWrapperImpl) AskBlocksIDs(ids []proto.BlockID) {
	if a.p.Handshake().Version.Cmp(proto.NewVersion(1, 2, 0)) < 0 {
		sigs := make([]crypto.Signature, len(ids))
		for i, b := range ids {
			sigs[i] = b.Signature()
		}
		a.p.SendMessage(&proto.GetSignaturesMessage{Signatures: sigs})
	} else {
		a.p.SendMessage(&proto.GetBlockIdsMessage{Blocks: ids})
	}
}

func (a PeerWrapperImpl) AskBlock(id proto.BlockID) {
	a.p.SendMessage(&proto.GetBlockMessage{BlockID: id})
}
