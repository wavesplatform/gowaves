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
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueSmartAssetAPIPositiveSuite struct {
	f.BaseSuite
}

func (suite *IssueSmartAssetAPIPositiveSuite) Test_IssueSmartAssetAPIPositive() {
	versions := issue.GetVersionsSmartAsset(&suite.BaseSuite)
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					issue.BroadcastIssueTxAndGetBalances(&suite.BaseSuite, td, v, true)
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Issue smart asset tx: %s", caseName, tx.TxID.String())
				issue.SmartAssetPositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails.AssetInfoGo.Script.ScriptBytes, assetDetails.AssetInfoScala.Script.ScriptBytes, errMsg)
			})
		}
	}
}

func TestIssueSmartAssetAPIPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueSmartAssetAPIPositiveSuite))
}

type IssueSmartAssetAPINegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *IssueSmartAssetAPINegativeSuite) Test_IssueSmartAssetAPINegative() {
	utl.WaitForHeight(&suite.BaseSuite, utl.DefaultSponsorshipActivationHeight)
	versions := issue.GetVersionsSmartAsset(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeAssetScriptData(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					issue.BroadcastIssueTxAndGetBalances(&suite.BaseSuite, td, v, false)
				txIds[name] = &tx.TxID
				errMsg := fmt.Sprintf("Case: %s; Broadcast Issue smart asset tx: %s", caseName, tx.TxID.String())
				issue.SmartAssetNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueSmartAssetAPINegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueSmartAssetAPINegativeSuite))
}
