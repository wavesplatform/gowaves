package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

type RewardDistributionTestData[T any] struct {
	Miner1Account     config.AccountInfo
	Miner2Account     config.AccountInfo
	DaoAccount        config.AccountInfo
	XtnBuyBackAccount config.AccountInfo
	Expected          T
}

type RewardDistributionExpectedValuesPositive struct {
	MinersSumDiffBalance uint64
	DaoDiffBalance       uint64
	XtnDiffBalance       uint64
	_                    struct{}
}

func NewRewardDistributionTestData[T any](miner1Account, miner2Account, daoAccount, xtnAccount config.AccountInfo, expected T) RewardDistributionTestData[T] {
	return RewardDistributionTestData[T]{
		Miner1Account:     miner1Account,
		Miner2Account:     miner2Account,
		DaoAccount:        daoAccount,
		XtnBuyBackAccount: xtnAccount,
		Expected:          expected,
	}
}

func GetRewardDistributionTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	//need to change to current reward for other cases!!!!!
	return NewRewardDistributionTestData(
		utl.GetAccount(suite, utl.DefaultMinerGo),
		utl.GetAccount(suite, utl.DefaultMinerScala),
		utl.GetAccount(suite, utl.DAOAccount),
		utl.GetAccount(suite, utl.XTNBuyBackAccount),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: utl.GetInitReward(suite) - 200000000 - 200000000,
			DaoDiffBalance:       200000000,
			XtnDiffBalance:       200000000,
		})
}
