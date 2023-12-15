package itests

import (
	"math/rand"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueTxApiSuite struct {
	f.BaseSuite
}

func issuePositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func issueNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
		tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *IssueTxApiSuite) Test_IssueTxApiPositive() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Issue API Tx tests in short mode")
	}
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Broadcast Issue tx:" + tx.TxID.String()
				issuePositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiWithSameDataPositive() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Issue API Tx tests in short mode")
	}
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				for j := 0; j < 2; j++ {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(&suite.BaseSuite,
						testdata.DataChangedTimestamp(&td), v, waitForTx)
					errMsg := caseName + "Broadcast Issue tx:" + tx.TxID.String()
					issuePositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
				}
			})
		}
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiNegative() {
	if testing.Short() {
		suite.T().Skip("skipping long negative Issue API Tx tests in short mode")
	}
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Broadcast Issue tx:" + tx.TxID.String()
				issueNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *IssueTxApiSuite) Test_IssueTxApiSmokePositive() {
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetPositiveDataMatrix(&suite.BaseSuite))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(
				&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Broadcast Issue tx:" + tx.TxID.String()
			issuePositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiSmokeNegative() {
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetNegativeDataMatrix(&suite.BaseSuite))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(
				&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Broadcast Issue tx:" + tx.TxID.String()
			issueNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxApiSuite))
}
