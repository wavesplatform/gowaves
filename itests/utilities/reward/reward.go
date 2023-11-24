package reward

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

func getAddressesBalances(suite *f.BaseSuite,
	addresses testdata.AddressesForDistribution) (utl.BalanceInWaves, utl.BalanceInWaves, utl.BalanceInWaves) {
	var balanceMiner1Go, balanceMiner1Scala, balanceMiner2Go, balanceMiner2Scala,
		balanceDaoGo, balanceDaoScala, balanceXtnGo, balanceXtnScala int64
	// Get balance in waves of miners accounts.
	if addresses.MinerGoAccount != nil {
		balanceMiner1Go, balanceMiner1Scala = utl.GetAvailableBalanceInWaves(suite, addresses.MinerGoAccount.Address)
	}
	if addresses.MinerScalaAccount != nil {
		balanceMiner2Go, balanceMiner2Scala = utl.GetAvailableBalanceInWaves(suite, addresses.MinerScalaAccount.Address)
	}
	// We will be summing up balances of both miners accounts.
	sumBalanceMinersGo := balanceMiner1Go + balanceMiner2Go
	sumBalanceMinersScala := balanceMiner1Scala + balanceMiner2Scala
	suite.T().Logf("Go: Sum Miners balance: %d, Go current height:%d, "+
		"Scala: Sum Miners balance: %d, Scala current height: %d",
		sumBalanceMinersGo, utl.GetHeightGo(suite), sumBalanceMinersScala, utl.GetHeightScala(suite))
	// Get balances of dao and xtn buy-back accounts.
	if addresses.DaoAccount != nil {
		balanceDaoGo, balanceDaoScala = utl.GetAvailableBalanceInWaves(suite, addresses.DaoAccount.Address)
	}
	suite.T().Logf("Go: DAO balance: %d, Go current height:%d, Scala: DAO balance: %d, Scala current height: %d",
		balanceDaoGo, utl.GetHeightGo(suite), balanceDaoScala, utl.GetHeightScala(suite))
	if addresses.XtnBuyBackAccount != nil {
		balanceXtnGo, balanceXtnScala = utl.GetAvailableBalanceInWaves(suite, addresses.XtnBuyBackAccount.Address)
	}
	suite.T().Logf("Go: XTN balance: %d, Go current height:%d, Scala: XTN balance: %d, Scala current height: %d",
		balanceXtnGo, utl.GetHeightGo(suite), balanceXtnScala, utl.GetHeightScala(suite))
	return utl.NewBalanceInWaves(sumBalanceMinersGo, sumBalanceMinersScala),
		utl.NewBalanceInWaves(balanceDaoGo, balanceDaoScala), utl.NewBalanceInWaves(balanceXtnGo, balanceXtnScala)
}

func getDiffBalance(suite *f.BaseSuite, addressType string, currentBalance utl.BalanceInWaves, currentHeight uint64,
	initBalance utl.BalanceInWaves, initHeight uint64) utl.BalanceInWaves {
	diffBalanceGo := currentBalance.BalanceInWavesGo - initBalance.BalanceInWavesGo
	diffBalanceScala := currentBalance.BalanceInWavesScala - initBalance.BalanceInWavesScala
	suite.T().Logf("Go: Diff %s balance: %d on heights: %d - %d, Scala: Diff %s balance: %d, on heights: %d - %d",
		addressType, diffBalanceGo, initHeight, currentHeight,
		addressType, diffBalanceScala, initHeight, currentHeight)
	return utl.NewBalanceInWaves(diffBalanceGo, diffBalanceScala)
}

func getAddressesDiffBalances(suite *f.BaseSuite, currentSumMinersBalance, currentDaoBalance, currentXtnBalance,
	initSumMinersBalance, initDaoBalance, initXtnBalance utl.BalanceInWaves,
	initHeight, currentHeight uint64) (utl.BalanceInWaves, utl.BalanceInWaves, utl.BalanceInWaves) {
	// Diff sum miners balances.
	diffMinersSumBalances := getDiffBalance(suite, "Miners", currentSumMinersBalance, currentHeight,
		initSumMinersBalance, initHeight)
	// Diff dao balances.
	diffDao := getDiffBalance(suite, "DAO", currentDaoBalance, currentHeight, initDaoBalance, initHeight)
	// Diff xtn.
	diffXtn := getDiffBalance(suite, "XTN", currentXtnBalance, currentHeight, initXtnBalance, initHeight)
	return diffMinersSumBalances, diffDao, diffXtn
}

func GetBlockRewardDistribution(suite *f.BaseSuite, addresses testdata.AddressesForDistribution) (
	utl.RewardDiffBalancesInWaves, utl.RewardTerm) {
	// Get balances in waves before block applied.
	suite.T().Log("Balances before applied block: ")
	initHeight := utl.GetHeight(suite)
	initSumMinersBalance, initDaoBalance, initXtnBalance := getAddressesBalances(suite, addresses)
	// Wait for 1 block.
	utl.WaitForNewHeight(suite)
	// Get balances after block applied.
	suite.T().Log("Balances after applied block: ")
	currentHeight := utl.GetHeight(suite)
	currentSumMinersBalance, currentDaoBalance, currentXtnBalance := getAddressesBalances(suite, addresses)
	term := utl.GetRewardTermAtHeight(suite, currentHeight)
	// Get diff balances.
	suite.T().Log("Diff Balances: ")
	diffMinersSumBalances, diffDaoBalance, diffXtnBalance := getAddressesDiffBalances(suite, currentSumMinersBalance,
		currentDaoBalance, currentXtnBalance, initSumMinersBalance, initDaoBalance, initXtnBalance,
		initHeight, currentHeight)
	return utl.NewRewardDiffBalances(diffMinersSumBalances.BalanceInWavesGo, diffMinersSumBalances.BalanceInWavesScala,
		diffDaoBalance.BalanceInWavesGo, diffDaoBalance.BalanceInWavesScala, diffXtnBalance.BalanceInWavesGo,
		diffXtnBalance.BalanceInWavesScala), term
}
