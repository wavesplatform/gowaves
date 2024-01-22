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

type IssueTxSuite struct {
	f.BaseSuite
}

func (suite *IssueTxSuite) Test_IssueTxPositive() {
	versions := issue.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Issue tx: %s", caseName, tx.TxID.String())
				issue.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *IssueTxSuite) Test_IssueTxWithSameDataPositive() {
	versions := issue.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				for j := 0; j < 2; j++ {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						issue.SendIssueTxAndGetBalances(&suite.BaseSuite,
							testdata.DataChangedTimestamp(&td), v, true)
					errMsg := fmt.Sprintf("Case: %s; Issue tx: %s", caseName, tx.TxID.String())
					issue.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
				}
			})
		}
	}
}

func (suite *IssueTxSuite) Test_IssueTxNegative() {
	versions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue.SendIssueTxAndGetBalances(
					&suite.BaseSuite, td, v, false)
				txIds[name] = &tx.TxID
				errMsg := fmt.Sprintf("Case: %s; Issue tx: %s", caseName, tx.TxID.String())
				issue.NegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxSuite))
}
