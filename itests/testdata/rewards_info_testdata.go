package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type RewardDistributionApiTestData[T any] struct {
	Expected T
}

type Votes struct {
	Increase uint32
	Decrease uint32
}

type RewardInfoApiExpectedValues struct {
	TotalWavesAmount    uint64
	CurrentReward       uint64
	MinIncrement        uint64
	Term                uint64
	NextCheck           uint64
	VotingIntervalStart uint64
	VotingInterval      uint64
	Votes               Votes
	DaoAddress          *proto.WavesAddress
	XtnAddress          *proto.WavesAddress
	_                   struct{}
}

func NewRewardDistributionApiTestData[T any](expected T) RewardDistributionApiTestData[T] {
	return RewardDistributionApiTestData[T]{
		Expected: expected,
	}
}

func GetRewardInfoApiAfterPreactivated20TestData(suite *f.BaseSuite) RewardDistributionApiTestData[RewardInfoApiExpectedValues] {
	totalWavesAmount := utl.GetTotalWavesAmount(suite)
	term := utl.GetRewardTermAfter20Cfg(suite)
	minIncrement := utl.GetIncrementCfg(suite)
	votingInterval := utl.GetVotingIntervalCfg(suite)
	daoAddress, xtnAddress := utl.GetDaoAndXtnAddresses(suite)
	return NewRewardDistributionApiTestData(
		RewardInfoApiExpectedValues{
			TotalWavesAmount:    totalWavesAmount,
			MinIncrement:        minIncrement,
			Term:                term,
			NextCheck:           term,
			VotingIntervalStart: term,
			VotingInterval:      votingInterval,
			Votes: Votes{
				Decrease: 0,
				Increase: 1,
			},
			DaoAddress: daoAddress,
			XtnAddress: xtnAddress,
		})
}

func GetRewardInfoApiAfterSupported20TestData(suite *f.BaseSuite) RewardDistributionApiTestData[RewardInfoApiExpectedValues] {
	totalWavesAmount := utl.GetTotalWavesAmount(suite)
	term := utl.GetRewardTermAfter20Cfg(suite)
	minIncrement := utl.GetIncrementCfg(suite)
	votingInterval := utl.GetVotingIntervalCfg(suite)
	daoAddress, xtnAddress := utl.GetDaoAndXtnAddresses(suite)
	return NewRewardDistributionApiTestData(
		RewardInfoApiExpectedValues{
			TotalWavesAmount:    totalWavesAmount,
			MinIncrement:        minIncrement,
			Term:                term,
			NextCheck:           3 * term,
			VotingIntervalStart: 3 * term,
			VotingInterval:      votingInterval,
			Votes: Votes{
				Decrease: 0,
				Increase: 1,
			},
			DaoAddress: daoAddress,
			XtnAddress: xtnAddress,
		})
}

func GetRewardInfoApiBefore20TestData(suite *f.BaseSuite) RewardDistributionApiTestData[RewardInfoApiExpectedValues] {
	totalWavesAmount := utl.GetTotalWavesAmount(suite)
	term := utl.GetRewardTermCfg(suite)
	minIncrement := utl.GetIncrementCfg(suite)
	votingInterval := utl.GetVotingIntervalCfg(suite)
	daoAddress, xtnAddress := utl.GetDaoAndXtnAddresses(suite)
	return NewRewardDistributionApiTestData(
		RewardInfoApiExpectedValues{
			TotalWavesAmount:    totalWavesAmount,
			MinIncrement:        minIncrement,
			Term:                term,
			NextCheck:           term,
			VotingIntervalStart: term,
			VotingInterval:      votingInterval,
			DaoAddress:          daoAddress,
			XtnAddress:          xtnAddress,
		})
}
