package reward_utilities

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
)

func GetDesiredRewardGo(suite *f.BaseSuite, height uint64) int64 {
	block := suite.Clients.GoClients.GrpcClient.GetBlock(suite.T(), height).GetBlock()
	return block.GetHeader().RewardVote
}

func GetDesiredRewardScala(suite *f.BaseSuite, height uint64) int64 {
	return suite.Clients.ScalaClients.GrpcClient.GetBlock(suite.T(), height).GetBlock().GetHeader().RewardVote
}

func GetInitReward(suite *f.BaseSuite) uint64 {
	return suite.Cfg.BlockchainSettings.InitialBlockReward
}

func GetRewardIncrement(suite *f.BaseSuite) uint64 {
	return suite.Cfg.BlockchainSettings.BlockRewardIncrement
}

// GetBlockRewardVotingPeriod returns voting interval (voting-interval)
// the interval in which votes for increasing/decreasing the reward are taken into account
func GetBlockRewardVotingPeriod(suite *f.BaseSuite) uint64 {
	return suite.Cfg.BlockchainSettings.BlockRewardVotingPeriod
}

// GetRewardTerm is max period of voting (term)
func GetRewardTerm(suite *f.BaseSuite) uint64 {
	return suite.Cfg.BlockchainSettings.BlockRewardTerm
}

// GetRewardTermAfter20 returns term after feature 20 activation (term-after-capped-reward-feature), =1/2 term
func GetRewardTermAfter20(suite *f.BaseSuite) uint64 {
	return suite.Cfg.BlockchainSettings.BlockRewardTermAfter20
}
