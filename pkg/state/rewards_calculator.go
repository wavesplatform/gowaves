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

type doMinerReward func(reward uint64) error
type doAddressReward func(addr proto.WavesAddress, reward uint64) error

func (c *rewardCalculator) performCalculation(
	appendMinerReward doMinerReward,
	appendAddressReward doAddressReward,
	height proto.Height,
	reward uint64,
) error {
	feature19Activated, err := c.features.newestIsActivated(int16(settings.BlockRewardDistribution))
	if err != nil {
		return err
	}
	if !feature19Activated { // pay rewards only to the miner if feature19 is not activated on the CURRENT height
		return appendMinerReward(reward)
	}

	feature20Activated, err := c.features.newestIsActivated(int16(settings.CappedRewards))
	if err != nil {
		return err
	}
	if feature20Activated { // feature 19 activated and feature20 for the CURRENT height
		feature19ActivatedAtHeight := c.features.newestIsActivatedAtHeight(int16(settings.BlockRewardDistribution), height)
		if !feature19ActivatedAtHeight { // append rewards only to the miner if feature19 is not activated for the given height
			return appendMinerReward(reward)
		}
	}

	rewardAddresses := c.settings.RewardAddresses
	feature21ActivatedAtHeight := c.features.newestIsActivatedAtHeight(int16(settings.XTNBuyBackCessation), height)
	if feature21ActivatedAtHeight {
		rewardAddresses = c.handleFeature21(height, rewardAddresses)
	}

	addressReward := reward / uint64(len(rewardAddresses)+1) // reward / (len(rewardAddresses) + minerAddr)
	feature20ActivatedAtHeight := c.features.newestIsActivatedAtHeight(int16(settings.CappedRewards), height)
	if feature20ActivatedAtHeight {
		addressReward = c.handleFeature20(reward)
		if addressReward == 0 {
			return appendMinerReward(reward) // give full reward to the miner
		}
	}

	return c.appendRewards(appendMinerReward, appendAddressReward, rewardAddresses, reward, addressReward)
}

func (c *rewardCalculator) appendRewards(
	appendMinerReward doMinerReward,
	appendAddressReward doAddressReward,
	rewardAddresses []proto.WavesAddress,
	reward uint64,
	addressReward uint64,
) error {
	minerReward := reward
	for _, a := range rewardAddresses {
		if err := appendAddressReward(a, addressReward); err != nil {
			return err
		}
		minerReward -= addressReward
	}
	return appendMinerReward(minerReward)
}

func (c *rewardCalculator) handleFeature21(height proto.Height, rewardAddresses []proto.WavesAddress) []proto.WavesAddress {
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
	return rewardAddresses
}

func (c *rewardCalculator) handleFeature20(reward uint64) uint64 {
	const (
		sixWaves                 = 6 * proto.PriceConstant
		twoWaves                 = 2 * proto.PriceConstant
		additionalAddressesCount = 2
	)
	switch {
	case reward < twoWaves: // give all reward to the miner if reward value is lower than 2 WAVES
		return 0
	case reward < sixWaves: // give miner guaranteed reward with 2 WAVES
		// We always calculates XTN/DAO reward for 2 addresses even there is only one present
		return (reward - twoWaves) / additionalAddressesCount
	default: // reward is greater or equal six waves, then give fixed 2 WAVES rewards to addresses
		return twoWaves
	}
}
