package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

// NODE - 858. Rollback (/debug/rollback) on height before BlockRewardDistribution feature activation should be correct.
type RewardDistributionAPIRollbackBeforeF19Suite struct {
	f.RewardDaoXtnSupported19Suite
}

func (suite *RewardDistributionAPIRollbackBeforeF19Suite) Test_NODE858() {
	name := "NODE-858. Rollback on height before BlockRewardDistribution feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRewardDistributionAfterF14Before19TestData)
		getActivationOfFeatures(&suite.BaseSuite, 19)
		activationH19 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF19TestData)
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH19-3), true)
		getActivationOfFeatures(&suite.BaseSuite, 19)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF19TestData)
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
	name := "NODE-859. Rollback on height after BlockRewardDistribution feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		activationH19 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF19TestData)
		utl.WaitForHeight(&suite.BaseSuite, uint64(activationH19+4))
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH19+1), true)
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF19TestData)
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
	name := " NODE - 860. Rollback on height before CappedReward feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRewardDistributionAfterF14Before19TestData)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		activationH20 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 20, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF20TestData)
		utl.GetRollbackToHeight(&suite.BaseSuite,
			uint64(activationH20)-utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)+1, true)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF20TestData)
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
	name := "NODE - 861. Rollback on height after CappedReward feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		activationH20 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 20, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF20TestData)
		utl.WaitForHeight(&suite.BaseSuite, uint64(activationH20+4))
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH20+1), true)
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF20TestData)
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
	name := "NODE - 862. Rollback on height before CeaseXTNBuyback feature activation should be correct"
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF21TestData)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
			19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		activationH21 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 21, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackBeforeF21TestData)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF21TestData)
		utl.GetRollbackToHeight(&suite.BaseSuite,
			uint64(activationH21)-utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)-1, true)
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20, 21)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRollbackAfterF21TestData)
	})
}

func TestRewardDistributionAPIRollbackBeforeF21Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackBeforeF21Suite))
}
