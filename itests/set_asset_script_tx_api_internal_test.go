//go:build !smoke

package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/setassetscript"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type SetAssetScriptAPIPositiveSuite struct {
	f.BaseSuite
}

func (suite *SetAssetScriptAPIPositiveSuite) Test_SetAssetScriptAPIPositive() {
	versions := setassetscript.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue.BroadcastWithTestData(&suite.BaseSuite, smartAsset, v, true)
		tdmatrix := testdata.GetSetAssetScriptPositiveData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					setassetscript.BroadcastSetAssetScriptTxAndGetBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Set Asset Script tx: %s", caseName, tx.TxID.String())
				setassetscript.APIPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func TestSetAssetScriptAPIPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SetAssetScriptAPIPositiveSuite))
}

type SetAssetScriptAPINegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *SetAssetScriptAPINegativeSuite) Test_SetAssetScriptAPINegative() {
	utl.WaitForHeight(&suite.BaseSuite, utl.DefaultSponsorshipActivationHeight,
		config.WaitWithTimeoutInBlocks(utl.DefaultSponsorshipActivationHeight))
	versions := setassetscript.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue.BroadcastWithTestData(&suite.BaseSuite, smartAsset, v, true)
		tdmatrix := testdata.GetSetAssetScriptNegativeData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					setassetscript.BroadcastSetAssetScriptTxAndGetBalances(
						&suite.BaseSuite, td, v, false)
				txIds[name] = &tx.TxID
				errMsg := fmt.Sprintf("Case: %s; Broadcast Set Asset Script tx: %s", caseName, tx.TxID.String())
				setassetscript.APINegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SetAssetScriptAPINegativeSuite) Test_SetScriptForNotScriptedAssetAPINegative() {
	utl.WaitForHeight(&suite.BaseSuite, utl.DefaultSponsorshipActivationHeight,
		config.WaitWithTimeoutInBlocks(utl.DefaultSponsorshipActivationHeight))
	versions := setassetscript.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		asset := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.BroadcastWithTestData(&suite.BaseSuite, asset, v, true)
		name := "Set script for not scripted asset"
		td := testdata.GetSimpleSmartAssetNegativeData(&suite.BaseSuite, itx.TxID)
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				setassetscript.BroadcastSetAssetScriptTxAndGetBalances(
					&suite.BaseSuite, td, v, false)
			txIds[name] = &tx.TxID
			errMsg := fmt.Sprintf("Case: %s; Broadcast Set Asset Script tx: %s", caseName, tx.TxID.String())
			setassetscript.APINegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestSetAssetScriptAPINegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SetAssetScriptAPINegativeSuite))
}
