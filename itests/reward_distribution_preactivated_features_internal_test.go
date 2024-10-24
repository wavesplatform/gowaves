//go:build !smoke

package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reward"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	node815 = "XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES"
	node816 = "XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES"
	node817 = "Single dao address should get 2 WAVES when full block reward >= 6 WAVES"
	node818 = "Single DAO or XTN buyback address should get max((R - 2)/2, 0) WAVES when full block " +
		"reward < 6 WAVES, mainer gets all reward If reward R <= 2 WAVES"
	node820 = "Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined"
	node821 = "Miner should get full block reward after CappedReward feature (20) activation " +
		"if BlockRewardDistribution feature (19) is not activated"
	node825 = "XTN buyback reward should be cancelled when CeaseXtnBuyback activated " +
		"after f19 activation height + xtnBuybackRewardPeriod (full reward > 6 WAVES)"
	node826 = "XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 " +
		"activation height + xtnBuybackRewardPeriod (full reward < 6 WAVES)"
	node829 = "Miner should get full block reward if daoAddress and xtnBuybackAddress are not " +
		"defined after f19 activation height + xtnBuybackRewardPeriod"
	node830 = "Block reward distribution should not change after CeaseXtnBuyback activation " +
		"if CappedReward not activated"
)

// NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20).
type RewardDistributionIncreaseDaoXtnPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionIncreaseDaoXtnPreactivatedSuite) Test_NODE815() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node815, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite,
			addresses, testdata.GetRewardIncreaseDaoXtnTestData)
	})
}

func TestRewardDistributionIncreaseDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoXtnPreactivatedSuite))
}

type RewardDistributionUnchangedDaoXtnPreactivatedSuite struct {
	f.RewardUnchangedDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionUnchangedDaoXtnPreactivatedSuite) Test_NODE815_2() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node815, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardUnchangedDaoXtnTestData)
	})
}

func TestRewardDistributionUnchangedDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedDaoXtnPreactivatedSuite))
}

// NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20).
type RewardDistributionDecreaseDaoXtnPreactivatedSuite struct {
	f.RewardDecreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionDecreaseDaoXtnPreactivatedSuite) Test_NODE816() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node816, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseDaoXtnTestData)
	})
}

func TestRewardDistributionDecreaseDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoXtnPreactivatedSuite))
}

// NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES after
// CappedReward feature (20) activation.
type RewardDistributionIncreaseDaoPreactivatedSuite struct {
	f.RewardIncreaseDaoPreactivatedSuite
}

func (suite *RewardDistributionIncreaseDaoPreactivatedSuite) Test_NODE817() {
	addresses := testdata.GetAddressesMinersDao(&suite.BaseSuite)
	suite.Run(node817, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseDaoTestData)
	})
}

func TestRewardDistributionIncreaseDaoPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoPreactivatedSuite))
}

type RewardDistributionUnchangedXtnPreactivatedSuite struct {
	f.RewardUnchangedXtnPreactivatedSuite
}

func (suite *RewardDistributionUnchangedXtnPreactivatedSuite) Test_NODE817_2() {
	addresses := testdata.GetAddressesMinersXtn(&suite.BaseSuite)
	suite.Run(node817, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardUnchangedXtnTestData)
	})
}

func TestRewardDistributionUnchangedXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedXtnPreactivatedSuite))
}

// NODE-818. Single XTN buyback or DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES
// after CappedReward feature (20) activation.
type RewardDistributionDecreaseXtnPreactivatedSuite struct {
	f.RewardDecreaseXtnPreactivatedSuite
}

func (suite *RewardDistributionDecreaseXtnPreactivatedSuite) Test_NODE818() {
	addresses := testdata.GetAddressesMinersXtn(&suite.BaseSuite)
	suite.Run(node818, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseXtnTestData)
	})
}

func TestRewardDistributionDecreaseXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseXtnPreactivatedSuite))
}

type RewardDistributionDecreaseDaoPreactivatedSuite struct {
	f.RewardDecreaseDaoPreactivatedSuite
}

func (suite *RewardDistributionDecreaseDaoPreactivatedSuite) Test_NODE818_2() {
	addresses := testdata.GetAddressesMinersDao(&suite.BaseSuite)
	suite.Run(node818, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseDaoTestData)
	})
}

func TestRewardDistributionDecreaseDaoPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoPreactivatedSuite))
}

type RewardDistribution2WUnchangedDaoXtnPreactivatedSuite struct {
	f.Reward2WUnchangedDaoXtnPreactivatedSuite
}

func (suite *RewardDistribution2WUnchangedDaoXtnPreactivatedSuite) Test_NODE818_3() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node818, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward2WUnchangedDaoXtnTestData)
	})
}

