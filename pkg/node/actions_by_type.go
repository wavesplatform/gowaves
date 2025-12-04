package node

import (
	"fmt"
	"log/slog"
	"math/big"
	"reflect"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/node/fsm"
	"github.com/wavesplatform/gowaves/pkg/node/peers/storage"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

type Action func(services services.Services, msg peer.ProtoMessage, fsm *fsm.FSM, nl *slog.Logger) (fsm.Async, error)

func ScoreAction(_ services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	b := new(big.Int)
	b.SetBytes(mess.Message.(*proto.ScoreMessage).Score)
	return fsm.Score(mess.ID, b)
}

func GetPeersAction(services services.Services, mess peer.ProtoMessage, _ *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	metricGetPeersMessage.Inc()
	rs := services.Peers.KnownPeers()

	var out []proto.PeerInfo
	for _, r := range rs {
		ipPort := proto.IpPort(r)
		out = append(out, proto.PeerInfo{
			Addr: ipPort.Addr(),
			Port: uint16(ipPort.Port()),
		})
	}
	mess.ID.SendMessage(&proto.PeersMessage{Peers: out})
	return nil, nil
}

func PeersAction(services services.Services, mess peer.ProtoMessage, _ *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	metricPeersMessage.Inc()
	rs := services.Peers.KnownPeers()

	m := mess.Message.(*proto.PeersMessage).Peers
	if len(m) == 0 {
		return nil, nil
	}
	for _, p := range m {
		known := storage.KnownPeer(proto.NewTCPAddr(p.Addr, int(p.Port)).ToIpPort())
		rs = append(rs, known)
	}
	return nil, services.Peers.UpdateKnownPeers(rs)
}

func BlockAction(services services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	metricBlockMessage.Inc()
	b := &proto.Block{}
	err := b.UnmarshalBinary(mess.Message.(*proto.BlockMessage).BlockBytes, services.Scheme)
	if err != nil {
		return nil, err
	}
	return fsm.Block(mess.ID, b)
}

func GetBlockAction(services services.Services, mess peer.ProtoMessage, _ *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	metricGetBlockMessage.Inc()
	block, err := services.State.Block(mess.Message.(*proto.GetBlockMessage).BlockID)
	if err != nil {
		return nil, err
	}
	bm, err := proto.MessageByBlock(block, services.Scheme)
	if err != nil {
		return nil, err
	}
	mess.ID.SendMessage(bm)
	return nil, nil
}

// SignaturesAction receives requested earlier signatures
func SignaturesAction(_ services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	signatures := mess.Message.(*proto.SignaturesMessage).Signatures
	blockIDs := make([]proto.BlockID, len(signatures))
	for i, sig := range signatures {
		blockIDs[i] = proto.NewBlockIDFromSignature(sig)
	}
	return fsm.BlockIDs(mess.ID, blockIDs)
}

// GetSignaturesAction replies to signature requests
func GetSignaturesAction(
	services services.Services, mess peer.ProtoMessage, _ *fsm.FSM, _ *slog.Logger,
) (fsm.Async, error) {
	for _, sig := range mess.Message.(*proto.GetSignaturesMessage).Signatures {
		block, err := services.State.Header(proto.NewBlockIDFromSignature(sig))
		if err != nil {
			continue
		}
		sendSignatures(services, block, mess.ID)
		break
	}
	return nil, nil
}

func sendSignatures(services services.Services, block *proto.BlockHeader, p peer.Peer) {
	height, err := services.State.BlockIDToHeight(block.BlockID())
	if err != nil {
		slog.Error("Failed to get height for blockID and send signatures to peer",
			slog.String("blockID", block.BlockID().String()), slog.String("peer", p.RemoteAddr().String()),
			logging.Error(err))
		return
	}

	var out []crypto.Signature
	out = append(out, block.BlockSignature)

	for i := 1; i < 101; i++ {
		b, err := services.State.HeaderByHeight(height + uint64(i))
		if err != nil {
			break
		}
		out = append(out, b.BlockSignature)
	}

	// There are block signatures to send in addition to requested one
	if len(out) > 1 {
		p.SendMessage(&proto.SignaturesMessage{
			Signatures: out,
		})
	}
}

func sendBlockIds(services services.Services, block *proto.BlockHeader, p peer.Peer) {
	height, err := services.State.BlockIDToHeight(block.BlockID())
	if err != nil {
		slog.Error("Failed to get height for blockID and send blockIDs to peer",
			slog.String("blockID", block.BlockID().String()), slog.String("peer", p.RemoteAddr().String()),
			logging.Error(err))
		return
	}

	var out []proto.BlockID
	out = append(out, block.BlockID())

	for i := 1; i < 101; i++ {
		b, err := services.State.HeaderByHeight(height + uint64(i))
		if err != nil {
			break
		}
		out = append(out, b.BlockID())
	}

	// There are block signatures to send in addition to requested one
	if len(out) > 1 {
		p.SendMessage(&proto.BlockIDsMessage{
			Blocks: out,
		})
	}
}

// MicroBlockInvAction handles notification about new microblock.
func MicroBlockInvAction(_ services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	inv := &proto.MicroBlockInv{}
	err := inv.UnmarshalBinary(mess.Message.(*proto.MicroBlockInvMessage).Body)
	if err != nil {
		return nil, err
	}
	return fsm.MicroBlockInv(mess.ID, inv)
}

// MicroBlockRequestAction handles microblock requests.
func MicroBlockRequestAction(
	services services.Services, mess peer.ProtoMessage, _ *fsm.FSM, nl *slog.Logger,
) (fsm.Async, error) {
	msg, ok := mess.Message.(*proto.MicroBlockRequestMessage)
	if !ok {
		return nil, fmt.Errorf("unexpected message type %T", mess.Message)
	}
	micro, ok := services.MicroBlockCache.GetBlock(msg.TotalBlockSig)
	if ok {
		_ = extension.NewPeerExtension(mess.ID, services.Scheme, nl).SendMicroBlock(micro)
	}
	return nil, nil
}

func MicroBlockAction(
	services services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, _ *slog.Logger,
) (fsm.Async, error) {
	micro := &proto.MicroBlock{}
	err := micro.UnmarshalBinary(mess.Message.(*proto.MicroBlockMessage).Body, services.Scheme)
	if err != nil {
		return nil, err
	}
	return fsm.MicroBlock(mess.ID, micro)
}

// PBBlockAction handles protobuf block message.
func PBBlockAction(_ services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, nl *slog.Logger) (fsm.Async, error) {
	b := &proto.Block{}
	if err := b.UnmarshalFromProtobuf(mess.Message.(*proto.PBBlockMessage).PBBlockBytes); err != nil {
		nl.Debug("Failed to deserialize protobuf block", logging.Error(err))
		return nil, err
	}
	nl.Debug("Protobuf block received", "blockID", b.ID.String())
	return fsm.Block(mess.ID, b)
}

func PBMicroBlockAction(
	_ services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, nl *slog.Logger,
) (fsm.Async, error) {
	micro := &proto.MicroBlock{}
	if err := micro.UnmarshalFromProtobuf(mess.Message.(*proto.PBMicroBlockMessage).MicroBlockBytes); err != nil {
		nl.Debug("Failed to deserialize microblock", logging.Error(err))
		return nil, err
	}
	nl.Debug("Microblock received", "blockID", micro.TotalBlockID.String())
	return fsm.MicroBlock(mess.ID, micro)
}

func GetBlockIdsAction(
	services services.Services, mess peer.ProtoMessage, _ *fsm.FSM, _ *slog.Logger,
) (fsm.Async, error) {
	msg, ok := mess.Message.(*proto.GetBlockIDsMessage)
	if !ok {
		return nil, fmt.Errorf("unexpected message type %T", mess.Message)
	}
	for _, id := range msg.Blocks {
		block, err := services.State.Header(id)
		if err != nil {
			continue
		}
		sendBlockIds(services, block, mess.ID)
		break
	}
	return nil, nil
}

func BlockIDsAction(_ services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	msg, ok := mess.Message.(*proto.BlockIDsMessage)
	if !ok {
		return nil, fmt.Errorf("unexpected message type %T", mess.Message)
	}
	return fsm.BlockIDs(mess.ID, msg.Blocks)
}

// TransactionAction handles new transaction message.
func TransactionAction(s services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	b := mess.Message.(*proto.TransactionMessage).Transaction
	tx, err := proto.BytesToTransaction(b, s.Scheme)
	if err != nil {
		return nil, err
	}
	return fsm.Transaction(mess.ID, tx)
}

// PBTransactionAction handles protobuf transaction message.
func PBTransactionAction(_ services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, _ *slog.Logger) (fsm.Async, error) {
	b := mess.Message.(*proto.PBTransactionMessage).Transaction
	t, err := proto.SignedTxFromProtobuf(b)
	if err != nil {
		return nil, err
	}
	// TODO add transaction re-broadcast
	return fsm.Transaction(mess.ID, t)
}

func MicroSnapshotRequestAction(
	services services.Services, mess peer.ProtoMessage, _ *fsm.FSM, _ *slog.Logger,
) (fsm.Async, error) {
	msg, ok := mess.Message.(*proto.MicroBlockSnapshotRequestMessage)
	if !ok {
		return nil, fmt.Errorf("unexpected message type %T", mess.Message)
	}
	sn, ok := services.MicroBlockCache.GetSnapshot(msg.BlockID)
	if ok {
		snapshotProto, errToProto := sn.ToProtobuf()
		if errToProto != nil {
			return nil, errToProto
		}
		sProto := g.MicroBlockSnapshot{
			Snapshots:    snapshotProto,
			TotalBlockId: msg.BlockID.Bytes(),
		}
		bsmBytes, errMarshall := sProto.MarshalVTStrict()
		if errMarshall != nil {
			return nil, errMarshall
		}
		bs := proto.MicroBlockSnapshotMessage{Bytes: bsmBytes}
		mess.ID.SendMessage(&bs)
	}
	return nil, nil
}

func GetSnapshotAction(
	services services.Services, mess peer.ProtoMessage, _ *fsm.FSM, _ *slog.Logger,
) (fsm.Async, error) {
	blockID := mess.Message.(*proto.GetBlockSnapshotMessage).BlockID
	h, err := services.State.BlockIDToHeight(blockID)
	if err != nil {
		return nil, err
	}
	snapshot, err := services.State.SnapshotsAtHeight(h)
	if err != nil {
		return nil, err
	}
	snapshotProto, err := snapshot.ToProtobuf()
	if err != nil {
		return nil, err
	}
	sProto := g.BlockSnapshot{
		Snapshots: snapshotProto,
		BlockId:   blockID.Bytes(),
	}
	bsmBytes, err := sProto.MarshalVTStrict()
	if err != nil {
		return nil, err
	}
	bs := proto.BlockSnapshotMessage{Bytes: bsmBytes}
	mess.ID.SendMessage(&bs)
	return nil, nil
}

func BlockSnapshotAction(
	services services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, nl *slog.Logger,
) (fsm.Async, error) {
	protoMess := g.BlockSnapshot{}
	if err := protoMess.UnmarshalVT(mess.Message.(*proto.BlockSnapshotMessage).Bytes); err != nil {
		nl.Debug("Failed to deserialize block snapshot", logging.Error(err))
		return nil, err
	}
	blockID, err := proto.NewBlockIDFromBytes(protoMess.BlockId)
	if err != nil {
		return nil, err
	}
	blockSnapshot, err := proto.BlockSnapshotFromProtobuf(services.Scheme, protoMess.Snapshots)
	if err != nil {
		return nil, err
	}
	nl.Debug("Snapshot received", "blockID", blockID.String())
	return fsm.BlockSnapshot(mess.ID, blockID, blockSnapshot)
}

func MicroBlockSnapshotAction(
	services services.Services, mess peer.ProtoMessage, fsm *fsm.FSM, nl *slog.Logger,
) (fsm.Async, error) {
	protoMess := g.MicroBlockSnapshot{}
	if err := protoMess.UnmarshalVT(mess.Message.(*proto.MicroBlockSnapshotMessage).Bytes); err != nil {
		nl.Debug("Failed to deserialize micro block snapshot", logging.Error(err))
		return nil, err
	}
	blockID, err := proto.NewBlockIDFromBytes(protoMess.TotalBlockId)
	if err != nil {
		return nil, err
	}
	blockSnapshot, err := proto.BlockSnapshotFromProtobuf(services.Scheme, protoMess.Snapshots)
	if err != nil {
		return nil, err
	}
	return fsm.MicroBlockSnapshot(mess.ID, blockID, blockSnapshot)
}

func createActions() map[reflect.Type]Action {
	return map[reflect.Type]Action{
		reflect.TypeFor[*proto.ScoreMessage]():                     ScoreAction,
		reflect.TypeFor[*proto.GetPeersMessage]():                  GetPeersAction,
		reflect.TypeFor[*proto.PeersMessage]():                     PeersAction,
		reflect.TypeFor[*proto.BlockMessage]():                     BlockAction,
		reflect.TypeFor[*proto.GetBlockMessage]():                  GetBlockAction,
		reflect.TypeFor[*proto.SignaturesMessage]():                SignaturesAction,
		reflect.TypeFor[*proto.GetSignaturesMessage]():             GetSignaturesAction,
		reflect.TypeFor[*proto.MicroBlockInvMessage]():             MicroBlockInvAction,
		reflect.TypeFor[*proto.MicroBlockRequestMessage]():         MicroBlockRequestAction,
		reflect.TypeFor[*proto.MicroBlockMessage]():                MicroBlockAction,
		reflect.TypeFor[*proto.PBBlockMessage]():                   PBBlockAction,
		reflect.TypeFor[*proto.PBMicroBlockMessage]():              PBMicroBlockAction,
		reflect.TypeFor[*proto.GetBlockIDsMessage]():               GetBlockIdsAction,
		reflect.TypeFor[*proto.BlockIDsMessage]():                  BlockIDsAction,
		reflect.TypeFor[*proto.TransactionMessage]():               TransactionAction,
		reflect.TypeFor[*proto.PBTransactionMessage]():             PBTransactionAction,
		reflect.TypeFor[*proto.GetBlockSnapshotMessage]():          GetSnapshotAction,
		reflect.TypeFor[*proto.MicroBlockSnapshotRequestMessage](): MicroSnapshotRequestAction,
		reflect.TypeFor[*proto.BlockSnapshotMessage]():             BlockSnapshotAction,
		reflect.TypeFor[*proto.MicroBlockSnapshotMessage]():        MicroBlockSnapshotAction,
	}
}
