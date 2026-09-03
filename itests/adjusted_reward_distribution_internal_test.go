//go:build !smoke

package itests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reward"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// Preactivated features 14, 19, 20, 26, 2 miners, dao, xtn, initR = 2000000000.
// The block reward of 20 WAVES is distributed as 8 WAVES to the miner, 10 WAVES to the DAO address and
// 2 WAVES to the XTN buy-back address.
type AdjustedRewardDaoXtnPreactivatedTestSuite struct {
	f.AdjustedRewardDaoXtnPreactivatedSuite
}

func (s *AdjustedRewardDaoXtnPreactivatedTestSuite) Test_AdjustedRewardDistributionDaoXtn() {
	const name = "Adjusted block reward distribution with DAO and XTN buy-back addresses"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.AdjustedBlockRewardDistribution)
		assert.Equal(s.T(), uint64(testdata.AdjustedFullReward),
			utl.GetCurrentReward(&s.BaseSuite, utl.GetHeight(&s.BaseSuite)))
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoXtnTestData)
	})
}

func TestAdjustedRewardDaoXtnPreactivatedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdjustedRewardDaoXtnPreactivatedTestSuite))
}

// Preactivated features 14, 19, 20, 26, 2 miners, dao only, initR = 2000000000.
// The share of the missing XTN buy-back address goes to the miner.
type AdjustedRewardDaoPreactivatedTestSuite struct {
	f.AdjustedRewardDaoPreactivatedSuite
}

func (s *AdjustedRewardDaoPreactivatedTestSuite) Test_AdjustedRewardDistributionDao() {
	const name = "Adjusted block reward distribution with the DAO address only"
	addresses := testdata.GetAddressesMinersDao(&s.BaseSuite)
	s.Run(name, func() {
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.AdjustedBlockRewardDistribution)
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoTestData)
	})
}

func TestAdjustedRewardDaoPreactivatedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdjustedRewardDaoPreactivatedTestSuite))
}

// Preactivated features 14, 19, 20, 26, 2 miners, xtn only, initR = 2000000000.
// The share of the missing DAO address goes to the miner. This configuration is the one the removed
// positional reward address list could not express.
type AdjustedRewardXtnPreactivatedTestSuite struct {
	f.AdjustedRewardXtnPreactivatedSuite
}

func (s *AdjustedRewardXtnPreactivatedTestSuite) Test_AdjustedRewardDistributionXtn() {
	const name = "Adjusted block reward distribution with the XTN buy-back address only"
	addresses := testdata.GetAddressesMinersXtn(&s.BaseSuite)
	s.Run(name, func() {
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.AdjustedBlockRewardDistribution)
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionXtnTestData)
	})
}

func TestAdjustedRewardXtnPreactivatedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdjustedRewardXtnPreactivatedTestSuite))
}

// Preactivated features 14, 19, 20, 26, 2 miners, dao, xtn, initR = 1400000000.
// The block reward is below the full reward of feature 26, so the miner receives its guaranteed reward
// of 8 WAVES and the addresses share the remainder as 5/6 and 1/6.
type AdjustedRewardBelowFullRewardPreactivatedTestSuite struct {
	f.AdjustedRewardBelowFullRewardPreactivatedSuite
}

func (s *AdjustedRewardBelowFullRewardPreactivatedTestSuite) Test_AdjustedRewardDistributionBelowFullReward() {
	const name = "Adjusted block reward distribution below the full reward"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.AdjustedBlockRewardDistribution)
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoXtnTestData)
	})
}

func TestAdjustedRewardBelowFullRewardPreactivatedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdjustedRewardBelowFullRewardPreactivatedTestSuite))
}

// Preactivated features 14, 19, 20, 21, 26, 2 miners, dao, xtn, initR = 2000000000,
// min_xtn_buy_back_period = 5. After the cessation of XTN buy-back its share goes to the miner.
type AdjustedRewardCeaseXtnBuybackPreactivatedTestSuite struct {
	f.AdjustedRewardCeaseXtnBuybackPreactivatedSuite
}

func (s *AdjustedRewardCeaseXtnBuybackPreactivatedTestSuite) Test_AdjustedRewardDistributionCeaseXtnBuyback() {
	const name = "Adjusted block reward distribution with the cessation of XTN buy-back"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.AdjustedBlockRewardDistribution)
		// Before the end of the XTN buy-back period all three receivers get their shares.
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoXtnTestData)
		// The XTN buy-back is ceased once the period passes since the activation of feature 19.
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BlockRewardDistribution, utl.GetHeight(&s.BaseSuite)) +
			utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)
		utl.WaitForHeight(&s.BaseSuite, ceaseXtnBuybackHeight,
			config.WaitWithTimeoutInBlocks(utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)))
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoTestData)
	})
}

func TestAdjustedRewardCeaseXtnBuybackPreactivatedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdjustedRewardCeaseXtnBuybackPreactivatedTestSuite))
}

// Preactivated features 14, 19, 20, 22, 23, 26, 2 miners, dao, xtn, initR = 2000000000.
// Feature 26 supersedes feature 23, so the block reward must not be boosted.
type AdjustedRewardWithBoostPreactivatedTestSuite struct {
	f.AdjustedRewardWithBoostPreactivatedSuite
}

