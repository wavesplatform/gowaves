package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
)

// NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20)
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation
type RewardDistributionIncreaseDaoXtnSupportedSuite struct {
	f.RewardIncreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionIncreaseDaoXtnSupportedSuite) Test_NODE815() {
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardIncreaseDaoXtnSupportedTestData, 19, 20)
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
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardUnchangedDaoXtnSupportedTestData, 19, 20)
	})
}

func TestRewardDistributionUnchangedDaoXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedDaoXtnSupportedSuite))
}

// NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20)
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation
type RewardDistributionDecreaseDaoXtnSupportedSuite struct {
	f.RewardDecreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionDecreaseDaoXtnSupportedSuite) Test_NODE816() {
	name := "NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDecreaseDaoXtnSupportedTestData, 19, 20)
	})

}

func TestRewardDistributionDecreaseDaoXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoXtnSupportedSuite))
}

// NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES after
// CappedReward feature (20) activation
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation
type RewardDistributionIncreaseDaoSupportedSuite struct {
	f.RewardIncreaseDaoSupportedSuite
}

func (suite *RewardDistributionIncreaseDaoSupportedSuite) Test_NODE817() {
	name := "NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardIncreaseDaoSupportedTestData, 19, 20)
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
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardUnchangedXtnSupportedTestData, 19, 20)
	})
}

func TestRewardDistributionUnchangedXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedXtnSupportedSuite))
}

// NODE-818. Single XTN buyback or DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES
// after CappedReward feature (20) activation
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation
type RewardDistributionDecreaseXtnSupportedSuite struct {
	f.RewardDecreaseXtnSupportedSuite
}

func (suite *RewardDistributionDecreaseXtnSupportedSuite) Test_NODE818() {
	name := "NODE-818. Single XTN buyback address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDecreaseXtnSupportedTestData, 19, 20)
	})
}

func TestRewardDistributionDecreaseXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseXtnSupportedSuite))
}

// NODE-818_2. Single XTN Buyback or DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES
// after CappedReward feature (20) activation
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation
type RewardDistributionDecreaseDaoSupportedSuite struct {
	f.RewardDecreaseDaoSupportedSuite
}

func (suite *RewardDistributionDecreaseDaoSupportedSuite) Test_NODE818_2() {
	name := "NODE-818. Single DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDecreaseDaoSupportedTestData, 19, 20)
	})
}

func TestRewardDistributionDecreaseDaoSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoSupportedSuite))
}

// NODE-818_3 If reward R <= 2 Waves mainer gets all reward
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation
type RewardDistribution2WUnchangedDaoXtnSupportedSuite struct {
	f.Reward2WUnchangedDaoXtnSupportedSuite
}

func (suite *RewardDistribution2WUnchangedDaoXtnSupportedSuite) Test_NODE818_3() {
	name := "NODE-818. mainer gets all reward If reward R <= 2 WAVES; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetReward2WUnchangedDaoXtnSupportedTestData, 19, 20)
	})
}

func TestRewardDistribution2WUnchangedDaoXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistribution2WUnchangedDaoXtnSupportedSuite))
}

// NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined
// after CappedReward feature (20) activation
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation
type RewardDistributionIncreaseSupportedSuite struct {
	f.RewardIncreaseSupportedSuite
}

func (suite *RewardDistributionIncreaseSupportedSuite) Test_NODE820() {
	name := "NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 19 and 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardSupportedTestData, 19, 20)
	})
}

func TestRewardDistributionIncreaseSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseSupportedSuite))
}

// NODE-821. Miner should get full block reward after CappedReward feature (20) activation
// if BlockRewardDistribution feature (19) is not activated
// NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation
type RewardDistributionDaoXtnSupportedWithout19Suite struct {
	f.RewardDaoXtnSupportedWithout19Suite
}

func (suite *RewardDistributionDaoXtnSupportedWithout19Suite) Test_NODE821() {
	name := "NODE-821. Miner should get full block reward after CappedReward feature (20) activation " +
		"if BlockRewardDistribution feature (19) is not activated; " +
		"NODE-822. termAfterCappedRewardFeature option should be used instead of term option after CappedReward activation"
	suite.Run(name, func() {
		//check rewards and terms before activation 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDistributionAfter14Before19)
		//check rewards and terms after activation 20
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDaoXtnSupportedWithout19TestData, 20)
	})
}

func TestRewardDistributionDaoXtnSupportedWithout19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDaoXtnSupportedWithout19Suite))
}
