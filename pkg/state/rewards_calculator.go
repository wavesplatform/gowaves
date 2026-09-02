package state

import (
	"errors"
	"fmt"

	"github.com/ccoveille/go-safecast/v2"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state/internal"
)

const (
	boostedRewardMultiplier = 10
	defaultRewardMultiplier = 1

	// maxRewardPayoutsCount is the number of the block reward receivers: the miner, the DAO address and
	// the XTN buy-back address.
	maxRewardPayoutsCount = 3
)

// Block reward distribution parameters of feature 20 "CappedRewards".
const (
	fullRewardInit        = 6 * proto.PriceConstant
	maxAddressReward      = 2 * proto.PriceConstant
	guaranteedMinerReward = 2 * proto.PriceConstant
)

// Block reward distribution parameters of feature 26 "AdjustedBlockRewardDistribution".
const (
	adjustedFullReward            = 20 * proto.PriceConstant
	adjustedDAOReward             = 10 * proto.PriceConstant
	adjustedXTNBuybackReward      = 2 * proto.PriceConstant
	adjustedGuaranteedMinerReward = adjustedFullReward - adjustedDAOReward - adjustedXTNBuybackReward

	// The DAO address receives 5/6 and the XTN buy-back address receives 1/6 of the block reward left
	// after the guaranteed miner reward.
	adjustedDAORemainderDividend        = 5
	adjustedXTNBuybackRemainderDividend = 1
	adjustedRemainderDivider            = 6
)

// fraction is a part of a value. Its apply method reproduces the rounding of the Scala node's
// BlockDiffer.Fraction: the value is divided first and only then multiplied.
type fraction struct {
	dividend uint64
	divider  uint64
}

func (f fraction) apply(value uint64) uint64 { return value / f.divider * f.dividend }

// currentBlockRewardPartDivider is the divider of the part of the block reward every reward address
// receives after the activation of feature 19 and before the activation of feature 20.
const currentBlockRewardPartDivider = 3

// currentBlockRewardPart is the part of the block reward every reward address receives after the
// activation of feature 19 "BlockRewardDistribution" and before the activation of feature 20.
//
//nolint:gochecknoglobals // no writes
var currentBlockRewardPart = fraction{dividend: 1, divider: currentBlockRewardPartDivider}

// rewardDistribution describes how the block reward is split between the miner, the DAO address and the
// XTN buy-back address once feature 20 "CappedRewards" is activated.
type rewardDistribution struct {
	// fullReward is the block reward at which both the DAO and the XTN buy-back addresses receive their
	// maximum shares.
	fullReward uint64
	// guaranteedMinerReward is the part of the block reward the miner receives regardless of the votes.
	guaranteedMinerReward uint64
	// maxDAOReward is the maximum share of the DAO address.
	maxDAOReward uint64
	// maxXTNBuybackReward is the maximum share of the XTN buy-back address.
	maxXTNBuybackReward uint64
	// daoRemainderPart is the part of a below-fullReward block reward left after the guaranteed miner
	// reward that goes to the DAO address.
	daoRemainderPart fraction
	// xtnBuybackRemainderPart is the same for the XTN buy-back address.
	xtnBuybackRemainderPart fraction
}

//nolint:gochecknoglobals // no writes
var (
	// defaultDistribution is 2 (DAO) / 2 (miner) / 2 (XTN buy-back) at the initial block reward of
	// 6 WAVES.
	defaultDistribution = rewardDistribution{
		fullReward:              fullRewardInit,
		guaranteedMinerReward:   guaranteedMinerReward,
		maxDAOReward:            maxAddressReward,
		maxXTNBuybackReward:     maxAddressReward,
		daoRemainderPart:        fraction{dividend: 1, divider: 2},
		xtnBuybackRemainderPart: fraction{dividend: 1, divider: 2},
	}
	// adjustedDistribution is 10 (DAO) / 8 (miner) / 2 (XTN buy-back) at the block reward of 20 WAVES,
	// used after the activation of feature 26 "AdjustedBlockRewardDistribution". 20 WAVES is the amount
	// the block reward is reset to at the activation height, it stays votable afterward.
	adjustedDistribution = rewardDistribution{
		fullReward:            adjustedFullReward,
		guaranteedMinerReward: adjustedGuaranteedMinerReward,
		maxDAOReward:          adjustedDAOReward,
		maxXTNBuybackReward:   adjustedXTNBuybackReward,
		daoRemainderPart: fraction{
			dividend: adjustedDAORemainderDividend, divider: adjustedRemainderDivider,
		},
		xtnBuybackRemainderPart: fraction{
			dividend: adjustedXTNBuybackRemainderDividend, divider: adjustedRemainderDivider,
		},
	}
)

