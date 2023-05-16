package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/set_asset_script_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type SetAssetScriptSuite struct {
	f.BaseSuite
}

func (suite *SetAssetScriptSuite) Test_SetAssetScriptPositive() {
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smartAsset, v, waitForTx)
		tdmatrix := testdata.GetSetAssetScriptPositiveData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := set_asset_script_utilities.SendSetAssetScriptTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

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

func (suite *SetAssetScriptSuite) Test_SetAssetScriptNegative() {
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smartAsset, v, waitForTx)
		tdmatrix := testdata.GetSetAssetScriptNegativeData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := set_asset_script_utilities.SendSetAssetScriptTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
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

func (suite *SetAssetScriptSuite) Test_SetScriptForNotScriptedAssetNegative() {
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		asset := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, asset, v, waitForTx)
		name := "Set script for not scripted asset"
		td := testdata.GetSimpleSmartAssetNegativeData(&suite.BaseSuite, itx.TxID)
		suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := set_asset_script_utilities.SendSetAssetScriptTxAndGetBalances(
				&suite.BaseSuite, td, v, !waitForTx)
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

func TestSetAssetScriptSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SetAssetScriptSuite))
}
