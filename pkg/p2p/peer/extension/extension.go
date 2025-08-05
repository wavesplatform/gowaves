package extension

import (
	"log/slog"

	"github.com/wavesplatform/gowaves/pkg/crypto"
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
	logger *slog.Logger
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

func NewPeerExtension(p peer.Peer, scheme proto.Scheme, logger *slog.Logger) PeerExtension {
	return PeerWrapperImpl{p: p, scheme: scheme, logger: logger}
}

func (a PeerWrapperImpl) AskBlocksIDs(ids []proto.BlockID) {
	if len(ids) == 0 {
		a.logger.Debug("No block IDs to request", "peer", a.p.ID().String())
		return
	}
	if a.p.Handshake().Version.Cmp(peerVersionWithProtobuf) < 0 {
		sigs := make([]crypto.Signature, len(ids))
		for i, b := range ids {
			sigs[i] = b.Signature()
		}
		a.logger.Debug("Requesting signatures", "peer", a.p.ID().String(), "from", sigs[0].ShortString(),
			"to", sigs[len(sigs)-1].ShortString())
		a.p.SendMessage(&proto.GetSignaturesMessage{Signatures: sigs})
	} else {
		a.logger.Debug("Requesting blocks IDs", "peer", a.p.ID().String(), "from", ids[0].ShortString(),
			"to", ids[len(ids)-1].ShortString())
		a.p.SendMessage(&proto.GetBlockIDsMessage{Blocks: ids})
	}
}

func (a PeerWrapperImpl) AskBlock(id proto.BlockID) {
	a.logger.Debug("Requesting block", "peer", a.p.ID().String(), "blockID", id.ShortString())
	a.p.SendMessage(&proto.GetBlockMessage{BlockID: id})
}

func (a PeerWrapperImpl) AskBlockSnapshot(id proto.BlockID) {
	a.logger.Debug("Requesting block snapshot", "peer", a.p.ID().String(), "blockID", id.ShortString())
	a.p.SendMessage(&proto.GetBlockSnapshotMessage{BlockID: id})
}

func (a PeerWrapperImpl) AskMicroBlockSnapshot(id proto.BlockID) {
	a.logger.Debug("Requesting microblock snapshot", "peer", a.p.ID().String(), "blockID", id.ShortString())
	a.p.SendMessage(&proto.MicroBlockSnapshotRequestMessage{BlockID: id})
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
