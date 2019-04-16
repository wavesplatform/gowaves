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
