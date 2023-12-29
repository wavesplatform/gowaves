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

type IssueSmartAssetApiSuite struct {
	f.BaseSuite
}

func issueSmartAssetPositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	actualScriptGo, actualScriptScala []byte, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
	utl.AssetScriptCheck(t, td.Script, actualScriptGo, actualScriptScala, errMsg)
}

func issueSmartAssetNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
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

func (suite *IssueSmartAssetApiSuite) Test_IssueSmartAssetApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					issue_utilities.BroadcastIssueTxAndGetBalances(&suite.BaseSuite, td, v, true)
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
				errMsg := caseName + "Broadcast Issue smart asset tx:" + tx.TxID.String()
				issueSmartAssetPositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails.AssetInfoGo.Script.ScriptBytes, assetDetails.AssetInfoScala.Script.ScriptBytes, errMsg)
			})
		}
	}
}

func (suite *IssueSmartAssetApiSuite) Test_IssueSmartAssetApiNegative() {
	utl.SkipLongTest(suite.T())
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					issue_utilities.BroadcastIssueTxAndGetBalances(&suite.BaseSuite, td, v, false)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Broadcast Issue smart asset tx:" + tx.TxID.String()
				issueSmartAssetNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *IssueSmartAssetApiSuite) Test_IssueSmartAssetApiSmokePositive() {
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetPositiveAssetScriptData(&suite.BaseSuite))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(
				&suite.BaseSuite, td, randV, true)
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
			errMsg := caseName + "Broadcast Issue smart asset tx:" + tx.TxID.String()
			issueSmartAssetPositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails.AssetInfoGo.Script.ScriptBytes, assetDetails.AssetInfoScala.Script.ScriptBytes, errMsg)
		})
	}
}

func (suite *IssueSmartAssetApiSuite) Test_IssueSmartAssetApiSmokeNegative() {
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetNegativeAssetScriptData(&suite.BaseSuite))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(
				&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Broadcast Issue smart asset tx:" + tx.TxID.String()
			issueSmartAssetNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueSmartAssetApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueSmartAssetApiSuite))
}
