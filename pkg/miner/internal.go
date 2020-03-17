package miner

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func MineBlock(version proto.BlockVersion, nxt proto.NxtConsensus, pair proto.KeyPair, validatedFeatured Features, t proto.Timestamp, parent crypto.Signature, reward int64, scheme proto.Scheme) (*proto.Block, error) {
	b, err := proto.CreateBlock(proto.Transactions(nil), t, parent, pair.Public, nxt, version, FeaturesToInt16(validatedFeatured), reward, scheme)
	if err != nil {
		return nil, err
	}
	err = b.Sign(scheme, pair.Secret)
	if err != nil {
		return nil, err
	}
	return b, nil
}
