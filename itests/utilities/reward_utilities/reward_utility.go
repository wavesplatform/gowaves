package reward_utilities

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

func GetBlockRewardDistribution[T any](suite *f.BaseSuite, testdata testdata.RewardDistributionTestData[T]) utl.RewardDiffBalancesInWaves {
	var initBalanceMiner1Go, initBalanceMiner1Scala, initBalanceMiner2Go, initBalanceMiner2Scala,
		initBalanceDaoGo, initBalanceDaoScala, initBalanceXtnGo, initBalanceXtnScala int64
	var currentBalanceMiner1Go, currentBalanceMiner1Scala, currentBalanceMiner2Go, currentBalanceMiner2Scala,
		currentBalanceDaoGo, currentBalanceDaoScala, currentBalanceXtnGo, currentBalanceXtnScala int64
	//get init balance in waves of miners accounts
	if testdata.Miner1Account != nil {
		initBalanceMiner1Go, initBalanceMiner1Scala = utl.GetAvailableBalanceInWaves(suite, testdata.Miner1Account.Address)
	}
	if testdata.Miner2Account != nil {
		initBalanceMiner2Go, initBalanceMiner2Scala = utl.GetAvailableBalanceInWaves(suite, testdata.Miner2Account.Address)
	}
	//we will be summing up balances of both miners accounts
	initSumBalanceMinersGo := initBalanceMiner1Go + initBalanceMiner2Go
	initSumBalanceMinersScala := initBalanceMiner1Scala + initBalanceMiner2Scala
	suite.T().Logf("Go: Init miners sum balance: %d, Scala: Init miners sum balance Scala: %d, current height: %d",
		initSumBalanceMinersGo, initSumBalanceMinersScala, utl.GetHeight(suite))
	//get init balances of dao and xtn buy-back accounts
	if testdata.DaoAccount != nil {
		initBalanceDaoGo, initBalanceDaoScala = utl.GetAvailableBalanceInWaves(suite, testdata.DaoAccount.Address)
	}
	suite.T().Logf("Go: Init DAO balance: %d, Scala: Init DAO balance: %d, current height: %d",
		initBalanceDaoGo, initBalanceDaoScala, utl.GetHeight(suite))
	if testdata.XtnBuyBackAccount != nil {
		initBalanceXtnGo, initBalanceXtnScala = utl.GetAvailableBalanceInWaves(suite, testdata.XtnBuyBackAccount.Address)
	}
	suite.T().Logf("Go: Init XTN balance: %d, Scala: Init XTN balance: %d, current height: %d",
		initBalanceXtnGo, initBalanceXtnScala, utl.GetHeight(suite))
	//wait for 1 block
	utl.WaitForNewHeight(suite)
	//get current balances of miners
	if testdata.Miner1Account != nil {
		currentBalanceMiner1Go, currentBalanceMiner1Scala = utl.GetAvailableBalanceInWaves(suite, testdata.Miner1Account.Address)
	}
	if testdata.Miner2Account != nil {
		currentBalanceMiner2Go, currentBalanceMiner2Scala = utl.GetAvailableBalanceInWaves(suite, testdata.Miner2Account.Address)
	}
	currentSumBalanceMinersGo := currentBalanceMiner1Go + currentBalanceMiner2Go
	currentSumBalanceMinersScala := currentBalanceMiner1Scala + currentBalanceMiner2Scala
	suite.T().Logf("Go: Current miners sum balance: %d, Scala: Current miners sum balance: %d, current height: %d",
		currentSumBalanceMinersGo, currentSumBalanceMinersScala, utl.GetHeight(suite))
	//get current dao and xtn buy-back balance
	if testdata.DaoAccount != nil {
		currentBalanceDaoGo, currentBalanceDaoScala = utl.GetAvailableBalanceInWaves(suite, testdata.DaoAccount.Address)
	}
	suite.T().Logf("Go: Current Balance DAO: %d, Scala: Current Balance DAO: %d, current height: %d",
		currentBalanceDaoGo, currentBalanceDaoScala, utl.GetHeight(suite))
	if testdata.XtnBuyBackAccount != nil {
		currentBalanceXtnGo, currentBalanceXtnScala = utl.GetAvailableBalanceInWaves(suite, testdata.XtnBuyBackAccount.Address)
	}
	suite.T().Logf("Go: Current Balance XTN: %d, Scala: Current Balance XTN: %d, current height: %d",
		currentBalanceXtnGo, currentBalanceXtnScala, utl.GetHeight(suite))
	//diff miners balance
	diffMinersSumBalancesGo := currentSumBalanceMinersGo - initSumBalanceMinersGo
	diffMinersSumBalancesScala := currentSumBalanceMinersScala - initSumBalanceMinersScala
	suite.T().Logf("Go: Diff sum miners: %d, Scala: Diff sum miners: %d, on height: %d",
		diffMinersSumBalancesGo, diffMinersSumBalancesScala, utl.GetHeight(suite))
	//diff dao
	diffDaoGo := currentBalanceDaoGo - initBalanceDaoGo
	diffDaoScala := currentBalanceDaoScala - initBalanceDaoScala
	suite.T().Logf("Go: Diff DAO: %d, Scala: Diff DAO: %d, on height: %d",
		diffDaoGo, diffDaoScala, utl.GetHeight(suite))
	//diff xtn
	diffXtnGo := currentBalanceXtnGo - initBalanceXtnGo
	diffXtnScala := currentBalanceXtnScala - initBalanceXtnScala
	suite.T().Logf("Go: Diff XTN: %d, Scala: Diff XTN: %d, on height: %d",
		diffXtnGo, diffXtnScala, utl.GetHeight(suite))
	return utl.NewRewardDiffBalances(diffMinersSumBalancesGo, diffMinersSumBalancesScala, diffDaoGo, diffDaoScala, diffXtnGo, diffXtnScala)
}
