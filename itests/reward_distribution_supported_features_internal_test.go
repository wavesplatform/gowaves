package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

// NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20).
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistributionIncreaseDaoXtnSupportedSuite struct {
	f.RewardIncreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionIncreaseDaoXtnSupportedSuite) Test_NODE815() {
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option" +
		" after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardIncreaseDaoXtnSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
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
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option" +
		" after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardUnchangedDaoXtnSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
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
	name := "NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option " +
		"after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardDecreaseDaoXtnSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
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
	name := "NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option" +
		" after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardIncreaseDaoSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
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
	name := "NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term" +
		" option after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardUnchangedXtnSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
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
	name := "NODE-818. Single XTN buyback address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option" +
		" after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardDecreaseXtnSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionDecreaseXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseXtnSupportedSuite))
}

// NODE-818_2. Single XTN Buyback or DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES
// after CappedReward feature (20) activation.
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistributionDecreaseDaoSupportedSuite struct {
	f.RewardDecreaseDaoSupportedSuite
}

func (suite *RewardDistributionDecreaseDaoSupportedSuite) Test_NODE818_2() {
	name := "NODE-818. Single DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option" +
		" after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardDecreaseDaoSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionDecreaseDaoSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoSupportedSuite))
}

// NODE-818_3 If reward R <= 2 Waves mainer gets all reward.
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation.
type RewardDistribution2WUnchangedDaoXtnSupportedSuite struct {
	f.Reward2WUnchangedDaoXtnSupportedSuite
}

func (suite *RewardDistribution2WUnchangedDaoXtnSupportedSuite) Test_NODE818_3() {
	name := "NODE-818. mainer gets all reward If reward R <= 2 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option " +
		"after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetReward2WUnchangedDaoXtnSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
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
	name := "NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option" +
		" after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
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
	name := "NODE-821. Miner should get full block reward after CappedReward feature (20) activation " +
		"if BlockRewardDistribution feature (19) is not activated; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	td := testdata.GetRewardDaoXtnSupportedWithout19TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td)
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
	name := "NODE-825, NODE-841. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19" +
		" activation height + xtnBuybackRewardPeriod (full reward >= 6 WAVES)"
	td := testdata.GetRewardIncreaseDaoXtnCeaseXTNBuybackSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
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
	name := "NODE-825, NODE-841. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 " +
		"activation height + xtnBuybackRewardPeriod (full reward >= 6 WAVES)"
	td := testdata.GetRewardIncreaseXtnCeaseXTNBuybackSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
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
	name := "NODE-825, NODE-841. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 " +
		"activation height + xtnBuybackRewardPeriod (full reward = 6 WAVES)"
	td := testdata.GetRewardUnchangedDaoXtnCeaseXTNBuybackSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
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
	name := "NODE - 826, NODE-841. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 " +
		"activation height + xtnBuybackRewardPeriod (full reward < 6 WAVES)"
	td := testdata.GetRewardDecreaseDaoXtnCeaseXTNBuybackSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
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
	name := "NODE - 826, NODE-841. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 " +
		"activation height + xtnBuybackRewardPeriod (full reward < 6 WAVES)"
	td := testdata.GetRewardDecreaseXtnCeaseXTNBuybackSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
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
	name := "NODE - 826, NODE-841. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 " +
		"activation height + xtnBuybackRewardPeriod (full reward = 2 WAVES)"
	td := testdata.GetReward2WUnchangedDaoXtnCeaseXTNBuybackSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
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

func (suite *Reward5W2MinersIncreaseCeaseXTNBuybackSupportedSuite) Test_NODE826_3() {
	name := "NODE - 829, NODE-841. Miner should get full block reward if daoAddress and xtnBuybackAddress are not" +
		" defined after f19 activation height + xtnBuybackRewardPeriod"
	td := testdata.GetReward5W2MinersIncreaseCeaseXTNBuybackSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
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
	name := "NODE - 830, NODE-841. Block reward distribution should not change after CeaseXtnBuyback activation " +
		"if CappedReward not activated"
	td := testdata.GetRewardDaoXtnSupportedWithout20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19,
			utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
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
	name := "NODE - 830, NODE-841. Block reward distribution should not change after CeaseXtnBuyback " +
		"activation if CappedReward not activated"
	td := testdata.GetRewardDaoXtnSupportedWithout19And20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		ceaseXtnBuybackHeight := utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})
}

func TestRewardDaoXtnSupportedWithout19And20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDaoXtnSupportedWithout19And20Suite))
}
