package reward

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func getSynchronizedBalances(
	suite *f.BaseSuite, addresses testdata.AddressesForDistribution,
) (utl.BalanceInWaves, utl.BalanceInWaves, utl.BalanceInWaves, proto.Height) {
	balances := suite.Clients.SynchronizedWavesBalances(suite.T(), addresses.AsList()...)

	goMinerBalance := balances.GetByAccountInfo(addresses.MinerGoAccount)
	scalaMinerBalance := balances.GetByAccountInfo(addresses.MinerScalaAccount)
	cumulativeMinersBalance := goMinerBalance.Add(scalaMinerBalance)
	suite.T().Logf("Go: Miners cumulative balance: %d; Scala: Miners cumulative balance: %d; Height: %d",
		cumulativeMinersBalance.GoBalance, cumulativeMinersBalance.ScalaBalance, balances.Height)

	daoBalance := balances.GetByAccountInfo(addresses.DaoAccount)
	suite.T().Logf("Go: DAO balance: %d; Scala: DAO balance: %d; Height: %d",
		daoBalance.GoBalance, daoBalance.ScalaBalance, balances.Height)

	xtnBuybackBalance := balances.GetByAccountInfo(addresses.XtnBuyBackAccount)
	suite.T().Logf("Go: XTN balance: %d; Scala: XTN balance: %d; Height: %d",
		xtnBuybackBalance.GoBalance, xtnBuybackBalance.ScalaBalance, balances.Height)

	return utl.NewBalanceInWaves(cumulativeMinersBalance.GoBalance, cumulativeMinersBalance.ScalaBalance),
		utl.NewBalanceInWaves(daoBalance.GoBalance, daoBalance.ScalaBalance),
		utl.NewBalanceInWaves(xtnBuybackBalance.GoBalance, xtnBuybackBalance.ScalaBalance), balances.Height
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
	utl.RewardDiffBalancesInWaves, utl.RewardTerm, proto.Height) {
	// Get balances in waves before block applied.
	suite.T().Log("Balances before applied block: ")
	initSumMinersBalance, initDaoBalance, initXtnBalance, initHeight := getSynchronizedBalances(suite, addresses)
	// Wait for 1 block.
	utl.WaitForHeight(suite, initHeight+1)
	// Get balances after block applied.
	suite.T().Log("Balances after applied block: ")
	currentSumMinersBalance, currentDaoBalance, currentXtnBalance, currentHeight := getSynchronizedBalances(suite,
		addresses)
	term := utl.GetRewardTermAtHeight(suite, currentHeight)
	// Get diff balances.
	suite.T().Log("Diff Balances: ")
	diffMinersSumBalances, diffDaoBalance, diffXtnBalance := getAddressesDiffBalances(suite, currentSumMinersBalance,
		currentDaoBalance, currentXtnBalance, initSumMinersBalance, initDaoBalance, initXtnBalance,
		initHeight, currentHeight)

	if hd := currentHeight - initHeight; hd != 1 {
		suite.T().Fatalf("Heights difference %d is not equal to 1", hd)
	}
	return utl.NewRewardDiffBalances(diffMinersSumBalances.BalanceInWavesGo, diffMinersSumBalances.BalanceInWavesScala,
		diffDaoBalance.BalanceInWavesGo, diffDaoBalance.BalanceInWavesScala, diffXtnBalance.BalanceInWavesGo,
		diffXtnBalance.BalanceInWavesScala), term, currentHeight
}

type GetTestData func(
	suite *f.BaseSuite, addresses testdata.AddressesForDistribution, height uint64,
) testdata.RewardDistributionTestData[testdata.RewardDistributionExpectedValues]

func GetRewardDistributionAndChecks(suite *f.BaseSuite, addresses testdata.AddressesForDistribution,
	testdata GetTestData) {
	// Get reward for 1 block.
	rewardDistributions, term, h := GetBlockRewardDistribution(suite, addresses)
	// Get expected results on current height
	td := testdata(suite, addresses, h)
	// Check results.
	utl.TermCheck(suite.T(), td.Expected.Term, term.TermGo, term.TermScala)
	utl.MinersSumDiffBalanceInWavesCheck(suite.T(), td.Expected.MinersSumDiffBalance,
		rewardDistributions.MinersSumDiffBalance.BalanceInWavesGo,
		rewardDistributions.MinersSumDiffBalance.BalanceInWavesScala)
	utl.DaoDiffBalanceInWavesCheck(suite.T(), td.Expected.DaoDiffBalance,
		rewardDistributions.DAODiffBalance.BalanceInWavesGo,
		rewardDistributions.DAODiffBalance.BalanceInWavesScala)
	utl.XtnBuyBackDiffBalanceInWavesCheck(suite.T(), td.Expected.XtnDiffBalance,
		rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesGo,
		rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesScala)
}

func GetRewardInfoAndChecks(suite *f.BaseSuite,
	td testdata.RewardDistributionApiTestData[testdata.RewardInfoApiExpectedValues]) {
	rewardInfoGo, rewardInfoScala := utl.GetRewards(suite)
	utl.TermCheck(suite.T(), td.Expected.Term, rewardInfoGo.Term, rewardInfoScala.Term)
	utl.NextCheckParameterCheck(suite.T(), td.Expected.NextCheck, rewardInfoGo.NextCheck, rewardInfoScala.NextCheck)
	utl.VotingIntervalStartCheck(suite.T(), td.Expected.VotingIntervalStart, rewardInfoGo.VotingIntervalStart,
		rewardInfoScala.VotingIntervalStart)
}

func GetRewardInfoAtHeightAndChecks(suite *f.BaseSuite,
	td testdata.RewardDistributionApiTestData[testdata.RewardInfoApiExpectedValues], height uint64) {
	rewardInfoGo, rewardInfoScala := utl.GetRewardsAtHeight(suite, height)
	utl.TermCheck(suite.T(), td.Expected.Term, rewardInfoGo.Term, rewardInfoScala.Term)
	utl.NextCheckParameterCheck(suite.T(), td.Expected.NextCheck, rewardInfoGo.NextCheck, rewardInfoScala.NextCheck)
	utl.VotingIntervalStartCheck(suite.T(), td.Expected.VotingIntervalStart, rewardInfoGo.VotingIntervalStart,
		rewardInfoScala.VotingIntervalStart)
}
