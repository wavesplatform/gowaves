package itests

import (
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

func (suite *IssueSmartAssetApiSuite) Test_IssueSmartAssetApiPositive() {
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, utl.GetTestcaseNameWithVersion(name, v))
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
					utl.GetTestcaseNameWithVersion(name, v), tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
				utl.AssetScriptCheck(suite.T(), td.Script, assetDetailsGo.Script.ScriptBytes, assetDetailsScala.Script.ScriptBytes)
			})
		}
	}
}

func (suite *IssueSmartAssetApiSuite) Test_IssueSmartAssetApiNegative() {
	versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue_utilities.BroadcastIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)

				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx,
					utl.GetTestcaseNameWithVersion(name, v))
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, utl.GetTestcaseNameWithVersion(name, v))

				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
					tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestIssueSmartAssetApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueSmartAssetApiSuite))
}
