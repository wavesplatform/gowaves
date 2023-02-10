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
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		alias := utl.RandStringBytes(15, testdata.AliasSymbolSet)
		alias_utilities.SetAliasToAccount(&suite.BaseSuite, v, utl.TestChainID, alias, utl.DefaultRecipientNotMiner)
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, alias)

		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferTxMaxAmountAndFeePositive() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		n := transfer_utilities.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		itxID := issue_utilities.IssueAssetAmount(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultSenderNotMiner, utl.MaxAmount)
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, v, utl.TestChainID, itxID, utl.DefaultSenderNotMiner, n)

		tdmatrix := testdata.GetTransferMaxAmountPositive(&suite.BaseSuite, itxID, n)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferTxNegative() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetTransferNegativeData(&suite.BaseSuite, itx.TxID)
		txIds := make(map[string]*crypto.Digest)

		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *TransferTxSuite) Test_TransferTxChainIDNegative() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetTransferChainIDChangedNegativeData(&suite.BaseSuite, itx.TxID)

		txIds := make(map[string]*crypto.Digest)

		if v > 2 {
			maps.Copy(tdmatrix, testdata.GetTransferChainIDDataNegative(&suite.BaseSuite, itx.TxID))
		}

		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				initBalanceWavesGoSender, initBalanceWavesScalaSender := utl.GetAvailableBalanceInWaves(
					&suite.BaseSuite, td.Sender.Address)
				initBalanceAssetGoSender, initBalanceAssetScalaSender := utl.GetAssetBalance(
					&suite.BaseSuite, td.Sender.Address, td.Asset.ID)

				tx := transfer_utilities.TransferSendWithTestData(&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID

				actualDiffBalanceWavesGoSender, actualDiffBalanceWavesScalaSender := utl.GetActualDiffBalanceInWaves(
					&suite.BaseSuite, td.Sender.Address, initBalanceWavesGoSender, initBalanceWavesScalaSender)

				actuallDiffBalanceAssetGoSender, actualDiffBalanceAssetScalaSender := utl.GetActualDiffBalanceInAssets(
					&suite.BaseSuite, td.Sender.Address, td.Asset.ID, initBalanceAssetGoSender, initBalanceAssetScalaSender)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceWavesGoSender,
					actualDiffBalanceWavesScalaSender, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actuallDiffBalanceAssetGoSender,
					actualDiffBalanceAssetScalaSender, utl.GetTestcaseNameWithVersion(name, v))

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *TransferTxSuite) Test_TransferTxChainIDBinaryVersions() {
	versions := []byte{1, 2}
	waitForTx := true
	name := "Skipping chainId for binary versions"
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		td := testdata.GetTransferChainIDDataBinaryVersions(&suite.BaseSuite, itx.TxID)

		txIds := make(map[string]*crypto.Digest)

		suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
			//txs should be successful because of ignoring chainID
			tx0, _, _ := transfer_utilities.SendTransferTxAndGetBalances(&suite.BaseSuite, td[0], v, waitForTx)
			txIds[name] = &tx0.TxID

			utl.TxInfoCheck(suite.T(), tx0.WtErr.ErrWtGo, tx0.WtErr.ErrWtScala, "Transfer: "+tx0.TxID.String(),
				utl.GetTestcaseNameWithVersion(name, v))

			tx1, diffBalancesSender, diffBalancesRecipient := transfer_utilities.SendTransferTxAndGetBalances(
				&suite.BaseSuite, td[1], v, waitForTx)
			txIds[name] = &tx1.TxID

			utl.TxInfoCheck(suite.T(), tx1.WtErr.ErrWtGo, tx1.WtErr.ErrWtScala, "Transfer: "+tx1.TxID.String(),
				utl.GetTestcaseNameWithVersion(name, v))

			utl.WavesDiffBalanceCheck(suite.T(), td[1].Expected.WavesDiffBalanceSender,
				diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
				diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

			utl.AssetBalanceCheck(suite.T(), td[1].Expected.AssetDiffBalance,
				diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
				diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

			utl.WavesDiffBalanceCheck(suite.T(), td[1].Expected.WavesDiffBalanceRecipient,
				diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
				diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

			utl.AssetBalanceCheck(suite.T(), td[1].Expected.AssetDiffBalance,
				diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
				diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

		})
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 2, "IDs: %#v", actualTxIds)
	}
}

func TestTransferTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxSuite))
}
