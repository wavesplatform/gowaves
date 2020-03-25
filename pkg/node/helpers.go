package node

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

const (
	maxShiftFromNow = 600000 // 10 minutes.
)

func MaybeEnableExtendedApi(state state.State, time types.Time) error {
	lastBlock := state.TopBlock()
	return maybeEnableExtendedApi(state, lastBlock, proto.NewTimestampFromTime(time.Now()))
}

type startProvidingExtendedApi interface {
	StartProvidingExtendedApi() error
}

func maybeEnableExtendedApi(state startProvidingExtendedApi, lastBlock *proto.Block, now proto.Timestamp) error {
	provideExtended := false
	if lastBlock.Timestamp > now {
		provideExtended = true
	} else if now-lastBlock.Timestamp < maxShiftFromNow {
		provideExtended = true
	}
	if provideExtended {
		if err := state.StartProvidingExtendedApi(); err != nil {
			return err
		}
	}
	return nil
}

func supportsNewBlockId(p sendMessage) bool {
	v := p.Handshake().Version
	if v.Major > 1 || (v.Major == 1 && v.Minor >= 2) {
		// Version >= 1.2.0.
		return true
	}
	return false
}

func signatures(ids []proto.BlockID) []crypto.Signature {
	var res []crypto.Signature
	for _, id := range ids {
		if id.IsSignature() {
			res = append(res, id.Signature())
		}
	}
	return res
}

func onlySignatures(ids []proto.BlockID) bool {
	for _, id := range ids {
		if !id.IsSignature() {
			return false
		}
	}
	return true
}

func sendGetBlockIds(ids []proto.BlockID, p sendMessage) {
	if supportsNewBlockId(p) {
		p.SendMessage(&proto.GetBlockIdsMessage{
			Blocks: ids,
		})
	} else if onlySignatures(ids) {
		p.SendMessage(&proto.GetSignaturesMessage{
			Blocks: signatures(ids),
		})
	}
}

func sendBlockIds(ids []proto.BlockID, p sendMessage) {
	if supportsNewBlockId(p) {
		p.SendMessage(&proto.BlockIdsMessage{
			Blocks: ids,
		})
	} else {
		supported := signatures(ids)
		p.SendMessage(&proto.SignaturesMessage{
			Signatures: supported,
		})
	}
}

func sendBlockIdsFromBlock(block *proto.Block, stateManager state.State, p sendMessage) {
	height, err := stateManager.BlockIDToHeight(block.BlockID())
	if err != nil {
		zap.S().Error(err)
		return
	}

	var out []proto.BlockID
	out = append(out, block.BlockID())

	for i := 1; i < 101; i++ {
		b, err := stateManager.BlockByHeight(height + uint64(i))
		if err != nil {
			break
		}
		out = append(out, b.BlockID())
	}

	// if we put smth except first block
	if len(out) > 1 {
		sendBlockIds(out, p)
	}
}

func LastBlockIds(state state.State) (*BlockIds, error) {
	var ids []proto.BlockID

	height, err := state.Height()
	if err != nil {
		zap.S().Error(err)
		return nil, err
	}

	for i := 0; i < 100 && height > 0; i++ {
		id, err := state.HeightToBlockID(height)
		if err != nil {
			zap.S().Error(err)
			return nil, err
		}
		ids = append(ids, id)
		height -= 1
	}
	return NewBlockIds(ids...), nil
}
