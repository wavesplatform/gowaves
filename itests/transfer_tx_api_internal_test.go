package itests

import (
	"math/rand"
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

func transferPositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, errMsg)
}

func transferNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesNegative],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, errMsg)
}

func transferBaseNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *TransferTxApiSuite) Test_TransferTxApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		alias := utl.RandStringBytes(15, testdata.AliasSymbolSet)
		alias_utilities.SetAliasToAccountByAPI(&suite.BaseSuite, v, utl.TestChainID, alias,
			utl.DefaultRecipientNotMiner)
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, alias)
		if v <= 2 {
			maps.Copy(tdmatrix, testdata.GetTransferChainIDDataBinaryVersions(&suite.BaseSuite, itx.TxID))
		}
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, true)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
				transferPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxApiSuite) Test_TransferSmartAssetApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	saversions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	name := "Check transfer smart asset"
	for _, v := range versions {
		for _, sav := range saversions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, smart, sav, true)
			td := testdata.GetCommonTransferData(&suite.BaseSuite, &itx.TxID).Smart
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, true)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
				transferPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxApiSuite) Test_TransferTxApiMaxAmountAndFeePositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
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
					&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
				transferPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxApiSuite) Test_TransferTxApiNegative() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetTransferNegativeData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, false)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				transferNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferTxApiSuite) Test_TransferTxApiChainIDNegative() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
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
				tx := transfer_utilities.TransferBroadcastWithTestData(&suite.BaseSuite, td, v, false)
				errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				actualDiffBalanceWavesSender := utl.GetActualDiffBalanceInWaves(&suite.BaseSuite, td.Sender.Address,
					initBalanceWavesGoSender, initBalanceWavesScalaSender)
				actualDiffBalanceAssetSender := utl.GetActualDiffBalanceInAssets(&suite.BaseSuite, td.Sender.Address,
					td.Asset.ID, initBalanceAssetGoSender, initBalanceAssetScalaSender)
				transferBaseNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceWavesSender,
					actualDiffBalanceAssetSender, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferTxApiSuite) Test_TransferTxApiSmokePositive() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	alias := utl.RandStringBytes(15, testdata.AliasSymbolSet)
	alias_utilities.SetAliasToAccountByAPI(&suite.BaseSuite, randV, utl.TestChainID, alias,
		utl.DefaultRecipientNotMiner)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, alias))
	if randV <= 2 {
		maps.Copy(tdmatrix, utl.GetRandomValueFromMap(testdata.GetTransferChainIDDataBinaryVersions(
			&suite.BaseSuite, itx.TxID)))
	}
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), randV, true)
			errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
			transferPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *TransferTxApiSuite) Test_TransferTxApiSmokeNegative() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetTransferNegativeData(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td, randV, false)
			errMsg := caseName + "Broadcast Transfer tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			transferNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxApiSuite))
}
