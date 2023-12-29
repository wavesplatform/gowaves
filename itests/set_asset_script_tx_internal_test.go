package itests

import (
	"math/rand"
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

func setAssetScriptPositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SetAssetScriptTestData[testdata.SetAssetScriptExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func setAssetScriptNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SetAssetScriptTestData[testdata.SetAssetScriptExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *SetAssetScriptSuite) Test_SetAssetScriptPositive() {
	utl.SkipLongTest(suite.T())
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smartAsset, v, true)
		tdmatrix := testdata.GetSetAssetScriptPositiveData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					set_asset_script_utilities.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Set Asset Script tx: " + tx.TxID.String()
				setAssetScriptPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *SetAssetScriptSuite) Test_SetAssetScriptNegative() {
	utl.SkipLongTest(suite.T())
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smartAsset, v, true)
		tdmatrix := testdata.GetSetAssetScriptNegativeData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					set_asset_script_utilities.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, v, false)
				errMsg := caseName + "Set Asset Script tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				setAssetScriptNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SetAssetScriptSuite) Test_SetScriptForNotScriptedAssetNegative() {
	utl.SkipLongTest(suite.T())
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		asset := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, asset, v, true)
		name := "Set script for not scripted asset"
		td := testdata.GetSimpleSmartAssetNegativeData(&suite.BaseSuite, itx.TxID)
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				set_asset_script_utilities.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, v, false)
			errMsg := caseName + "Set Asset Script tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			setAssetScriptNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SetAssetScriptSuite) Test_SetAssetScriptSmokePositive() {
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smartAsset, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetSetAssetScriptPositiveData(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				set_asset_script_utilities.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Set Asset Script tx: " + tx.TxID.String()
			setAssetScriptPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SetAssetScriptSuite) Test_SetAssetScriptSmokeNegative() {
	versions := set_asset_script_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smartAsset, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetSetAssetScriptNegativeData(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				set_asset_script_utilities.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, randV, false)
			errMsg := caseName + "Set Asset Script tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			setAssetScriptNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestSetAssetScriptSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SetAssetScriptSuite))
}
