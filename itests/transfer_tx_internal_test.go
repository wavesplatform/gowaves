//go:build !smoke

package itests

import (
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/alias"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type TransferTxSuite struct {
	f.BaseSuite
}

func (suite *TransferTxSuite) Test_TransferTxPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		aliasStr := utl.RandStringBytes(15, testdata.AliasSymbolSet)
		alias.SetAliasToAccount(&suite.BaseSuite, v, utl.TestChainID, aliasStr, utl.DefaultRecipientNotMiner)
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.SendWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, aliasStr)
		if v <= 2 {
			maps.Copy(tdmatrix, testdata.GetTransferChainIDDataBinaryVersions(&suite.BaseSuite, itx.TxID))
		}
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, true)
				errMsg := fmt.Sprintf("Case: %s; Transfer tx: %s", caseName, tx.TxID.String())
				transfer.PositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferSmartAssetPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	saversions := issue.GetVersionsSmartAsset(&suite.BaseSuite)
	name := "Check transfer smart asset"
	for _, v := range versions {
		for _, sav := range saversions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue.SendWithTestData(&suite.BaseSuite, smart, sav, true)
			td := testdata.GetCommonTransferData(&suite.BaseSuite, &itx.TxID).Smart
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, true)
				errMsg := fmt.Sprintf("Case: %s; Transfer tx: %s", caseName, tx.TxID.String())
				transfer.PositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferTxMaxAmountAndFeePositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		n := transfer.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		itxID := issue.IssuedAssetAmount(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultSenderNotMiner, utl.MaxAmount)
		transfer.TransferringAssetAmount(&suite.BaseSuite, v, utl.TestChainID, itxID,
			utl.DefaultSenderNotMiner, n)
		tdmatrix := testdata.GetTransferMaxAmountPositive(&suite.BaseSuite, itxID, n)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Transfer tx: %s", caseName, tx.TxID.String())
				transfer.PositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferTxSuite) Test_TransferTxNegative() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.SendWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetTransferNegativeData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, false)
				errMsg := fmt.Sprintf("Case: %s; Transfer tx: %s", caseName, tx.TxID.String())
				txIds[name] = &tx.TxID
				transfer.NegativeChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferTxSuite) Test_TransferTxChainIDNegative() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.SendWithTestData(&suite.BaseSuite, reissuable, v, true)
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
				tx := transfer.SendWithTestData(&suite.BaseSuite, td, v, false)
				errMsg := fmt.Sprintf("Case: %s; Transfer tx: %s", caseName, tx.TxID.String())
				txIds[name] = &tx.TxID
				actualDiffBalanceWavesSender := utl.GetActualDiffBalanceInWaves(&suite.BaseSuite, td.Sender.Address,
					initBalanceWavesGoSender, initBalanceWavesScalaSender)
				actuallDiffBalanceAssetSender := utl.GetActualDiffBalanceInAssets(&suite.BaseSuite, td.Sender.Address,
					td.Asset.ID, initBalanceAssetGoSender, initBalanceAssetScalaSender)
				transfer.BaseNegativeChecks(suite.T(), tx, td, actualDiffBalanceWavesSender,
					actuallDiffBalanceAssetSender, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxSuite))
}
