package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/alias_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"golang.org/x/exp/maps"
)

type TransferTxSuite struct {
	f.BaseSuite
}

func (suite *TransferTxSuite) Test_TransferTxPositive() {
	versions := testdata.GetVersions()
	waitForTx := true
	for _, v := range versions {
		//создаем произвольный алиас
		alias := utl.RandStringBytes(15, testdata.AliasSymbolSet)
		//устанавливаем алиас аккаунту, которому будем пересылать токены
		alias_utilities.SetAliasToAccount(&suite.BaseSuite, v, utl.TestChainID, alias, utl.DefaultRecipientNotMiner)
		//выпускаем токен, который будем переводить другому аккаунту
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		//используя новый токен, создаем тестовые данные для проверки транзакции перевода
		tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, alias)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
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
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferTxMaxAmountAndFeePositive() {
	versions := testdata.GetVersions()
	waitForTx := true
	for _, v := range versions {
		//создаем новый аккаунт с ненулевым балансом
		n := transfer_utilities.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		//выпускаем токен, который будем переводить другому аккаунту
		itxID := issue_utilities.IssueAssetAmount(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultSenderNotMiner, utl.MaxAmount)
		//переводим токен с аккаунта эмитента на новый аккаунт
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, v, utl.TestChainID, itxID, utl.DefaultSenderNotMiner, n)
		//используя новый токен, создаем тестовые данные для проверки транзакции перевода
		tdmatrix := testdata.GetTransferMaxAmountPositive(&suite.BaseSuite, itxID, n)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
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
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferTxNegative() {
	versions := testdata.GetVersions()
	waitForTx := true
	for _, v := range versions {
		//выпускаем токен, который будем переводить другому аккаунту
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		//используя новый токен, создаем тестовые данные для проверки транзакции перевода
		tdmatrix := testdata.GetTransferNegativeData(&suite.BaseSuite, itx.TxID)

		if v > 2 {
			maps.Copy(tdmatrix, testdata.GetTransferChainIDNegativeData(&suite.BaseSuite, itx.TxID))
		}
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				//выпускаем транзакцию перевода
				tx, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID
				//проверяем балансы аккаунтов
				//баланс вавесов аккаунта, с которого переводят не меняется
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)
				//баланс ассетов аккаунта, с которого переводят, не меняется
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, name, "Version: ", v)
				//баланс вавесов аккаунта, на который переводят токен, не меняется
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)
				//баланс ассетов аккаунта, на который переводят, не меняется
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, name, "Version: ", v)
				//проверяем сообщения об ошибках
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, "Case: ", name, "Version: ", v)
			})
		}
		//проверяем, что ни одна из транзакций не попала в блокчейн
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestTransferTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxSuite))
}