// shares splits the given block reward according to the distribution. Shares of the addresses that are
// not taken into account are given to the miner.
func (d rewardDistribution) shares(reward uint64, daoTaken, xtnBuybackTaken bool) rewardShares {
	switch {
	case reward < d.guaranteedMinerReward: // give the whole reward to the miner
		return rewardShares{miner: reward}
	case reward < d.fullReward: // give the miner its guaranteed reward and share the remainder
		remainder := reward - d.guaranteedMinerReward
		return newRewardShares(reward,
			d.daoRemainderPart.apply(remainder), d.xtnBuybackRemainderPart.apply(remainder),
			daoTaken, xtnBuybackTaken,
		)
	default: // the reward is at or above the full reward, the addresses receive their maximum shares
		return newRewardShares(reward, d.maxDAOReward, d.maxXTNBuybackReward, daoTaken, xtnBuybackTaken)
	}
}

// rewardShares is the block reward split between the miner, the DAO address and the XTN buy-back
// address.
type rewardShares struct {
	miner      uint64
	dao        uint64
	xtnBuyback uint64
}

// newRewardShares creates the shares of the reward giving the miner everything that is left after the
// addresses. Shares of the addresses that are not taken into account are zeroed.
func newRewardShares(reward, dao, xtnBuyback uint64, daoTaken, xtnBuybackTaken bool) rewardShares {
	if !daoTaken {
		dao = 0
	}
	if !xtnBuybackTaken {
		xtnBuyback = 0
	}
	return rewardShares{miner: reward - dao - xtnBuyback, dao: dao, xtnBuyback: xtnBuyback}
}

func (s rewardShares) multiply(by uint64) rewardShares {
	return rewardShares{miner: s.miner * by, dao: s.dao * by, xtnBuyback: s.xtnBuyback * by}
}

type rewardCalculator struct {
	settings *settings.BlockchainSettings
	features featuresStateForRewardsCalculator
}

type featuresStateForRewardsCalculator interface {
	newestIsActivatedAtHeight(featureID int16, height uint64) bool
	newestActivationHeight(featureID int16) (uint64, error)
}

func newRewardsCalculator(
	settings *settings.BlockchainSettings,
	features featuresStateForRewardsCalculator,
) *rewardCalculator {
	return &rewardCalculator{settings: settings, features: features}
}

func (c *rewardCalculator) calculateRewards(
	generator proto.WavesAddress, height proto.Height, reward uint64,
) (proto.Rewards, error) {
	multiplier, err := rewardMultiplier(c.settings, c.features, height)
	if err != nil {
		return nil, err
	}
	dao := c.settings.DAOAddress
	var xtnBuyback *proto.WavesAddress
	if !c.isXTNBuybackCeased(height) {
		xtnBuyback = c.settings.XTNBuybackAddress
	}
	shares := c.shares(height, reward, dao != nil, xtnBuyback != nil).multiply(multiplier)
	payouts := make([]proto.Reward, 0, maxRewardPayoutsCount)
	if shares.miner > 0 {
		payouts = append(payouts, proto.NewReward(generator, shares.miner))
	}
	if shares.dao > 0 && dao != nil {
		payouts = append(payouts, proto.NewReward(*dao, shares.dao))
	}
	if shares.xtnBuyback > 0 && xtnBuyback != nil {
		payouts = append(payouts, proto.NewReward(*xtnBuyback, shares.xtnBuyback))
	}
	return payouts, nil
}

