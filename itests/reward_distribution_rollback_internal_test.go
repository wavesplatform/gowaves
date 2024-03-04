package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/utilities/reward"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// NODE - 858. Rollback (/debug/rollback) on height before BlockRewardDistribution feature activation should be correct.
type RewardDistributionAPIRollbackBeforeF19Suite struct {
	f.RewardDaoXtnSupported19Suite
}

func (suite *RewardDistributionAPIRollbackBeforeF19Suite) Test_NODE858() {
	const node858 = "Rollback on height before BlockRewardDistribution feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node858, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution)
		activationH19 := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution, utl.GetHeight(&suite.BaseSuite))
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF19TestData)
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH19-3), true)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF19TestData)
	})
}

func TestRewardDistributionAPIRollbackBeforeF19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackBeforeF19Suite))
}

// NODE - 859. Rollback (/debug/rollback) on height after BlockRewardDistribution feature activation should be correct.
type RewardDistributionAPIRollbackAfterF19Suite struct {
	f.RewardDaoXtnPreactivatedWithout20Suite
}

func (suite *RewardDistributionAPIRollbackAfterF19Suite) Test_NODE859() {
	const node859 = "Rollback on height after BlockRewardDistribution feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node859, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution)
		activationH19 := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution, utl.GetHeight(&suite.BaseSuite))
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF19TestData)
		utl.WaitForHeight(&suite.BaseSuite, uint64(activationH19+3))
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH19+1), true)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF19TestData)
	})
}

func TestRewardDistributionAPIRollbackAfterF19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackAfterF19Suite))
}

// NODE - 860. Rollback (/debug/rollback) on height before CappedReward feature activation should be correct.
type RewardDistributionAPIRollbackBeforeF20Suite struct {
	f.RewardIncreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionAPIRollbackBeforeF20Suite) Test_NODE860() {
	const node860 = "Rollback on height before CappedReward feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node860, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution, settings.CappedRewards)
		activationH20 := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.CappedRewards, utl.GetHeight(&suite.BaseSuite))
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF20TestData)
		utl.GetRollbackToHeight(&suite.BaseSuite,
			uint64(activationH20)-utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)+1, true)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution, settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF20TestData)
	})
}

func TestRewardDistributionAPIRollbackBeforeF20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackBeforeF20Suite))
}

// NODE - 861. Rollback (/debug/rollback) on height after CappedReward feature activation should be correct.
type RewardDistributionAPIRollbackAfterF20Suite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionAPIRollbackAfterF20Suite) Test_NODE861() {
	const node861 = "Rollback on height after CappedReward feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node861, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite,
			settings.BlockReward, settings.BlockRewardDistribution, settings.CappedRewards)
		activationH20 := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.CappedRewards, utl.GetHeight(&suite.BaseSuite))
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF20TestData)
		utl.WaitForHeight(&suite.BaseSuite, uint64(activationH20+3))
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH20+1), true)
		utl.GetActivationOfFeatures(&suite.BaseSuite,
			settings.BlockReward, settings.BlockRewardDistribution, settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF20TestData)
	})
}

func TestRewardDistributionAPIRollbackAfterF20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackAfterF20Suite))
}

// NODE - 862. Rollback on height before CeaseXTNBuyback feature activation should be correct.
type RewardDistributionAPIRollbackBeforeF21Suite struct {
	f.RewardDistributionRollbackBefore21Suite
}

func (suite *RewardDistributionAPIRollbackBeforeF21Suite) Test_NODE862() {
	const node862 = "Rollback on height before CeaseXTNBuyback feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node862, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite,
			settings.BlockReward, settings.BlockRewardDistribution, settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF21TestData)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		activationH21 := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.XTNBuyBackCessation, utl.GetHeight(&suite.BaseSuite))
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF21TestData)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF21TestData)
		utl.GetRollbackToHeight(&suite.BaseSuite,
			uint64(activationH21)-utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)-1, true)
		utl.GetActivationOfFeatures(&suite.BaseSuite,
			settings.BlockReward, settings.BlockRewardDistribution, settings.CappedRewards,
			settings.XTNBuyBackCessation)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF21TestData)
	})
}

func TestRewardDistributionAPIRollbackBeforeF21Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackBeforeF21Suite))
}
