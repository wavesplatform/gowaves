package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

type RewardDistributionSuite struct {
	f.BaseSuite
}

func (suite *RewardDistributionSuite) Test_RewardDistributionPositive() {
	h := utl.GetHeight(&suite.BaseSuite)
	fmt.Println(h)
	fmt.Println("Miner 1 Go Balance:", utl.GetAvailableBalanceInWavesGo(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 0).Address))
	fmt.Println("Miner 1 Scala Balance:", utl.GetAvailableBalanceInWavesScala(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 0).Address))
	fmt.Println("Miner 2 Address Go Balance:", utl.GetAvailableBalanceInWavesGo(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 1).Address))
	fmt.Println("Miner 2 Address Scala Balance:", utl.GetAvailableBalanceInWavesScala(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 1).Address))
	fmt.Println("DAO Address Go Balance:", utl.GetAvailableBalanceInWavesGo(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 5).Address))
	fmt.Println("DAO Address Scala Balance:", utl.GetAvailableBalanceInWavesScala(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 5).Address))
	fmt.Println("XTN buy-back Address Go Balance:", utl.GetAvailableBalanceInWavesGo(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 6).Address))
	fmt.Println("XTN buy-back Address Scala Balance:", utl.GetAvailableBalanceInWavesScala(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 6).Address))
	fmt.Println("Go feature 20 status:", utl.GetFeatureBlockchainStatusGo(&suite.BaseSuite, 20, h))
	fmt.Println("Go feature 14 status:", utl.GetFeatureBlockchainStatusGo(&suite.BaseSuite, 14, h))
	fmt.Println("Scala feature 20 status:", utl.GetFeatureBlockchainStatusScala(&suite.BaseSuite, 20, h))
	fmt.Println("Scala feature 14 status:", utl.GetFeatureBlockchainStatusScala(&suite.BaseSuite, 14, h))
	fmt.Println("Go feature info:", utl.GetActivationFeaturesStatusInfoGo(&suite.BaseSuite, h))
	fmt.Println("Scala feature info:", utl.GetActivationFeaturesStatusInfoScala(&suite.BaseSuite, h))
	for i := 0; i < 10; i++ {
		utl.WaitForHeight(&suite.BaseSuite, h+1)
		h = utl.GetHeight(&suite.BaseSuite)
		fmt.Println(h)
		fmt.Println("Miner 1 Go Balance:", utl.GetAvailableBalanceInWavesGo(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 0).Address))
		fmt.Println("Miner 1 Scala Balance:", utl.GetAvailableBalanceInWavesScala(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 0).Address))
		fmt.Println("Miner 2 Address Go Balance:", utl.GetAvailableBalanceInWavesGo(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 1).Address))
		fmt.Println("Miner 2 Address Scala Balance:", utl.GetAvailableBalanceInWavesScala(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 1).Address))
		fmt.Println("DAO Address Go Balance:", utl.GetAvailableBalanceInWavesGo(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 5).Address))
		fmt.Println("DAO Address Scala Balance:", utl.GetAvailableBalanceInWavesScala(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 5).Address))
		fmt.Println("XTN buy-back Address Go Balance:", utl.GetAvailableBalanceInWavesGo(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 6).Address))
		fmt.Println("XTN buy-back Address Scala Balance:", utl.GetAvailableBalanceInWavesScala(&suite.BaseSuite, utl.GetAccount(&suite.BaseSuite, 6).Address))
		fmt.Println("Go feature 20 status:", utl.GetFeatureBlockchainStatusGo(&suite.BaseSuite, 20, h))
		fmt.Println("Go feature 14 status:", utl.GetFeatureBlockchainStatusGo(&suite.BaseSuite, 14, h))
		fmt.Println("Scala feature 20 status:", utl.GetFeatureBlockchainStatusScala(&suite.BaseSuite, 20, h))
		fmt.Println("Scala feature 14 status:", utl.GetFeatureBlockchainStatusScala(&suite.BaseSuite, 14, h))
		fmt.Println("Go feature info:", utl.GetActivationFeaturesStatusInfoGo(&suite.BaseSuite, h))
		fmt.Println("Scala feature info:", utl.GetActivationFeaturesStatusInfoScala(&suite.BaseSuite, h))
	}
}

func TestRewardDistributionSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionSuite))
}