func TestRewardDistribution2WUnchangedDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistribution2WUnchangedDaoXtnPreactivatedSuite))
}

// NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined
// after CappedReward feature (20) activation.
type RewardDistributionIncreasePreactivatedSuite struct {
	f.RewardIncreasePreactivatedSuite
}

func (suite *RewardDistributionIncreasePreactivatedSuite) Test_NODE820() {
	addresses := testdata.GetAddressesMiners(&suite.BaseSuite)
	suite.Run(node820, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardMinersTestData)
	})
}

func TestRewardDistributionIncreasePreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreasePreactivatedSuite))
}

// NODE-821. Miner should get full block reward after CappedReward feature (20) activation
// if BlockRewardDistribution feature (19) is not activated.
type RewardDistributionDaoXtnPreactivatedWithout19Suite struct {
	f.RewardDaoXtnPreactivatedWithout19Suite
}

func (suite *RewardDistributionDaoXtnPreactivatedWithout19Suite) Test_NODE821() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node821, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDaoXtnWithout19TestData)
	})
}

func TestRewardDistributionDaoXtnPreactivatedWithout19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDaoXtnPreactivatedWithout19Suite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward >= 6 WAVES)".
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type RewardDistributionIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardDistributionIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE825() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node825, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards, settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseDaoXtnCeaseXTNBuybackBeforePeriodTestData)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseDaoXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardDistributionIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward > 6 WAVES)".
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type RewardDistributionIncreaseXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardIncreaseXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardDistributionIncreaseXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE825_2() {
	addresses := testdata.GetAddressesMinersXtn(&suite.BaseSuite)
	suite.Run(node825, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards, settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseXtnCeaseXTNBuybackBeforePeriodTestData)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardDistributionIncreaseXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward = 6 WAVES)".
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE825_3() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node825, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards, settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardUnchangedDaoXtnCeaseXTNBuybackBeforePeriodTestData)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardUnchangedDaoXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod
// blocks starting from BlockRewardDistribution activation height (full reward < 6 WAVES)".
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE826() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node826, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards, settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseDaoXtnCeaseXTNBuybackBeforePeriodTestData)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseDaoXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod
// blocks starting from BlockRewardDistribution activation height (full reward < 6 WAVES)".
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE826_2() {
	addresses := testdata.GetAddressesMinersXtn(&suite.BaseSuite)
	suite.Run(node826, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards, settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseXtnCeaseXTNBuybackBeforePeriodTestData)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod
// blocks starting from BlockRewardDistribution activation height (full reward = 2 WAVES)".
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE826_3() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node826, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards, settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward2WUnchangedDaoXtnCeaseXTNBuybackTestData)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward2WUnchangedDaoXtnCeaseXTNBuybackTestData)
	})
}

func TestReward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE - 829. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod
// blocks starting from BlockRewardDistribution activation height (full reward changes from 5 W to 7 W)".
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite struct {
	f.Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite
}

func (suite *Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite) Test_NODE829() {
	addresses := testdata.GetAddressesMiners(&suite.BaseSuite)
	suite.Run(node829, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards, settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward5W2MinersIncreaseCeaseXTNBuybackTestData)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward5W2MinersIncreaseCeaseXTNBuybackTestData)
	})
}

func TestReward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite))
}

// NODE-830. Block reward distribution should not change after CeaseXtnBuyback activation if CappedReward not activated.
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type RewardDaoXtnPreactivatedWithout20Suite struct {
	f.RewardDaoXtnPreactivatedWith21Suite
}

func (suite *RewardDaoXtnPreactivatedWithout20Suite) Test_NODE830() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node830, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDaoXtnBeforePeriodWithout20TestData)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDaoXtnAfterPeriodWithout20TestData)
	})
}

func TestRewardDaoXtnPreactivatedWithout20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDaoXtnPreactivatedWithout20Suite))
}

// NODE-830. Block reward distribution should not change after CeaseXtnBuyback activation if CappedReward not activated.
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)".
type RewardDaoXtnPreactivatedWithout19And20Suite struct {
	f.RewardDaoXtnPreactivatedWithout19And20Suite
}

func (suite *RewardDaoXtnPreactivatedWithout19And20Suite) Test_NODE830_2() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node830, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.XTNBuyBackCessation)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRewardDaoXtnWithout19And20TestData)
		ceaseXtnBuybackHeight := utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses, testdata.GetRewardDaoXtnWithout19And20TestData)
	})
}

func TestRewardDaoXtnPreactivatedWithout19And20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDaoXtnPreactivatedWithout19And20Suite))
}
