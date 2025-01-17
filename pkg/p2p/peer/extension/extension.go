package extension

import (
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var peerVersionWithProtobuf = proto.NewVersion(1, 2, 0)

type PeerExtension interface {
	AskBlocksIDs(id []proto.BlockID)
	AskBlock(id proto.BlockID)
	AskBlockSnapshot(id proto.BlockID)
	AskMicroBlockSnapshot(id proto.BlockID)
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
		zap.S().Named(logging.NetworkNamespace).Debugf("[%s] Requesting signatures for signatures range [%s...%s]",
			a.p.ID().String(), sigs[0].ShortString(), sigs[len(sigs)-1].ShortString())
		a.p.SendMessage(&proto.GetSignaturesMessage{Signatures: sigs})
	} else {
		zap.S().Named(logging.NetworkNamespace).Debugf("[%s] Requesting blocks IDs for IDs range [%s...%s]",
			a.p.ID().String(), ids[0].ShortString(), ids[len(ids)-1].ShortString())
		a.p.SendMessage(&proto.GetBlockIDsMessage{Blocks: ids})
	}
}

func (a PeerWrapperImpl) AskBlock(id proto.BlockID) {
	zap.S().Named(logging.NetworkNamespace).Debugf("[%s] Requesting block %s", a.p.ID().String(), id.ShortString())
	a.p.SendMessage(&proto.GetBlockMessage{BlockID: id})
}

func (a PeerWrapperImpl) AskBlockSnapshot(id proto.BlockID) {
	zap.S().Named(logging.NetworkNamespace).Debugf(
		"[%s] Requesting block snapshot for block %s", a.p.ID().String(), id.ShortString(),
	)
	a.p.SendMessage(&proto.GetBlockSnapshotMessage{BlockID: id})
}

func (a PeerWrapperImpl) AskMicroBlockSnapshot(id proto.BlockID) {
	zap.S().Named(logging.NetworkNamespace).Debugf(
		"[%s] Requesting micro block snapshot for micro block %s", a.p.ID().String(), id.ShortString(),
	)
	a.p.SendMessage(&proto.MicroBlockSnapshotRequestMessage{BlockIDBytes: id.Bytes()})
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
