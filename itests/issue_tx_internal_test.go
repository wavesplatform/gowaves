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

type IssueTxSuite struct {
	f.BaseSuite
}

func issuePositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func issueNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *IssueTxSuite) Test_IssueTxPositive() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Issue Tx tests in short mode")
	}
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Issue tx:" + tx.TxID.String()
				issuePositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *IssueTxSuite) Test_IssueTxWithSameDataPositive() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Issue Tx tests in short mode")
	}
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				for j := 0; j < 2; j++ {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(&suite.BaseSuite,
						testdata.DataChangedTimestamp(&td), v, waitForTx)
					errMsg := caseName + "Issue tx:" + tx.TxID.String()
					issuePositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
				}
			})
		}
	}
}

func (suite *IssueTxSuite) Test_IssueTxNegative() {
	if testing.Short() {
		suite.T().Skip("skipping long negative Issue Tx tests in short mode")
	}
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Issue tx:" + tx.TxID.String()
				issueNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *IssueTxSuite) Test_IssueTxSmokePositive() {
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetPositiveDataMatrix(&suite.BaseSuite))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
				&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Issue tx:" + tx.TxID.String()
			issuePositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *IssueTxSuite) Test_IssueTxSmokeNegative() {
	versions := issue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetNegativeDataMatrix(&suite.BaseSuite))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
				&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Issue tx:" + tx.TxID.String()
			issueNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxSuite))
}