func (c *rewardCalculator) applyToDiff(
	diff txDiff, generator proto.WavesAddress, height proto.Height, reward uint64,
) error {
	rewards, err := c.calculateRewards(generator, height, reward)
	if err != nil {
		return err
	}
	for _, r := range rewards {
		amount, cErr := safecast.Convert[int64](r.Amount())
		if cErr != nil {
			return fmt.Errorf("failed to apply the reward of address %q: %w", r.Address().String(), cErr)
		}
		key := wavesBalanceKey{r.Address().ID()}
		change := balanceDiff{balance: internal.NewIntChange(amount)}
		if abdErr := diff.appendBalanceDiff(key.bytes(), change); abdErr != nil {
			return abdErr
		}
	}
	return nil
}

// shares splits the block reward between the miner, the DAO address and the XTN buy-back address at the given height.
func (c *rewardCalculator) shares(height proto.Height, reward uint64, daoTaken, xtnBuybackTaken bool) rewardShares {
	// Give the full reward to the miner if feature 19 is not activated at the PROVIDED height.
	if !c.activatedAt(settings.BlockRewardDistribution, height) {
		return rewardShares{miner: reward}
	}
	// Before feature 20 every address receives a fixed part of the reward.
	if !c.activatedAt(settings.CappedRewards, height) {
		part := currentBlockRewardPart.apply(reward)
		return newRewardShares(reward, part, part, daoTaken, xtnBuybackTaken)
	}
	distribution := defaultDistribution
	if c.activatedAt(settings.AdjustedBlockRewardDistribution, height) {
		distribution = adjustedDistribution
	}
	return distribution.shares(reward, daoTaken, xtnBuybackTaken)
}

// xtnBuybackCeased reports whether the XTN buy-back is ceased at the given height.
func (c *rewardCalculator) isXTNBuybackCeased(height proto.Height) bool {
	if !c.activatedAt(settings.XTNBuyBackCessation, height) {
		// If feature 21 is not activated we don't have to check the minBuyBackPeriod, so we can return false.
		return false
	}
	// If feature 21 is activated we have to check that the required number of blocks passed since the activation
	// of feature 19. To do so we subtract minBuyBackPeriod from the block height and check that feature 19 was
	// activated at the resulting height.
	// If feature 19 was activated at or before the start of the period it means that we can cease XTN buy-back.
	if height <= c.settings.MinXTNBuyBackPeriod {
		return false
	}
	return c.activatedAt(settings.BlockRewardDistribution, height-c.settings.MinXTNBuyBackPeriod)
}

func (c *rewardCalculator) activatedAt(feature settings.Feature, height proto.Height) bool {
	return c.features.newestIsActivatedAtHeight(int16(feature), height)
}

func rewardMultiplier(
	s *settings.BlockchainSettings, f featuresStateForRewardsCalculator, h proto.Height,
) (uint64, error) {
	// Feature 26 "Adjusted Block Reward Distribution" supersedes feature 23: the block reward it votes
	// on is already the full amount issued per block, so no multiplication is applied anymore.
	if f.newestIsActivatedAtHeight(int16(settings.AdjustedBlockRewardDistribution), h) {
		return defaultRewardMultiplier, nil
	}
	// Feature 23 "Boost Block Reward" is working only for `BlockRewardBoostPeriod` count of blocks. We have to check
	// that feature already activated and not expired yet. In this case the multiplication can be applied, so we return
	// the value of `boostedRewardMultiplier`.
	ah, err := f.newestActivationHeight(int16(settings.BoostBlockReward))
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) { // feature 23 is not approved or activated.
			return defaultRewardMultiplier, nil
		}
		return 0, fmt.Errorf("failed to get activation height for feature 23: %w", err)
	}
	if h >= ah && h < ah+s.BlockRewardBoostPeriod {
		return boostedRewardMultiplier, nil
	}
	return defaultRewardMultiplier, nil
}
