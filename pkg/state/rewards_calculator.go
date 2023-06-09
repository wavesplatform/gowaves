package state

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type rewardCalculator struct {
	settings *settings.BlockchainSettings
	features featuresState
}

func newRewardsCalculator(settings *settings.BlockchainSettings, features featuresState) *rewardCalculator {
	return &rewardCalculator{settings: settings, features: features}
}

func (c *rewardCalculator) calculateRewards(generator proto.WavesAddress, height proto.Height, reward uint64) (proto.Rewards, error) {
	r := make(proto.Rewards, 0, len(c.settings.RewardAddresses)+1)
	if err := c.performCalculation(
		func(reward uint64) error {
			r = append(r, proto.NewReward(generator, reward))
			return nil
		},
		func(addr proto.WavesAddress, reward uint64) error {
			r = append(r, proto.NewReward(addr, reward))
			return nil
		},
		height, reward); err != nil {
		return nil, err
	}
	return r, nil
}

func (c *rewardCalculator) applyToDiff(diff txDiff, addr proto.AddressID, height proto.Height, reward uint64) error {
	return c.performCalculation(
		func(r uint64) error {
			key := wavesBalanceKey{addr}
			return diff.appendBalanceDiff(key.bytes(), balanceDiff{balance: int64(r)})
		},
		func(a proto.WavesAddress, r uint64) error {
			key := wavesBalanceKey{a.ID()}
			return diff.appendBalanceDiff(key.bytes(), balanceDiff{balance: int64(r)})
		},
		height, reward)
}

func (c *rewardCalculator) performCalculation(
	appendMinerReward func(reward uint64) error,
	appendAddressReward func(addr proto.WavesAddress, reward uint64) error,
	height proto.Height,
	reward uint64,
) error {
	minerReward := reward
	active19 := c.features.newestIsActivatedAtHeight(int16(settings.BlockRewardDistribution), height)
	if !active19 {
		return appendMinerReward(minerReward)
	}
	rewardAddresses := c.settings.RewardAddresses
	feature21Activated := c.features.newestIsActivatedAtHeight(int16(settings.XTNBuyBackCessation), height)
	if feature21Activated {
		// If feature 21 activated we have to check that required number of blocks passed since activation of feature 19.
		// To do so we subtract minBuyBackPeriod from the block height and check that feature 19 was activated at the
		// resulting height. If feature 19 was activated at or before the start of the period it means that we can cease
		// XTN buy-back.
		if minBuyBackPeriodStartHeight := int64(height) - int64(c.settings.MinXTNBuyBackPeriod); minBuyBackPeriodStartHeight > 0 {
			minBuyBackPeriodPassed := c.features.newestIsActivatedAtHeight(int16(settings.BlockRewardDistribution), uint64(minBuyBackPeriodStartHeight))
			if minBuyBackPeriodPassed {
				rewardAddresses = c.settings.RewardAddressesAfter21
			}
		}
	}
	numberOfAddresses := uint64(len(rewardAddresses) + 1) // len(rewardAddresses) + minerAddr
	for _, a := range rewardAddresses {
		addressReward := reward / numberOfAddresses
		if err := appendAddressReward(a, addressReward); err != nil {
			return err
		}
		minerReward -= addressReward
	}
	return appendMinerReward(minerReward)
}
