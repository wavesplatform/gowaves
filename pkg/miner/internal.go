package miner

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func mineKeyBlock(
	state state.State,
	version proto.BlockVersion,
	nxt proto.NxtConsensus,
	pair proto.KeyPair,
	validatedFeatured Features,
	ts proto.Timestamp,
	parent proto.BlockID,
	reward int64,
	scheme proto.Scheme,
) (*proto.Block, error) {
	b, err := proto.CreateBlock(proto.Transactions(nil), ts, parent, pair.Public,
		nxt, version, FeaturesToInt16(validatedFeatured), reward, scheme, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new key block")
	}
	blockchainHeight, err := state.Height()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get blockchain height")
	}
	// Key block it's a new block for the blockchain, so height should be increased by 1.
	newBlockHeight := blockchainHeight + 1
	lightNodeNewBlockActivated, err := state.IsActiveLightNodeNewBlocksFields(newBlockHeight)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to check if light node new block fields activated at height %d", newBlockHeight,
		)
	}
	if lightNodeNewBlockActivated {
		sh, errSH := state.CreateNextSnapshotHash(b)
		if errSH != nil {
			return nil, errors.Wrapf(errSH,
				"failed to create initial snapshot hash for new key block (reference to %s)", b.Parent.String(),
			)
		}
		b.StateHash = &sh
	}
	err = b.Sign(scheme, pair.Secret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign new key block")
	}
	err = b.GenerateBlockID(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate block ID for new key block")
	}
	return b, nil
}
