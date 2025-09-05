package miner

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type MicroblockMiner struct {
	utx         types.UtxPool
	state       state.State
	peer        peers.PeerManager
	constraints Constraints
	services    services.Services
	features    Features
	reward      int64
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

func (a *MicroblockMiner) MineKeyBlock(
	_ context.Context, t proto.Timestamp, k proto.KeyPair, parent proto.BlockID, baseTarget types.BaseTarget,
	gs []byte, _ []byte,
) (*proto.Block, proto.MiningLimits, error) {
	nxt := proto.NxtConsensus{
		BaseTarget:   baseTarget,
		GenSignature: gs,
	}
	var kb *proto.Block
	// Using MapUnsafe because mining process happens in parallel and with regular Map node can panic because of
	// CAS check in ThreadSafeReadWrapper of state.
	err := a.state.MapUnsafe(func(state state.NonThreadSafeState) error {
		v, err := blockVersion(state)
		if err != nil {
			return err
		}
		validatedFeatured, err := ValidateFeatures(state, a.features)
		if err != nil {
			return errors.Wrap(err, "failed to validate features")
		}
		b, err := mineKeyBlock(state, v, nxt, k, validatedFeatured, t, parent, a.reward, a.services.Scheme)
		if err != nil {
			return errors.Wrap(err, "failed mineKeyBlock")
		}
		kb = b
		return nil
	})
	if err != nil {
		return nil, proto.MiningLimits{}, errors.Wrap(err, "microblock miner failed to mine key block")
	}

	activated, err := a.state.IsActivated(int16(settings.RideV5))
	if err != nil {
		return nil, proto.MiningLimits{}, errors.Wrapf(err, "failed to check if feature %d is activated",
			settings.RideV5)
	}

	rest := proto.MiningLimits{
		MaxScriptRunsInBlock:        a.constraints.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: a.constraints.MaxScriptsComplexityInBlock.GetMaxScriptsComplexityInBlock(activated),
		ClassicAmountOfTxsInBlock:   a.constraints.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           a.constraints.MaxTxsSizeInBytes - 4,
	}

	return kb, rest, nil
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
			block, limits, err := a.MineKeyBlock(ctx, v.Timestamp, v.KeyPair, v.Parent, v.BaseTarget, v.GenSignature,
				v.VRF)
			if err != nil {
				slog.Error("Failed to mine key block", logging.Error(err))
				continue
			}
			internalCh <- messages.NewMinedBlockInternalMessage(block, limits, v.KeyPair, v.VRF)
		}
	}
}
