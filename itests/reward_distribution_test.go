package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reward_utilities"
)

type RewardDistributionSuite struct {
	f.BaseSuite
}

//tests with "BaseSuite" cfg

// NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20)
func (suite *RewardDistributionSuite) Test_RewardDistributionPositive() {
	h := utl.GetHeight(&suite.BaseSuite)
	h = utl.WaitForHeight(&suite.BaseSuite, 6)
	//feature 14 should be activated
	utl.FeatureShouldBeActivated(&suite.BaseSuite, 14, h)
	//feature 19 should be activated
	utl.FeatureShouldBeActivated(&suite.BaseSuite, 19, h)
	//feature 20 should be activated
	utl.FeatureShouldBeActivated(&suite.BaseSuite, 20, h)
	fmt.Println("Init height", h)
	fmt.Println("Desired Reward Go: ", reward_utilities.GetDesiredRewardGo(&suite.BaseSuite, h), "Desired Reward Scala: ", reward_utilities.GetDesiredRewardScala(&suite.BaseSuite, h))
	fmt.Println("Init reward : ", reward_utilities.GetInitReward(&suite.BaseSuite))
	fmt.Println("Increment: ", reward_utilities.GetRewardIncrement(&suite.BaseSuite))
	fmt.Println("voting-interval: ", reward_utilities.GetBlockRewardVotingPeriod(&suite.BaseSuite))
	fmt.Println("Max term: ", reward_utilities.GetRewardTerm(&suite.BaseSuite))
	fmt.Println("Term after f20 : ", reward_utilities.GetRewardTermAfter20(&suite.BaseSuite))

	//reward_utilities.GetBlockRewardInfo(&suite.BaseSuite, h)
	//init data
	//get init balance in waves of miners accounts
	initBalanceMiner1Go, initBalanceMiner1Scala := utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 0).Address)
	initBalanceMiner2Go, initBalanceMiner2Scala := utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 1).Address)
	//we will be summing up balances of both miners accounts
	initSumBalanceMinersGo := initBalanceMiner1Go + initBalanceMiner2Go
	initSumBalanceMinersScala := initBalanceMiner1Scala + initBalanceMiner2Scala
	fmt.Println("Init miner1 balance Go: ", initBalanceMiner1Go, "Init miner1 balance Scala: ", initBalanceMiner1Scala)
	fmt.Println("Init miner2 balance Go: ", initBalanceMiner2Go, "Init miner2 balance Scala: ", initBalanceMiner2Scala)
	fmt.Println("Init sum balance Go: ", initSumBalanceMinersGo, "Init sum balance Scala: ", initSumBalanceMinersScala)

	//get init balances of dao and xtn buy-back accounts
	initBalanceDaoGo, initBalanceDaoScala := utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 5).Address)
	initBalanceXtnGo, initBalanceXtnScala := utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 6).Address)
	fmt.Println("Init DAO balance Go: ", initBalanceDaoGo, "Init DAO balance Scala: ", initBalanceDaoScala)
	fmt.Println("Init XTN balance Go: ", initBalanceXtnGo, "Init XTN balance Scala: ", initBalanceXtnScala)

	//wait for 1 block
	utl.WaitForHeight(&suite.BaseSuite, h+1)
	//get current balances of miners
	currentBalanceMiner1Go, currentBalanceMiner1Scala := utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 0).Address)
	currentBalanceMiner2Go, currentBalanceMiner2Scala := utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 1).Address)
	currentSumBalanceMinersGo := currentBalanceMiner1Go + currentBalanceMiner2Go
	currentSumBalanceMinersScala := currentBalanceMiner1Scala + currentBalanceMiner2Scala
	fmt.Println("Current miner1 balance Go: ", currentBalanceMiner1Go, "Current miner1 balance Scala: ", currentBalanceMiner1Scala)
	fmt.Println("Current miner2 balance Go: ", currentBalanceMiner2Go, "Current miner2 balance Scala: ", currentBalanceMiner2Scala)
	fmt.Println("Current sum balance Go: ", currentSumBalanceMinersGo, "Current sum balance Scala: ", currentSumBalanceMinersScala)

	//get current dao and xtn buy-back balance
	currentBalanceDaoGo, currentBalanceDaoScala := utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 5).Address)
	currentBalanceXtnGo, currentBalanceXtnScala := utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 6).Address)
	fmt.Println("Current DAO balance Go: ", currentBalanceDaoGo, "Current DAO balance Scala: ", currentBalanceDaoScala)
	fmt.Println("Current XTN balance Go: ", currentBalanceXtnGo, "Current XTN balance Scala: ", currentBalanceXtnScala)
	//check diff miners balance
	diffMinersSumBalancesGo := currentSumBalanceMinersGo - initSumBalanceMinersGo
	diffMinersSumBalancesScala := currentSumBalanceMinersScala - initSumBalanceMinersScala
	fmt.Println("Diff sum balances: ", diffMinersSumBalancesGo, diffMinersSumBalancesScala)
	//check diff dao
	diffDaoGo := currentBalanceDaoGo - initBalanceDaoGo
	diffDaoScala := currentBalanceDaoScala - initBalanceDaoScala
	fmt.Println("Diff DAO balances: ", diffDaoGo, diffDaoScala)
	//check diff xtn
	diffXtnGo := currentBalanceXtnGo - initBalanceXtnGo
	diffXtnScala := currentBalanceXtnScala - initBalanceXtnScala
	fmt.Println("Diff XTN balances: ", diffXtnGo, diffXtnScala)

	//wait voting period
	utl.WaitForHeight(&suite.BaseSuite, h+suite.Cfg.BlockchainSettings.BlockRewardTerm)
	h = utl.GetHeight(&suite.BaseSuite)
	fmt.Println("Height after waiting voting period", h)
	fmt.Println("Desired Reward Go: ", reward_utilities.GetDesiredRewardGo(&suite.BaseSuite, h), "Desired Reward Scala: ", reward_utilities.GetDesiredRewardScala(&suite.BaseSuite, h))

	//get init balance in waves of miners accounts
	initBalanceMiner1Go, initBalanceMiner1Scala = utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 0).Address)
	initBalanceMiner2Go, initBalanceMiner2Scala = utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 1).Address)
	//we will be summing up balances of both miners accounts
	initSumBalanceMinersGo = initBalanceMiner1Go + initBalanceMiner2Go
	initSumBalanceMinersScala = initBalanceMiner1Scala + initBalanceMiner2Scala
	fmt.Println("Init miner1 balance Go after voting: ", initBalanceMiner1Go, "Init miner1 balance Scala after voting: ", initBalanceMiner1Scala)
	fmt.Println("Init miner2 balance Go after voting: ", initBalanceMiner2Go, "Init miner2 balance Scala after voting: ", initBalanceMiner2Scala)
	fmt.Println("Init sum balance Go after voting: ", initSumBalanceMinersGo, "Init sum balance Scala after voting: ", initSumBalanceMinersScala)

	//get init balances of dao and xtn buy-back accounts
	initBalanceDaoGo, initBalanceDaoScala = utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 5).Address)
	initBalanceXtnGo, initBalanceXtnScala = utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 6).Address)
	fmt.Println("Init DAO balance Go after voting: ", initBalanceDaoGo, "Init DAO balance Scala after voting: ", initBalanceDaoScala)
	fmt.Println("Init XTN balance Go after voting: ", initBalanceXtnGo, "Init XTN balance Scala after voting: ", initBalanceXtnScala)

	//wait for 1 block
	utl.WaitForHeight(&suite.BaseSuite, h+1)
	//get current balances of miners
	currentBalanceMiner1Go, currentBalanceMiner1Scala = utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 0).Address)
	currentBalanceMiner2Go, currentBalanceMiner2Scala = utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 1).Address)
	currentSumBalanceMinersGo = currentBalanceMiner1Go + currentBalanceMiner2Go
	currentSumBalanceMinersScala = currentBalanceMiner1Scala + currentBalanceMiner2Scala
	fmt.Println("Current miner1 balance Go after voting: ", currentBalanceMiner1Go, "Current miner1 balance Scala after voting: ", currentBalanceMiner1Scala)
	fmt.Println("Current miner2 balance Go after voting: ", currentBalanceMiner2Go, "Current miner2 balance Scala after voting: ", currentBalanceMiner2Scala)
	fmt.Println("Current sum balance Go after voting: ", currentSumBalanceMinersGo, "Current sum balance Scala after voting: ", currentSumBalanceMinersScala)

	//get current dao and xtn buy-back balance
	currentBalanceDaoGo, currentBalanceDaoScala = utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 5).Address)
	currentBalanceXtnGo, currentBalanceXtnScala = utl.GetAvailableBalanceInWaves(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 6).Address)
	fmt.Println("Current DAO balance Go after voting: ", currentBalanceDaoGo, "Current DAO balance Scala after voting: ", currentBalanceDaoScala)
	fmt.Println("Current XTN balance Go after voting: ", currentBalanceXtnGo, "Current XTN balance Scala after voting: ", currentBalanceXtnScala)
	//check diff miners balance
	diffMinersSumBalancesGo = currentSumBalanceMinersGo - initSumBalanceMinersGo
	diffMinersSumBalancesScala = currentSumBalanceMinersScala - initSumBalanceMinersScala
	fmt.Println("Diff sum balances after voting: ", diffMinersSumBalancesGo, diffMinersSumBalancesScala)
	//check diff dao
	diffDaoGo = currentBalanceDaoGo - initBalanceDaoGo
	diffDaoScala = currentBalanceDaoScala - initBalanceDaoScala
	fmt.Println("Diff DAO balances after voting: ", diffDaoGo, diffDaoScala)
	//check diff xtn
	diffXtnGo = currentBalanceXtnGo - initBalanceXtnGo
	diffXtnScala = currentBalanceXtnScala - initBalanceXtnScala
	fmt.Println("Diff XTN balances after voting: ", diffXtnGo, diffXtnScala)
}

func TestRewardDistributionSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionSuite))
}
