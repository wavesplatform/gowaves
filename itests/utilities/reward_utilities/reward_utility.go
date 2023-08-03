package reward_utilities

import (
	"fmt"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

func GetBlockRewardDistribution[T any](suite *f.BaseSuite, testdata testdata.RewardDistributionTestData[T]) utl.RewardDiffBalancesInWaves {
	//init data
	//get init balance in waves of miners accounts
	initBalanceMiner1Go, initBalanceMiner1Scala := utl.GetAvailableBalanceInWaves(suite, testdata.Miner1Account.Address)
	initBalanceMiner2Go, initBalanceMiner2Scala := utl.GetAvailableBalanceInWaves(suite, testdata.Miner2Account.Address)
	fmt.Println("Init Balance miner1 Go: ", initBalanceMiner1Go, "Init Balance miner1 Scala: ", initBalanceMiner1Scala, "current height: ", utl.GetHeight(suite))
	fmt.Println("Init Balance miner2 Go: ", initBalanceMiner2Go, "Init Balance miner2 Scala: ", initBalanceMiner2Scala, "current height: ", utl.GetHeight(suite))
	//we will be summing up balances of both miners accounts
	initSumBalanceMinersGo := initBalanceMiner1Go + initBalanceMiner2Go
	initSumBalanceMinersScala := initBalanceMiner1Scala + initBalanceMiner2Scala
	fmt.Println("Init miners sum balance Go: ", initSumBalanceMinersGo)
	fmt.Println("Init miners sum balance Scala: ", initSumBalanceMinersScala)
	//get init balances of dao and xtn buy-back accounts
	initBalanceDaoGo, initBalanceDaoScala := utl.GetAvailableBalanceInWaves(suite, testdata.DaoAccount.Address)
	fmt.Println("Init Balance dao Go: ", initBalanceDaoGo, "Init Balance dao Scala: ", initBalanceDaoScala, "current height: ", utl.GetHeight(suite))
	initBalanceXtnGo, initBalanceXtnScala := utl.GetAvailableBalanceInWaves(suite, testdata.XtnBuyBackAccount.Address)
	fmt.Println("Init Balance xtn Go: ", initBalanceXtnGo, "Init Balance xtn Scala: ", initBalanceXtnScala, "current height: ", utl.GetHeight(suite))

	//wait for 1 block
	utl.WaitForNewHeight(suite)
	//get current balances of miners
	currentBalanceMiner1Go, currentBalanceMiner1Scala := utl.GetAvailableBalanceInWaves(suite, testdata.Miner1Account.Address)
	currentBalanceMiner2Go, currentBalanceMiner2Scala := utl.GetAvailableBalanceInWaves(suite, testdata.Miner2Account.Address)
	fmt.Println("Current Balance miner1 Go: ", currentBalanceMiner1Go, "Current Balance miner1 Scala: ", currentBalanceMiner1Scala, "current height: ", utl.GetHeight(suite))
	fmt.Println("Current Balance miner2 Go: ", currentBalanceMiner2Go, "Current Balance miner2 Scala: ", currentBalanceMiner2Scala, "current height: ", utl.GetHeight(suite))
	currentSumBalanceMinersGo := currentBalanceMiner1Go + currentBalanceMiner2Go
	currentSumBalanceMinersScala := currentBalanceMiner1Scala + currentBalanceMiner2Scala
	fmt.Println("Current miners sum balance Go: ", currentSumBalanceMinersGo)
	fmt.Println("Current miners sum balance Scala: ", currentSumBalanceMinersScala)
	//get current dao and xtn buy-back balance
	currentBalanceDaoGo, currentBalanceDaoScala := utl.GetAvailableBalanceInWaves(suite, testdata.DaoAccount.Address)
	fmt.Println("Current Balance dao Go: ", currentBalanceDaoGo, "Current Balance dao Scala: ", currentBalanceDaoScala, "current height: ", utl.GetHeight(suite))
	currentBalanceXtnGo, currentBalanceXtnScala := utl.GetAvailableBalanceInWaves(suite, testdata.XtnBuyBackAccount.Address)
	fmt.Println("Current Balance xtn Go: ", currentBalanceXtnGo, "Current Balance xtn Scala: ", currentBalanceXtnScala, "current height: ", utl.GetHeight(suite))

	//diff miners balance
	diffMinersSumBalancesGo := currentSumBalanceMinersGo - initSumBalanceMinersGo
	diffMinersSumBalancesScala := currentSumBalanceMinersScala - initSumBalanceMinersScala
	fmt.Println("Current diff sum miners Go: ", diffMinersSumBalancesGo, "Current diff sum miners Scala: ", diffMinersSumBalancesScala, "current height: ", utl.GetHeight(suite))
	//diff dao
	diffDaoGo := currentBalanceDaoGo - initBalanceDaoGo
	diffDaoScala := currentBalanceDaoScala - initBalanceDaoScala
	fmt.Println("Current diff dao Go: ", diffDaoGo, "Current diff dao Scala: ", diffDaoScala, "current height: ", utl.GetHeight(suite))

	//diff xtn
	diffXtnGo := currentBalanceXtnGo - initBalanceXtnGo
	diffXtnScala := currentBalanceXtnScala - initBalanceXtnScala
	fmt.Println("Current diff xtn Go: ", diffXtnGo, "Current diff xtn Scala: ", diffXtnScala, "current height: ", utl.GetHeight(suite))
	return utl.NewRewardDiffBalances(diffMinersSumBalancesGo, diffMinersSumBalancesScala, diffDaoGo, diffDaoScala, diffXtnGo, diffXtnScala)
}
