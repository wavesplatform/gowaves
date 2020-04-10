package miner

import (
	"context"

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
	utx         types.UtxPool
	state       state.State
	peer        peer_manager.PeerManager
	constraints Constraints
	services    services.Services
	features    Features
	// reward vote 600000000
	reward int64
}

func NewMicroblockMiner(services services.Services, features Features, reward int64) *MicroblockMiner {
	return &MicroblockMiner{
		utx:         services.UtxPool,
		state:       services.State,
		peer:        services.Peers,
		constraints: DefaultConstraints(),
		services:    services,
		features:    features,
		reward:      reward,
	}
}

func (a *MicroblockMiner) MineKeyBlock(ctx context.Context, t proto.Timestamp, k proto.KeyPair, parent proto.BlockID, baseTarget types.BaseTarget, gs []byte, vrf []byte) (*proto.Block, proto.MiningLimits, error) {
	nxt := proto.NxtConsensus{
		BaseTarget:   baseTarget,
		GenSignature: gs,
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
		b, err := MineBlock(v, nxt, k, validatedFeatured, t, parent, a.reward, a.services.Scheme)
		if err != nil {
			return nil, err
		}
		return b, nil
	})
	if err != nil {
		return nil, proto.MiningLimits{}, err
	}
	b := bi.(*proto.Block)
	zap.S().Debugf("Miner: generated new block id: %s, time: %d, block: %+v", b.BlockID().String(), t, b)

	rest := proto.MiningLimits{
		MaxScriptRunsInBlock:        a.constraints.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: a.constraints.MaxScriptsComplexityInBlock,
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
		return proto.ProtoBlockVersion, nil
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

func Run(ctx context.Context, a types.Miner, s *scheduler.SchedulerImpl, internalCh chan messages.InternalMessage) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-s.Mine():
			block, limits, err := a.MineKeyBlock(ctx, v.Timestamp, v.KeyPair, v.Parent, v.BaseTarget, v.GenSignature, v.VRF)
			if err != nil {
				zap.S().Error(err)
				continue
			}
			internalCh <- messages.NewMinedBlockInternalMessage(block, limits, v.KeyPair, v.VRF)
		}
	}
}
