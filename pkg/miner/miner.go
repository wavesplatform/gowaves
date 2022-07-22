package miner

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type MicroblockMiner struct {
	utx                             types.UtxPool
	state                           state.State
	peer                            peer_manager.PeerManager
	constraints                     Constraints
	services                        services.Services
	features                        Features
	reward                          int64
	maxTransactionTimeForwardOffset proto.Timestamp
}

func NewMicroblockMiner(services services.Services, features Features, reward int64, maxTransactionTimeForwardOffset proto.Timestamp) *MicroblockMiner {
	return &MicroblockMiner{
		utx:                             services.UtxPool,
		state:                           services.State,
		peer:                            services.Peers,
		constraints:                     DefaultConstraints(),
		services:                        services,
		features:                        features,
		reward:                          reward,
		maxTransactionTimeForwardOffset: maxTransactionTimeForwardOffset,
	}
}

func (a *MicroblockMiner) MineKeyBlock(ctx context.Context, t proto.Timestamp, k proto.KeyPair, parent proto.BlockID, baseTarget types.BaseTarget, gs []byte, vrf []byte) (*proto.Block, proto.MiningLimits, error) {
	nxt := proto.NxtConsensus{
		BaseTarget:   baseTarget,
		GenSignature: gs,
	}
	// This prevents miner of creating excessive quantity of blocks.
	// If time is too outdate, that just create 1 block with minimal sensible time.
	ts := t
	now := a.services.Time.Now()
	if t < proto.NewTimestampFromTime(now)-a.maxTransactionTimeForwardOffset {
		ts = proto.NewTimestampFromTime(now) - a.maxTransactionTimeForwardOffset
	}

	bi, err := a.state.MapR(func(info state.StateInfo) (interface{}, error) {
		v, err := blockVersion(info)
		if err != nil {
			return nil, err
		}
		validatedFeatured, err := ValidateFeatures(info, a.features)
		if err != nil {
			return nil, err
		}
		b, err := MineBlock(v, nxt, k, validatedFeatured, ts, parent, a.reward, a.services.Scheme)
		if err != nil {
			return nil, err
		}
		return b, nil
	})
	if err != nil {
		return nil, proto.MiningLimits{}, err
	}
	b := bi.(*proto.Block)

	activated, err := a.state.IsActivated(int16(settings.RideV5))
	if err != nil {
		return nil, proto.MiningLimits{}, errors.Wrapf(err, "failed to check if feature %d is activated", settings.RideV5)
	}

	rest := proto.MiningLimits{
		MaxScriptRunsInBlock:        a.constraints.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: a.constraints.MaxScriptsComplexityInBlock.GetMaxScriptsComplexityInBlock(activated),
		ClassicAmountOfTxsInBlock:   a.constraints.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           a.constraints.MaxTxsSizeInBytes - 4,
	}

	return b, rest, nil
}

func blockVersion(state state.StateInfo) (proto.BlockVersion, error) {
	blockV5Activated, err := state.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return 0, err
	}
	if blockV5Activated {
		return proto.ProtobufBlockVersion, nil
	}
	height, err := state.Height()
	if err != nil {
		return 0, err
	}
	blockRewardActivated, err := state.IsActiveAtHeight(int16(settings.BlockReward), height)
	if err != nil {
		return 0, err
	}
	if blockRewardActivated {
		return proto.RewardBlockVersion, nil
	}
	return proto.NgBlockVersion, nil
}

type Mine interface {
	Mine() chan scheduler.Emit
}

func Run(ctx context.Context, a types.Miner, s Mine, internalCh chan<- messages.InternalMessage) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-s.Mine():
			block, limits, err := a.MineKeyBlock(ctx, v.Timestamp, v.KeyPair, v.Parent, v.BaseTarget, v.GenSignature, v.VRF)
			if err != nil {
				zap.S().Errorf("Failed to mine key block: %v", err)
				continue
			}
			internalCh <- messages.NewMinedBlockInternalMessage(block, limits, v.KeyPair, v.VRF)
		}
	}
}
