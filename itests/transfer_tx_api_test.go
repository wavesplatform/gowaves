package itests

import (
	"net/http"
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

type TransferTxApiSuite struct {
	f.BaseSuite
}

func (suite *TransferTxApiSuite) Test_TransferTxApiPositive() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		alias := utl.RandStringBytes(15, testdata.AliasSymbolSet)
		alias_utilities.SetAliasToAccountByAPI(&suite.BaseSuite, v, utl.TestChainID, alias, utl.DefaultRecipientNotMiner)
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, alias)

		if v <= 2 {
			maps.Copy(tdmatrix, testdata.GetTransferChainIDDataBinaryVersions(&suite.BaseSuite, itx.TxID))
		}

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, waitForTx)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, errMsg)
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					errMsg)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					errMsg)
			})
		}
	}
}

func (suite *TransferTxApiSuite) Test_TransferSmartAssetApiPositive() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	saversions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	waitForTx := true
	name := "Check transfer smart asset"
	for _, v := range versions {
		for _, sav := range saversions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, smart, sav, waitForTx)
			td := testdata.GetCommonTransferData(&suite.BaseSuite, &itx.TxID).Smart
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, waitForTx)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, errMsg)
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					errMsg)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					errMsg)
			})
		}
	}
}

func (suite *TransferTxApiSuite) Test_TransferTxApiMaxAmountAndFeePositive() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		n := transfer_utilities.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		itxID := issue_utilities.IssueAssetAmount(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultSenderNotMiner, utl.MaxAmount)
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, v, utl.TestChainID, itxID, utl.DefaultSenderNotMiner, n)
		tdmatrix := testdata.GetTransferMaxAmountPositive(&suite.BaseSuite, itxID, n)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, errMsg)
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					errMsg)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					errMsg)
			})
		}
	}
}

func (suite *TransferTxApiSuite) Test_TransferTxApiNegative() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetTransferNegativeData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)

			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID

				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					errMsg)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					errMsg)

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, errMsg)

			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferTxApiSuite) Test_TransferTxApiChainIDNegative() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetTransferChainIDChangedNegativeData(&suite.BaseSuite, itx.TxID)

		if v > 2 {
			maps.Copy(tdmatrix, testdata.GetTransferChainIDDataNegative(&suite.BaseSuite, itx.TxID))
		}

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				initBalanceWavesGoSender, initBalanceWavesScalaSender := utl.GetAvailableBalanceInWaves(
					&suite.BaseSuite, td.Sender.Address)
				initBalanceAssetGoSender, initBalanceAssetScalaSender := utl.GetAssetBalance(
					&suite.BaseSuite, td.Sender.Address, td.Asset.ID)

				tx := transfer_utilities.TransferBroadcastWithTestData(&suite.BaseSuite, td, v, !waitForTx)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID

				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala,
					errMsg)

				actualDiffBalanceWavesGoSender, actualDiffBalanceWavesScalaSender := utl.GetActualDiffBalanceInWaves(
					&suite.BaseSuite, td.Sender.Address, initBalanceWavesGoSender, initBalanceWavesScalaSender)

				actuallDiffBalanceAssetGoSender, actualDiffBalanceAssetScalaSender := utl.GetActualDiffBalanceInAssets(
					&suite.BaseSuite, td.Sender.Address, td.Asset.ID, initBalanceAssetGoSender, initBalanceAssetScalaSender)

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance,
					actualDiffBalanceWavesGoSender, actualDiffBalanceWavesScalaSender,
					errMsg)

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance,
					actuallDiffBalanceAssetGoSender, actualDiffBalanceAssetScalaSender,
					errMsg)

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxApiSuite))
}
