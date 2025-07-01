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
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueSmartAssetPositiveSuite struct {
	f.BaseSuite
}

func (suite *IssueSmartAssetPositiveSuite) Test_IssueSmartAssetPositive() {
	versions := issue.GetVersionsSmartAsset(&suite.BaseSuite)
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
				errMsg := fmt.Sprintf("Case: %s; Issue smart asset tx: %s", caseName, tx.TxID.String())
				issue.SmartAssetPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails.AssetInfoGo.Script.ScriptBytes, assetDetails.AssetInfoScala.Script.ScriptBytes, errMsg)
			})
		}
	}
}

func TestIssueSmartAssetPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueSmartAssetPositiveSuite))
}

type IssueSmartAssetNegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *IssueSmartAssetNegativeSuite) Test_IssueSmartAssetNegative() {
	utl.WaitForHeight(&suite.BaseSuite, utl.DefaultSponsorshipActivationHeight,
		config.WaitWithTimeoutInBlocks(utl.DefaultSponsorshipActivationHeight))
	versions := issue.GetVersionsSmartAsset(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, false)
				txIds[name] = &tx.TxID
				errMsg := fmt.Sprintf("Case: %s; Issue smart asset tx: %s", caseName, tx.TxID.String())
				issue.SmartAssetNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueSmartAssetNegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueSmartAssetNegativeSuite))
}
