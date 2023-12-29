package itests

import (
	"math/rand"
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

func transferPositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
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

func transferNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesNegative],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
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

func transferBaseNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *TransferTxSuite) Test_TransferTxPositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		alias := utl.RandStringBytes(15, testdata.AliasSymbolSet)
		alias_utilities.SetAliasToAccount(&suite.BaseSuite, v, utl.TestChainID, alias, utl.DefaultRecipientNotMiner)
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, alias)
		if v <= 2 {
			maps.Copy(tdmatrix, testdata.GetTransferChainIDDataBinaryVersions(&suite.BaseSuite, itx.TxID))
		}
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, true)
				errMsg := caseName + "Transfer tx: " + tx.TxID.String()
				transferPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferSmartAssetPositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	saversions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	name := "Check transfer smart asset"
	for _, v := range versions {
		for _, sav := range saversions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smart, sav, true)
			td := testdata.GetCommonTransferData(&suite.BaseSuite, &itx.TxID).Smart
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, true)
				errMsg := caseName + "Transfer tx: " + tx.TxID.String()
				transferPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferTxMaxAmountAndFeePositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		n := transfer_utilities.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		itxID := issue_utilities.IssueAssetAmount(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultSenderNotMiner, utl.MaxAmount)
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, v, utl.TestChainID, itxID,
			utl.DefaultSenderNotMiner, n)
		tdmatrix := testdata.GetTransferMaxAmountPositive(&suite.BaseSuite, itxID, n)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Transfer tx: " + tx.TxID.String()
				transferPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferTxNegative() {
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
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, false)
				errMsg := caseName + "Transfer tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				transferNegativeChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferTxSuite) Test_TransferTxChainIDNegative() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, true)
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
				tx := transfer_utilities.TransferSendWithTestData(&suite.BaseSuite, td, v, false)
				errMsg := caseName + "Transfer tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				actualDiffBalanceWavesSender := utl.GetActualDiffBalanceInWaves(&suite.BaseSuite, td.Sender.Address,
					initBalanceWavesGoSender, initBalanceWavesScalaSender)
				actuallDiffBalanceAssetSender := utl.GetActualDiffBalanceInAssets(&suite.BaseSuite, td.Sender.Address,
					td.Asset.ID, initBalanceAssetGoSender, initBalanceAssetScalaSender)
				transferBaseNegativeChecks(suite.T(), tx, td, actualDiffBalanceWavesSender,
					actuallDiffBalanceAssetSender, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferTxSuite) Test_TransferTxSmokePositive() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	alias := utl.RandStringBytes(15, testdata.AliasSymbolSet)
	alias_utilities.SetAliasToAccount(&suite.BaseSuite, randV, utl.TestChainID, alias, utl.DefaultRecipientNotMiner)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, alias))
	if randV <= 2 {
		maps.Copy(tdmatrix, utl.GetRandomValueFromMap(testdata.GetTransferChainIDDataBinaryVersions(
			&suite.BaseSuite, itx.TxID)))
	}
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
				&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), randV, true)
			errMsg := caseName + "Transfer tx: " + tx.TxID.String()
			transferPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *TransferTxSuite) Test_TransferTxSmokeNegative() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetTransferNegativeData(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
				&suite.BaseSuite, td, randV, false)
			errMsg := caseName + "Transfer tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			transferNegativeChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxSuite))
}
