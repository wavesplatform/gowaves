package reward_utilities

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

func GetBlockRewardDistribution[T any](suite *f.BaseSuite, testdata testdata.RewardDistributionTestData[T], height uint64) utl.RewardDiffBalancesInWaves {
	//init data
	//get init balance in waves of miners accounts
	initBalanceMiner1Go, initBalanceMiner1Scala := utl.GetAvailableBalanceInWaves(suite, testdata.Miner1Account.Address)
	initBalanceMiner2Go, initBalanceMiner2Scala := utl.GetAvailableBalanceInWaves(suite, testdata.Miner2Account.Address)
	//we will be summing up balances of both miners accounts
	initSumBalanceMinersGo := initBalanceMiner1Go + initBalanceMiner2Go
	initSumBalanceMinersScala := initBalanceMiner1Scala + initBalanceMiner2Scala
	//get init balances of dao and xtn buy-back accounts
	initBalanceDaoGo, initBalanceDaoScala := utl.GetAvailableBalanceInWaves(suite, testdata.DaoAccount.Address)
	initBalanceXtnGo, initBalanceXtnScala := utl.GetAvailableBalanceInWaves(suite, testdata.XtnBuyBackAccount.Address)
	//wait for 1 block
	utl.WaitForHeight(suite, height+1)
	//get current balances of miners
	currentBalanceMiner1Go, currentBalanceMiner1Scala := utl.GetAvailableBalanceInWaves(suite, testdata.Miner1Account.Address)
	currentBalanceMiner2Go, currentBalanceMiner2Scala := utl.GetAvailableBalanceInWaves(suite, testdata.Miner2Account.Address)
	currentSumBalanceMinersGo := currentBalanceMiner1Go + currentBalanceMiner2Go
	currentSumBalanceMinersScala := currentBalanceMiner1Scala + currentBalanceMiner2Scala
	//get current dao and xtn buy-back balance
	currentBalanceDaoGo, currentBalanceDaoScala := utl.GetAvailableBalanceInWaves(suite, testdata.DaoAccount.Address)
	currentBalanceXtnGo, currentBalanceXtnScala := utl.GetAvailableBalanceInWaves(suite, testdata.XtnBuyBackAccount.Address)

	//diff miners balance
	diffMinersSumBalancesGo := currentSumBalanceMinersGo - initSumBalanceMinersGo
	diffMinersSumBalancesScala := currentSumBalanceMinersScala - initSumBalanceMinersScala
	//diff dao
	diffDaoGo := currentBalanceDaoGo - initBalanceDaoGo
	diffDaoScala := currentBalanceDaoScala - initBalanceDaoScala
	//diff xtn
	diffXtnGo := currentBalanceXtnGo - initBalanceXtnGo
	diffXtnScala := currentBalanceXtnScala - initBalanceXtnScala

	return utl.NewRewardDiffBalances(diffMinersSumBalancesGo, diffMinersSumBalancesScala, diffDaoGo, diffDaoScala, diffXtnGo, diffXtnScala)
}
