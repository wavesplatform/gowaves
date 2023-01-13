package extension

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var peerVersionWithProtobuf = proto.NewVersion(1, 2, 0)

type PeerExtension interface {
	AskBlocksIDs(id []proto.BlockID)
	AskBlock(id proto.BlockID)
	SendMicroBlock(micro *proto.MicroBlock) error
	SendTransaction(t proto.Transaction) error
}

type PeerWrapperImpl struct {
	p      peer.Peer
	scheme proto.Scheme
}

func (a PeerWrapperImpl) SendTransaction(t proto.Transaction) error {
	if a.p.Handshake().Version.Cmp(peerVersionWithProtobuf) < 0 {
		bts, err := t.MarshalBinary(a.scheme)
		if err != nil {
			return err
		}
		a.p.SendMessage(&proto.TransactionMessage{Transaction: bts})
	} else {
		bts, err := t.MarshalSignedToProtobuf(a.scheme)
		if err != nil {
			return err
		}
		a.p.SendMessage(&proto.PBTransactionMessage{Transaction: bts})
	}
	return nil
}

func NewPeerExtension(p peer.Peer, scheme proto.Scheme) PeerExtension {
	return PeerWrapperImpl{p: p, scheme: scheme}
}

func (a PeerWrapperImpl) AskBlocksIDs(ids []proto.BlockID) {
	if a.p.Handshake().Version.Cmp(peerVersionWithProtobuf) < 0 {
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

func (a PeerWrapperImpl) SendMicroBlock(micro *proto.MicroBlock) error {
	if a.p.Handshake().Version.Cmp(peerVersionWithProtobuf) < 0 {
		bts, err := micro.MarshalBinary(a.scheme)
		if err != nil {
			return err
		}
		a.p.SendMessage(&proto.MicroBlockMessage{Body: bts})
	} else {
		bts, err := micro.MarshalToProtobuf(a.scheme)
		if err != nil {
			return err
		}
		a.p.SendMessage(&proto.PBMicroBlockMessage{MicroBlockBytes: bts})
	}
	return nil
}
