package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

type BoostBlockRewardTestSuite struct {
	f.BoostBlockRewardSuite
}

func (s *BoostBlockRewardTestSuite) SetupSuite() {
	s.BoostBlockRewardSuite.SetupSuite()

}

func (s *BoostBlockRewardTestSuite) Test_BoostBlockReward() {

	s.Run("", func() {
		// State Hashes at Height = 1 (GENESIS)
		//StateHashes
		/*stateHashGo := s.BoostBlockRewardSuite.Clients.GoClient.HTTPClient.StateHash(s.BoostBlockRewardSuite.BaseSuite.T(), 1)
		stateHashGo1 := utl.GetStateHashAsJsonGo(&s.BoostBlockRewardSuite.BaseSuite, 1)
		stateHashScala := s.BoostBlockRewardSuite.Clients.ScalaClient.HTTPClient.StateHash(s.BoostBlockRewardSuite.BaseSuite.T(), 1)
		stateHashScala1 := utl.GetStateHashAsJsonScala(&s.BoostBlockRewardSuite.BaseSuite, 1)
		fmt.Println("State Hash Go at Genesis = ", stateHashGo)
		fmt.Println("State Hash Go 1 at Genesis = ", stateHashGo1)
		fmt.Println("State Hash Scala at Genesis = ", stateHashScala)
		fmt.Println("State Hash Scala 1 at Genesis = ", stateHashScala1)*/
		for i := 1; i <= 10; i++ {

			heightGo := utl.GetHeightGo(&s.BoostBlockRewardSuite.BaseSuite)
			heightScala := utl.GetHeightScala(&s.BoostBlockRewardSuite.BaseSuite)
			fmt.Println("Height Go = ", heightGo)
			fmt.Println("Height Scala = ", heightScala)

			//StateHashes
			// shGo := s.BoostBlockRewardSuite.Clients.GoClient.HTTPClient.StateHash(s.BoostBlockRewardSuite.BaseSuite.T(), heightGo)
			// shGo1 := utl.GetStateHashAsJsonGo(&s.BoostBlockRewardSuite.BaseSuite, heightGo)
			// shScala := s.BoostBlockRewardSuite.Clients.ScalaClient.HTTPClient.StateHash(s.BoostBlockRewardSuite.BaseSuite.T(), heightScala)
			// shScala1 := utl.GetStateHashAsJsonScala(&s.BoostBlockRewardSuite.BaseSuite, heightScala)
			// fmt.Println("State Hash Go = ", shGo)
			// fmt.Println("State Hash Go 1 = ", shGo1)
			// fmt.Println("State Hash Scala = ", shScala)
			// fmt.Println("State Hash Scala 1 = ", shScala1)

			//Miner's Balances
			minerGoBalanceFromGo, minerGoBalanceFromScala := utl.GetAvailableBalanceInWaves(&s.BoostBlockRewardSuite.BaseSuite,
				utl.GetAccount(&s.BoostBlockRewardSuite.BaseSuite, utl.DefaultMinerGo).Address)
			minerScalaBalanceFromGo, minerScalaBalanceFromScala := utl.GetAvailableBalanceInWaves(&s.BoostBlockRewardSuite.BaseSuite,
				utl.GetAccount(&s.BoostBlockRewardSuite.BaseSuite, utl.DefaultMinerScala).Address)
			fmt.Println(fmt.Sprintf("Go miner balance: FromGoNode: %d, FromScala: %d",
				minerGoBalanceFromGo, minerGoBalanceFromScala))
			fmt.Println(fmt.Sprintf("Scala miner balance: FromGoNode: %d, FromScalaNode: %d",
				minerScalaBalanceFromGo, minerScalaBalanceFromScala))

			utl.WaitForNewHeight(&s.BoostBlockRewardSuite.BaseSuite)
		}
	})

}

func TestBoostBlockRewardTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BoostBlockRewardTestSuite))
}
