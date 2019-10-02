package node

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

func sendSignatures(block *proto.Block, stateManager state.State, p peer.Peer) {
	height, err := stateManager.BlockIDToHeight(block.BlockSignature)
	if err != nil {
		zap.S().Error(err)
		return
	}

	var out []crypto.Signature
	out = append(out, block.BlockSignature)

	for i := 1; i < 101; i++ {
		b, err := stateManager.BlockByHeight(height + uint64(i))
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

func LastSignatures(state state.State) (*Signatures, error) {
	var signatures []crypto.Signature

	height, err := state.Height()
	if err != nil {
		zap.S().Error(err)
		return nil, err
	}

	for i := 0; i < 100 && height > 0; i++ {
		sig, err := state.HeightToBlockID(height)
		if err != nil {
			zap.S().Error(err)
			return nil, err
		}
		signatures = append(signatures, sig)
		height -= 1
	}
	return NewSignatures(signatures...), nil
}
