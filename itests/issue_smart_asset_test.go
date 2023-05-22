package itests

import (
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

func (suite *IssueSmartAssetSuite) Test_IssueSmartAssetPositive() {
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Issue smart asset tx:" + tx.TxID.String()

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

				assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
				utl.AssetScriptCheck(suite.T(), td.Script, assetDetailsGo.Script.ScriptBytes, assetDetailsScala.Script.ScriptBytes,
					errMsg)
			})
		}
	}

}

func (suite *IssueSmartAssetSuite) Test_IssueSmartAssetNegative() {
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.SendIssueTxAndGetBalances(&suite.BaseSuite,
					td, v, !waitForTx)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Issue smart asset tx:" + tx.TxID.String()

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, errMsg)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueSmartAssetSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueSmartAssetSuite))
}
