package node

import (
	"math/big"
	"net"
	"reflect"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"go.uber.org/zap"
)

type Action func(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error)

func ScoreAction(_ services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	b := new(big.Int)
	b.SetBytes(mess.Message.(*proto.ScoreMessage).Score)
	return fsm.Score(mess.ID, b)
}

func GetPeersAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	rs, err := services.Peers.KnownPeers()
	if err != nil {
		zap.L().Error("failed got known peers", zap.Error(err))
		return fsm, nil, err
	}
	var out []proto.PeerInfo
	for _, r := range rs {
		out = append(out, proto.PeerInfo{
			Addr: net.IP(r.IP[:]),
			Port: uint16(r.Port),
		})
	}
	mess.ID.SendMessage(&proto.PeersMessage{Peers: out})
	return fsm, nil, nil
}

func BlockAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	b := &proto.Block{}
	err := b.UnmarshalBinary(mess.Message.(*proto.BlockMessage).BlockBytes, services.Scheme)
	if err != nil {
		return fsm, nil, err
	}
	return fsm.Block(mess.ID, b)
}

func GetBlockAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	block, err := services.State.Block(mess.Message.(*proto.GetBlockMessage).BlockID)
	if err != nil {
		return fsm, nil, err
	}
	bm, err := proto.MessageByBlock(block, services.Scheme)
	if err != nil {
		return fsm, nil, err
	}
	mess.ID.SendMessage(bm)
	return fsm, nil, nil
}

// received asked earlier signatures
func SignaturesAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	sigs := mess.Message.(*proto.SignaturesMessage).Signatures
	blockIDs := make([]proto.BlockID, len(sigs))
	for i, sig := range sigs {
		blockIDs[i] = proto.NewBlockIDFromSignature(sig)
	}
	return fsm.BlockIDs(mess.ID, blockIDs)
}

// peers asks us about our signatures
func GetSignaturesAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	for _, sig := range mess.Message.(*proto.GetSignaturesMessage).Signatures {
		block, err := services.State.Header(proto.NewBlockIDFromSignature(sig))
		if err != nil {
			continue
		}
		sendSignatures(services, block, mess.ID)
		break
	}
	return fsm, nil, nil
}

func sendSignatures(services services.Services, block *proto.BlockHeader, p peer.Peer) {
	height, err := services.State.BlockIDToHeight(block.BlockID())
	if err != nil {
		zap.S().Error(err)
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

	// if we put smth except first block
	if len(out) > 1 {
		p.SendMessage(&proto.SignaturesMessage{
			Signatures: out,
		})
	}
}

func sendBlockIds(services services.Services, block *proto.BlockHeader, p peer.Peer) {
	height, err := services.State.BlockIDToHeight(block.BlockID())
	if err != nil {
		zap.S().Error(err)
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

	// if we put smth except first block
	if len(out) > 1 {
		p.SendMessage(&proto.BlockIdsMessage{
			Blocks: out,
		})
	}
}

// remote node mined microblock and sent us info about it.
func MicroBlockInvAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	inv := &proto.MicroBlockInv{}
	err := inv.UnmarshalBinary(mess.Message.(*proto.MicroBlockInvMessage).Body)
	if err != nil {
		return fsm, nil, err
	}
	return fsm.MicroBlockInv(mess.ID, inv)
}

// Our miner mined microblock, sent MicroblockInv to other nodes, they asked us about Microblock.
func MicroBlockRequestAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	blockID, err := proto.NewBlockIDFromBytes(mess.Message.(*proto.MicroBlockRequestMessage).TotalBlockSig)
	if err != nil {
		return fsm, nil, err
	}
	micro, ok := services.MicroBlockCache.Get(blockID)
	if ok {
		_ = extension.NewPeerExtension(mess.ID, services.Scheme).SendMicroBlock(micro)
	}
	return fsm, nil, nil
}

func MicroBlockAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	micro := &proto.MicroBlock{}
	err := micro.UnmarshalBinary(mess.Message.(*proto.MicroBlockMessage).Body, services.Scheme)
	if err != nil {
		return fsm, nil, err
	}
	return fsm.MicroBlock(mess.ID, micro)
}

// arrived protobuf block
func PBBlockAction(_ services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	b := &proto.Block{}
	err := b.UnmarshalFromProtobuf(mess.Message.(*proto.PBBlockMessage).PBBlockBytes)
	if err != nil {
		zap.S().Debug(err)
		return fsm, nil, err
	}
	zap.S().Info("PBBlockAction ", b.BlockSignature)
	return fsm.Block(mess.ID, b)
}

func PBMicroBlockAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	micro := &proto.MicroBlock{}
	err := micro.UnmarshalFromProtobuf(mess.Message.(*proto.PBMicroBlockMessage).MicroBlockBytes)
	if err != nil {
		return fsm, nil, err
	}
	return fsm.MicroBlock(mess.ID, micro)
}

func GetBlockIdsAction(services services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	for _, sig := range mess.Message.(*proto.GetBlockIdsMessage).Blocks {
		block, err := services.State.Header(sig)
		if err != nil {
			continue
		}
		sendBlockIds(services, block, mess.ID)
		break
	}
	return fsm, nil, nil
}

func BlockIdsAction(_ services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	return fsm.BlockIDs(mess.ID, mess.Message.(*proto.BlockIdsMessage).Blocks)
}

// TODO broadcast
func TransactionAction(s services.Services, mess peer.ProtoMessage, fsm state_fsm.FSM) (state_fsm.FSM, state_fsm.Async, error) {
	tbts := mess.Message.(*proto.TransactionMessage).Transaction
	t, err := proto.BytesToTransaction(tbts, s.Scheme)
	if err != nil {
		return fsm, nil, err
	}
	return fsm.Transaction(mess.ID, t)
}

func CreateActions() map[reflect.Type]Action {
	return map[reflect.Type]Action{
		reflect.TypeOf(&proto.ScoreMessage{}):             ScoreAction,
		reflect.TypeOf(&proto.GetPeersMessage{}):          GetPeersAction,
		reflect.TypeOf(&proto.BlockMessage{}):             BlockAction,
		reflect.TypeOf(&proto.GetBlockMessage{}):          GetBlockAction,
		reflect.TypeOf(&proto.SignaturesMessage{}):        SignaturesAction,
		reflect.TypeOf(&proto.GetSignaturesMessage{}):     GetSignaturesAction,
		reflect.TypeOf(&proto.MicroBlockInvMessage{}):     MicroBlockInvAction,
		reflect.TypeOf(&proto.MicroBlockRequestMessage{}): MicroBlockRequestAction,
		reflect.TypeOf(&proto.MicroBlockMessage{}):        MicroBlockAction,
		reflect.TypeOf(&proto.PBBlockMessage{}):           PBBlockAction,
		reflect.TypeOf(&proto.PBMicroBlockMessage{}):      PBMicroBlockAction,
		reflect.TypeOf(&proto.GetBlockIdsMessage{}):       GetBlockIdsAction,
		reflect.TypeOf(&proto.BlockIdsMessage{}):          BlockIdsAction,

		reflect.TypeOf(&proto.TransactionMessage{}): TransactionAction,
	}
}
