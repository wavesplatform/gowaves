//go:build !smoke

package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/setassetscript"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type SetAssetScriptPositiveSuite struct {
	f.BaseSuite
}

func (suite *SetAssetScriptPositiveSuite) Test_SetAssetScriptPositive() {
	versions := setassetscript.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue.SendWithTestData(&suite.BaseSuite, smartAsset, v, true)
		tdmatrix := testdata.GetSetAssetScriptPositiveData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					setassetscript.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Set Asset Script tx: %s", caseName, tx.TxID.String())
				setassetscript.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func TestSetAssetScriptPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SetAssetScriptPositiveSuite))
}

type SetAssetScriptNegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *SetAssetScriptNegativeSuite) Test_SetAssetScriptNegative() {
	utl.WaitForHeight(&suite.BaseSuite, utl.DefaultSponsorshipActivationHeight)
	versions := setassetscript.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue.SendWithTestData(&suite.BaseSuite, smartAsset, v, true)
		tdmatrix := testdata.GetSetAssetScriptNegativeData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					setassetscript.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, v, false)
				errMsg := fmt.Sprintf("Case: %s; Set Asset Script tx: %s", caseName, tx.TxID.String())
				txIds[name] = &tx.TxID
				setassetscript.NegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SetAssetScriptNegativeSuite) Test_SetScriptForNotScriptedAssetNegative() {
	utl.WaitForHeight(&suite.BaseSuite, utl.DefaultSponsorshipActivationHeight)
	versions := setassetscript.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		asset := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.SendWithTestData(&suite.BaseSuite, asset, v, true)
		name := "Set script for not scripted asset"
		td := testdata.GetSimpleSmartAssetNegativeData(&suite.BaseSuite, itx.TxID)
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				setassetscript.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, v, false)
			errMsg := fmt.Sprintf("Case: %s; Set Asset Script tx: %s", caseName, tx.TxID.String())
			txIds[name] = &tx.TxID
			setassetscript.NegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestSetAssetScriptNegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SetAssetScriptNegativeSuite))
}