func (s *AdjustedRewardWithBoostPreactivatedTestSuite) Test_AdjustedRewardDistributionSupersedesBoost() {
	const name = "Adjusted block reward distribution supersedes the boost of the block reward"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.LightNode,
			settings.BoostBlockReward,
			settings.AdjustedBlockRewardDistribution)
		// The reward is not multiplied by the boost multiplier of feature 23.
		assert.Equal(s.T(), uint64(testdata.AdjustedFullReward),
			utl.GetCurrentReward(&s.BaseSuite, utl.GetHeight(&s.BaseSuite)))
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoXtnTestData)
	})
}

func TestAdjustedRewardWithBoostPreactivatedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdjustedRewardWithBoostPreactivatedTestSuite))
}

// Preactivated features 14, 19, 20 and supported feature 26, 2 miners, dao, xtn, initR = 600000000.
// The block reward must be reset to 20 WAVES at the activation height of feature 26 and the
// distribution must switch to the adjusted one.
type AdjustedRewardSupportedTestSuite struct {
	f.AdjustedRewardSupportedSuite
}

func (s *AdjustedRewardSupportedTestSuite) Test_AdjustedRewardDistributionResetsTheReward() {
	const name = "Activation of the adjusted block reward distribution resets the block reward"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards)
		// Before the activation of feature 26 the default distribution of 2 / 2 / 2 is in effect.
		reward.GetRewardDistributionAndChecks(&s.BaseSuite, addresses,
			testdata.GetRewardUnchangedDaoXtnTestData)

		utl.GetActivationOfFeatures(&s.BaseSuite, settings.AdjustedBlockRewardDistribution)
		activationHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.AdjustedBlockRewardDistribution, utl.GetHeight(&s.BaseSuite))
		// The reward of the block right before the activation height is the voted one.
		assert.Equal(s.T(), uint64(600000000), utl.GetCurrentReward(&s.BaseSuite, activationHeight-1))
		// Starting from the activation height the reward is the full reward of feature 26.
		assert.Equal(s.T(), uint64(testdata.AdjustedFullReward),
			utl.GetCurrentReward(&s.BaseSuite, activationHeight))
		// The reward stays votable after the activation, the miners of this configuration vote for
		// 6 WAVES, so the reward starts to decrease from the next term. The expected shares are derived
		// from the reward the blockchain currently has.
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoXtnTestData)

		// A rollback across the activation height must restore the distribution.
		utl.GetRollbackToHeight(&s.BaseSuite, activationHeight-1, true)
		utl.GetActivationOfFeatures(&s.BaseSuite, settings.AdjustedBlockRewardDistribution)
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoXtnTestData)
	})
}

func TestAdjustedRewardSupportedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdjustedRewardSupportedTestSuite))
}

// Preactivated features 14, 19, 20 and supported feature 26, 2 miners, dao, xtn, initR = 600000000,
// desiredR = 2100000000. The block reward must be reset to 20 WAVES at the activation height of
// feature 26 and then the votes of the miners must increase it to 21 WAVES. The reward is above the
// full reward of feature 26, so the surplus goes to the miner.
type AdjustedRewardSupportedIncreaseTestSuite struct {
	f.AdjustedRewardSupportedIncreaseSuite
}

func (s *AdjustedRewardSupportedIncreaseTestSuite) Test_AdjustedRewardDistributionIncreasesTheReward() {
	const name = "The block reward increases by the votes after the adjusted block reward distribution reset it"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.AdjustedBlockRewardDistribution)
		activationHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.AdjustedBlockRewardDistribution, utl.GetHeight(&s.BaseSuite))
		// The reward voted for before the activation height is below the full reward of feature 26.
		assert.Less(s.T(), utl.GetCurrentReward(&s.BaseSuite, activationHeight-1),
			uint64(testdata.AdjustedFullReward))
		// The activation resets the reward to the full reward of feature 26 regardless of the votes.
		assert.Equal(s.T(), uint64(testdata.AdjustedFullReward),
			utl.GetCurrentReward(&s.BaseSuite, activationHeight))

		// The reward stays votable, the miners of this configuration vote for 21 WAVES, so the reward
		// increases by one increment at the first term that starts after the activation height.
		term := utl.GetRewardTermAfter20Cfg(&s.BaseSuite)
		increasedRewardHeight := activationHeight + term
		utl.WaitForHeight(&s.BaseSuite, increasedRewardHeight, config.WaitWithTimeoutInBlocks(term+1))
		increasedReward := uint64(testdata.AdjustedFullReward) + utl.GetBlockRewardIncrementCfg(&s.BaseSuite)
		assert.Equal(s.T(), increasedReward, utl.GetCurrentReward(&s.BaseSuite, increasedRewardHeight))

		// The reward is above the full reward of feature 26, so the DAO and the XTN buy-back addresses
		// receive their maximum shares of 10 and 2 WAVES and the surplus goes to the miner.
		reward.GetRewardDistributionAndChecksWithoutTerm(&s.BaseSuite, addresses,
			testdata.GetRewardAdjustedDistributionDaoXtnTestData)
	})
}

func TestAdjustedRewardSupportedIncreaseTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdjustedRewardSupportedIncreaseTestSuite))
}
