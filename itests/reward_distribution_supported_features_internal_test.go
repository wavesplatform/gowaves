//go:build !smoke

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

// NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20).
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistributionIncreaseDaoXtnSupportedSuite struct {
	f.RewardIncreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionIncreaseDaoXtnSupportedSuite) Test_NODE815() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node815, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseDaoXtnTestData)
	})
}

func TestRewardDistributionIncreaseDaoXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoXtnSupportedSuite))
}

type RewardDistributionUnchangedDaoXtnSupportedSuite struct {
	f.RewardUnchangedDaoXtnSupportedSuite
}

func (suite *RewardDistributionUnchangedDaoXtnSupportedSuite) Test_NODE815_2() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node815, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardUnchangedDaoXtnTestData)
	})
}

func TestRewardDistributionUnchangedDaoXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedDaoXtnSupportedSuite))
}

// NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20).
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistributionDecreaseDaoXtnSupportedSuite struct {
	f.RewardDecreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionDecreaseDaoXtnSupportedSuite) Test_NODE816() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node816, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseDaoXtnTestData)
	})
}

func TestRewardDistributionDecreaseDaoXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoXtnSupportedSuite))
}

// NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES after
// CappedReward feature (20) activation.
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistributionIncreaseDaoSupportedSuite struct {
	f.RewardIncreaseDaoSupportedSuite
}

func (suite *RewardDistributionIncreaseDaoSupportedSuite) Test_NODE817() {
	addresses := testdata.GetAddressesMinersDao(&suite.BaseSuite)
	suite.Run(node817, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseDaoTestData)
	})
}

func TestRewardDistributionIncreaseDaoSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoSupportedSuite))
}

type RewardDistributionUnchangedXtnSupportedSuite struct {
	f.RewardUnchangedXtnSupportedSuite
}

func (suite *RewardDistributionUnchangedXtnSupportedSuite) Test_NODE817_2() {
	addresses := testdata.GetAddressesMinersXtn(&suite.BaseSuite)
	suite.Run(node817, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardUnchangedXtnTestData)
	})
}

func TestRewardDistributionUnchangedXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedXtnSupportedSuite))
}

// NODE-818. Single XTN buyback or DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES
// after CappedReward feature (20) activation.
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistributionDecreaseXtnSupportedSuite struct {
	f.RewardDecreaseXtnSupportedSuite
}

func (suite *RewardDistributionDecreaseXtnSupportedSuite) Test_NODE818() {
	addresses := testdata.GetAddressesMinersXtn(&suite.BaseSuite)
	suite.Run(node818, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseXtnTestData)
	})
}

func TestRewardDistributionDecreaseXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseXtnSupportedSuite))
}

type RewardDistributionDecreaseDaoSupportedSuite struct {
	f.RewardDecreaseDaoSupportedSuite
}

func (suite *RewardDistributionDecreaseDaoSupportedSuite) Test_NODE818_2() {
	addresses := testdata.GetAddressesMinersDao(&suite.BaseSuite)
	suite.Run(node818, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseDaoTestData)
	})
}

func TestRewardDistributionDecreaseDaoSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoSupportedSuite))
}

type RewardDistribution2WUnchangedDaoXtnSupportedSuite struct {
	f.Reward2WUnchangedDaoXtnSupportedSuite
}

func (suite *RewardDistribution2WUnchangedDaoXtnSupportedSuite) Test_NODE818_3() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node818, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward2WUnchangedDaoXtnTestData)
	})
}

func TestRewardDistribution2WUnchangedDaoXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistribution2WUnchangedDaoXtnSupportedSuite))
}

// NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined
// after CappedReward feature (20) activation.
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistributionIncreaseSupportedSuite struct {
	f.RewardIncreaseSupportedSuite
}

func (suite *RewardDistributionIncreaseSupportedSuite) Test_NODE820() {
	addresses := testdata.GetAddressesMiners(&suite.BaseSuite)
	suite.Run(node820, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardMinersTestData)
	})
}

func TestRewardDistributionIncreaseSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseSupportedSuite))
}

// NODE-821. Miner should get full block reward after CappedReward feature (20) activation
// if BlockRewardDistribution feature (19) is not activated.
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistributionDaoXtnSupportedWithout19Suite struct {
	f.RewardDaoXtnSupportedWithout19Suite
}

func (suite *RewardDistributionDaoXtnSupportedWithout19Suite) Test_NODE821() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node821, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDistributionAfterF14Before19TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDaoXtnWithout19TestData)
	})
}

func TestRewardDistributionDaoXtnSupportedWithout19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDaoXtnSupportedWithout19Suite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward > 6 WAVES)".
// "NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod).
type RewardIncreaseDaoXtnCeaseXTNBuybackSupportedSuite struct {
	f.RewardIncreaseDaoXtnCeaseXTNBuybackSupportedSuite
}

func (suite *RewardIncreaseDaoXtnCeaseXTNBuybackSupportedSuite) Test_NODE825() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node825, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseDaoXtnCeaseXTNBuybackBeforePeriodTestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseDaoXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardIncreaseDaoXtnCeaseXTNBuybackSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardIncreaseDaoXtnCeaseXTNBuybackSupportedSuite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward > 6 WAVES)".
// "NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod).
type RewardIncreaseXtnCeaseXTNBuybackSupportedSuite struct {
	f.RewardIncreaseXtnCeaseXTNBuybackSupportedSuite
}

