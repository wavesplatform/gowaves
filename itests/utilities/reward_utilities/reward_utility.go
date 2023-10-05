package reward_utilities

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

func getAddressesBalances[T any](suite *f.BaseSuite, testdata testdata.RewardDistributionTestData[T]) (utl.BalanceInWaves,
	utl.BalanceInWaves, utl.BalanceInWaves) {
	var balanceMiner1Go, balanceMiner1Scala, balanceMiner2Go, balanceMiner2Scala,
		balanceDaoGo, balanceDaoScala, balanceXtnGo, balanceXtnScala int64
	//get balance in waves of miners accounts
	if testdata.Miner1Account != nil {
		balanceMiner1Go, balanceMiner1Scala = utl.GetAvailableBalanceInWaves(suite, testdata.Miner1Account.Address)
	}
	if testdata.Miner2Account != nil {
		balanceMiner2Go, balanceMiner2Scala = utl.GetAvailableBalanceInWaves(suite, testdata.Miner2Account.Address)
	}
	//we will be summing up balances of both miners accounts
	sumBalanceMinersGo := balanceMiner1Go + balanceMiner2Go
	sumBalanceMinersScala := balanceMiner1Scala + balanceMiner2Scala
	suite.T().Logf("Go: Sum Miners balance: %d, Go current height:%d, Scala: Sum Miners balance: %d, Scala current height: %d",
		sumBalanceMinersGo, utl.GetHeightGo(suite), sumBalanceMinersScala, utl.GetHeightScala(suite))
	//get balances of dao and xtn buy-back accounts
	if testdata.DaoAccount != nil {
		balanceDaoGo, balanceDaoScala = utl.GetAvailableBalanceInWaves(suite, testdata.DaoAccount.Address)
	}
	suite.T().Logf("Go: DAO balance: %d, Go current height:%d, Scala: DAO balance: %d, Scala current height: %d",
		balanceDaoGo, utl.GetHeightGo(suite), balanceDaoScala, utl.GetHeightScala(suite))
	if testdata.XtnBuyBackAccount != nil {
		balanceXtnGo, balanceXtnScala = utl.GetAvailableBalanceInWaves(suite, testdata.XtnBuyBackAccount.Address)
	}
	suite.T().Logf("Go: XTN balance: %d, Go current height:%d, Scala: XTN balance: %d, Scala current height: %d",
		balanceXtnGo, utl.GetHeightGo(suite), balanceXtnScala, utl.GetHeight(suite))
	return utl.NewBalanceInWaves(sumBalanceMinersGo, sumBalanceMinersScala), utl.NewBalanceInWaves(balanceDaoGo, balanceDaoScala),
		utl.NewBalanceInWaves(balanceXtnGo, balanceXtnScala)
}

func getDiffBalance(suite *f.BaseSuite, addressType string, currentBalance utl.BalanceInWaves,
	initBalance utl.BalanceInWaves) utl.BalanceInWaves {
	diffBalanceGo := currentBalance.BalanceInWavesGo - initBalance.BalanceInWavesGo
	diffBalanceScala := currentBalance.BalanceInWavesScala - initBalance.BalanceInWavesScala
	suite.T().Logf("Go: Diff %s balance: %d on height: %d, Scala: Diff %s balance: %d, on height: %d",
		addressType, diffBalanceGo, utl.GetHeightGo(suite), addressType, diffBalanceScala, utl.GetHeightScala(suite))
	return utl.NewBalanceInWaves(diffBalanceGo, diffBalanceScala)
}

func getAddressesDiffBalances(suite *f.BaseSuite, currentSumMinersBalance, currentDaoBalance, currentXtnBalance,
	initSumMinersBalance, initDaoBalance, initXtnBalance utl.BalanceInWaves) (utl.BalanceInWaves, utl.BalanceInWaves, utl.BalanceInWaves) {
	//diff sum miners balances
	diffMinersSumBalances := getDiffBalance(suite, "Miners", currentSumMinersBalance, initSumMinersBalance)
	//diff dao balances
	diffDao := getDiffBalance(suite, "DAO", currentDaoBalance, initDaoBalance)
	//diff xtn
	diffXtn := getDiffBalance(suite, "XTN", currentXtnBalance, initXtnBalance)
	return diffMinersSumBalances, diffDao, diffXtn
}

func GetBlockRewardDistribution[T any](suite *f.BaseSuite, testdata testdata.RewardDistributionTestData[T]) (utl.RewardDiffBalancesInWaves, utl.RewardTerm) {
	//get init balance in waves of miners accounts
	suite.T().Log("Init Balances")
	initSumMinersBalance, initDaoBalance, initXtnBalance := getAddressesBalances(suite, testdata)
	//wait for 1 block
	utl.WaitForNewHeight(suite)
	h := utl.GetHeight(suite)
	term := utl.GetRewardTermAtHeight(suite, h)
	//get current balances of miners
	suite.T().Log("Current Balances")
	currentSumMinersBalance, currentDaoBalance, currentXtnBalance := getAddressesBalances(suite, testdata)
	//get diff balances
	diffMinersSumBalances, diffDaoBalance, diffXtnBalance := getAddressesDiffBalances(suite, currentSumMinersBalance,
		currentDaoBalance, currentXtnBalance, initSumMinersBalance, initDaoBalance, initXtnBalance)
	return utl.NewRewardDiffBalances(diffMinersSumBalances.BalanceInWavesGo, diffMinersSumBalances.BalanceInWavesScala,
		diffDaoBalance.BalanceInWavesGo, diffDaoBalance.BalanceInWavesScala, diffXtnBalance.BalanceInWavesGo,
		diffXtnBalance.BalanceInWavesScala), term
}
