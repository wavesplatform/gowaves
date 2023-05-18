package itests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/set_asset_script_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type SetAssetScriptApiSuite struct {
	f.BaseSuite
}

func (suite *SetAssetScriptApiSuite) Test_SetAssetScriptApiPositive() {
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, smartAsset, v, waitForTx)
		tdmatrix := testdata.GetSetAssetScriptPositiveData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := set_asset_script_utilities.BroadcastSetAssetScriptTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, utl.GetTestcaseNameWithVersion(name, v))
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Set Asset Script: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *SetAssetScriptApiSuite) Test_SetAssetScriptApiNegative() {
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, smartAsset, v, waitForTx)
		tdmatrix := testdata.GetSetAssetScriptNegativeData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := set_asset_script_utilities.BroadcastSetAssetScriptTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)

				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx,
					utl.GetTestcaseNameWithVersion(name, v))
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, utl.GetTestcaseNameWithVersion(name, v))

				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
					tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SetAssetScriptApiSuite) Test_SetScriptForNotScriptedAssetApiNegative() {
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		asset := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, asset, v, waitForTx)
		name := "Set script for not scripted asset"
		td := testdata.GetSimpleSmartAssetNegativeData(&suite.BaseSuite, itx.TxID)
		suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := set_asset_script_utilities.BroadcastSetAssetScriptTxAndGetBalances(
				&suite.BaseSuite, td, v, !waitForTx)

			utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx,
				utl.GetTestcaseNameWithVersion(name, v))
			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
				tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, utl.GetTestcaseNameWithVersion(name, v))

			txIds[name] = &tx.TxID

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
				tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
				actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestSetAssetScriptApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SetAssetScriptApiSuite))
}