func (suite *RewardIncreaseXtnCeaseXTNBuybackSupportedSuite) Test_NODE825_2() {
	addresses := testdata.GetAddressesMinersXtn(&suite.BaseSuite)
	suite.Run(node825, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward,
			settings.BlockRewardDistribution, settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseXtnCeaseXTNBuybackBeforePeriodTestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardIncreaseXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardIncreaseXtnCeaseXTNBuybackSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardIncreaseXtnCeaseXTNBuybackSupportedSuite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward = 6 WAVES)".
// "NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod).
type RewardUnchangedDaoXtnCeaseXTNBuybackSupportedSuite struct {
	f.RewardUnchangedDaoXtnCeaseXTNBuybackSupportedSuite
}

func (suite *RewardUnchangedDaoXtnCeaseXTNBuybackSupportedSuite) Test_NODE825_3() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node825, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardUnchangedDaoXtnCeaseXTNBuybackBeforePeriodTestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardUnchangedDaoXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardUnchangedDaoXtnCeaseXTNBuybackSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardUnchangedDaoXtnCeaseXTNBuybackSupportedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod
// blocks starting from BlockRewardDistribution activation height (full reward < 6 WAVES)".
//
//	"NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod).
type RewardDecreaseDaoXtnCeaseXTNBuybackSupportedSuite struct {
	f.RewardDecreaseDaoXtnCeaseXTNBuybackSupportedSuite
}

func (suite *RewardDecreaseDaoXtnCeaseXTNBuybackSupportedSuite) Test_NODE826() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node826, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseDaoXtnCeaseXTNBuybackBeforePeriodTestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseDaoXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardDecreaseDaoXtnCeaseXTNBuybackSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDecreaseDaoXtnCeaseXTNBuybackSupportedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod
// blocks starting from BlockRewardDistribution activation height (full reward < 6 WAVES)".
// "NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod).
type RewardDecreaseXtnCeaseXTNBuybackSupportedSuite struct {
	f.RewardDecreaseXtnCeaseXTNBuybackSupportedSuite
}

func (suite *RewardDecreaseXtnCeaseXTNBuybackSupportedSuite) Test_NODE826_2() {
	addresses := testdata.GetAddressesMinersXtn(&suite.BaseSuite)
	suite.Run(node826, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseXtnCeaseXTNBuybackBeforePeriodTestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDecreaseXtnCeaseXTNBuybackAfterPeriodTestData)
	})
}

func TestRewardDecreaseXtnCeaseXTNBuybackSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDecreaseXtnCeaseXTNBuybackSupportedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod
// blocks starting from BlockRewardDistribution activation height (full reward = 2 WAVES)".
// "NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod).
type Reward2WUnchangedDaoXtnCeaseXTNBuybackSupportedSuite struct {
	f.Reward2WUnchangedDaoXtnCeaseXTNBuybackSupportedSuite
}

func (suite *Reward2WUnchangedDaoXtnCeaseXTNBuybackSupportedSuite) Test_NODE826_3() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node826, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward2WUnchangedDaoXtnCeaseXTNBuybackTestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward2WUnchangedDaoXtnCeaseXTNBuybackTestData)
	})
}

func TestReward2WUnchangedDaoXtnCeaseXTNBuybackSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(Reward2WUnchangedDaoXtnCeaseXTNBuybackSupportedSuite))
}

// "NODE - 829. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod
// blocks starting from BlockRewardDistribution activation height (full reward changes from 5 W to 7 W)".
// "NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod)".
type Reward5W2MinersIncreaseCeaseXTNBuybackSupportedSuite struct {
	f.Reward5W2MinersIncreaseCeaseXTNBuybackSupportedSuite
}

func (suite *Reward5W2MinersIncreaseCeaseXTNBuybackSupportedSuite) Test_NODE829() {
	addresses := testdata.GetAddressesMiners(&suite.BaseSuite)
	suite.Run(node829, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward5W2MinersIncreaseCeaseXTNBuybackTestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetReward5W2MinersIncreaseCeaseXTNBuybackTestData)
	})
}

func TestReward5W2MinersIncreaseCeaseXTNBuybackSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(Reward5W2MinersIncreaseCeaseXTNBuybackSupportedSuite))
}

// NODE-830. Block reward distribution should not change after CeaseXtnBuyback activation if CappedReward not activated.
// "NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod)".
type RewardDaoXtnSupportedWithout20Suite struct {
	f.RewardDaoXtnSupportedWithout20Suite
}

func (suite *RewardDaoXtnSupportedWithout20Suite) Test_NODE830() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node830, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDaoXtnBeforePeriodWithout20TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDaoXtnAfterPeriodWithout20TestData)
	})
}

func TestRewardDaoXtnSupportedWithout20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDaoXtnSupportedWithout20Suite))
}

// NODE-830. Block reward distribution should not change after CeaseXtnBuyback activation if CappedReward not activated.
// "NODE-841. Reward Distribution changed after f21 activation (h21 >= h19 + xtnBuybackRewardPeriod)".
type RewardDaoXtnSupportedWithout19And20Suite struct {
	f.RewardDaoXtnSupportedWithout19And20Suite
}

func (suite *RewardDaoXtnSupportedWithout19And20Suite) Test_NODE830_2() {
	addresses := testdata.GetAddressesMinersDaoXtn(&suite.BaseSuite)
	suite.Run(node830, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDaoXtnWithout19And20TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.XTNBuyBackCessation)
		ceaseXtnBuybackHeight := utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		reward.GetRewardDistributionAndChecks(&suite.BaseSuite, addresses,
			testdata.GetRewardDaoXtnWithout19And20TestData)
	})
}

func TestRewardDaoXtnSupportedWithout19And20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDaoXtnSupportedWithout19And20Suite))
}
