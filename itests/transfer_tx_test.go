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
)

type TransferTxSuite struct {
	f.BaseSuite
}

func (suite *TransferTxSuite) Test_TransferTxPositive() {
	versions := testdata.GetVersions()
	waitForTx := true
	for _, v := range versions {
		alias := utl.RandStringBytes(15, testdata.AliasSymbolSet)
		alias_utilities.SetAliasToAccount(&suite.BaseSuite, v, utl.TestChainID, alias, utl.DefaultRecipientNotMiner)
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, alias)

		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name,
					"Transfer: "+tx.TxID.String(), "Version: ", v)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, name, "Version: ", v)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)

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
		n := transfer_utilities.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		itxID := issue_utilities.IssueAssetAmount(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultSenderNotMiner, utl.MaxAmount)
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, v, utl.TestChainID, itxID, utl.DefaultSenderNotMiner, n)

		tdmatrix := testdata.GetTransferMaxAmountPositive(&suite.BaseSuite, itxID, n)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name,
					"Transfer: "+tx.TxID.String(), "Version: ", v)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, name, "Version: ", v)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)

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
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetTransferNegativeData(&suite.BaseSuite, itx.TxID)

		/*if v > 2 {
			maps.Copy(tdmatrix, testdata.GetTransferChainIDNegativeData(&suite.BaseSuite, itx.TxID))
		}*/
		txIds := make(map[string]*crypto.Digest)

		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, name, "Version: ", v)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, name, "Version: ", v)

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, name, "Version: ", v)

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, "Case: ", name, "Version: ", v)
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestTransferTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxSuite))
}
