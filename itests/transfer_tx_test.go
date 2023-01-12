package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer_utilities"
)

type TransferTxSuite struct {
	f.BaseSuite
}

func (suite *TransferTxSuite) Test_TransferPositive() {
	versions := testdata.GetVersions()
	waitForTx := true
	for _, v := range versions {
		//выпускаем токен, который будем переводить другому аккаунту
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSend(&suite.BaseSuite, reissuable, v, waitForTx)
		//используя новый токен, создаем тестовые данные для проверки транзакции перевода
		tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			//suite.T().Run(name, func(t *testing.T) {})
			//выпускаем транзакцию перевода
			tx, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
				&suite.BaseSuite, td, v, waitForTx)
			utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name,
				"Transfer: "+tx.TxID.String(), "Version: ", v)
			//проверяем балансы аккаунтов
			//баланс вавесов аккаунта, с которого переводят, уменьшается на комиссию
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
				diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
				diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)
			//баланс ассетов аккаунта, с которого переводят, уменьшается на количество токена, переводимое другому аккаунту
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
				diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
				diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, name, "Version: ", v)
			//баланс вавесов аккаунта, на который переводят токен, не меняется
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
				diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
				diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)
			//баланс ассетов аккаунта, на который переводят, увеличивается на количество переводимого токена
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
				diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
				diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, name, "Version: ", v)
		}
	}
}

func TestTransferTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxSuite))
}
