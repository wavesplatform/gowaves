package itests

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueSmartAssetSuite struct {
	f.BaseSuite
}

func issueSmartAssetPositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	actualScriptGo, actualScriptScala []byte, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
	utl.AssetScriptCheck(t, td.Script, actualScriptGo, actualScriptScala, errMsg)
}

func issueSmartAssetNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *IssueSmartAssetSuite) Test_IssueSmartAssetPositive() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Issue Smart Asset Tx tests in short mode")
	}
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
				errMsg := caseName + "Issue smart asset tx:" + tx.TxID.String()
				issueSmartAssetPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails.AssetInfoGo.Script.ScriptBytes, assetDetails.AssetInfoScala.Script.ScriptBytes, errMsg)
			})
		}
	}
}

func (suite *IssueSmartAssetSuite) Test_IssueSmartAssetNegative() {
	if testing.Short() {
		suite.T().Skip("skipping long negative Issue Smart Asset Tx tests in short mode")
	}
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Issue smart asset tx:" + tx.TxID.String()
				issueSmartAssetNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *IssueSmartAssetSuite) Test_IssueSmartAssetSmokePositive() {
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetPositiveAssetScriptData(&suite.BaseSuite))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
				&suite.BaseSuite, td, randV, true)
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
			errMsg := caseName + "Issue smart asset tx:" + tx.TxID.String()
			issueSmartAssetPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails.AssetInfoGo.Script.ScriptBytes, assetDetails.AssetInfoScala.Script.ScriptBytes, errMsg)
		})
	}
}

func (suite *IssueSmartAssetSuite) Test_IssueSmartAssetSmokeNegative() {
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetNegativeAssetScriptData(&suite.BaseSuite))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
				&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Issue smart asset tx:" + tx.TxID.String()
			issueSmartAssetNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueSmartAssetSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueSmartAssetSuite))
}
