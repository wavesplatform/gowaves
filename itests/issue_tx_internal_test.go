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

type IssueTxPositiveSuite struct {
	f.BaseSuite
}

func (suite *IssueTxPositiveSuite) Test_IssueTxPositive() {
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

func (suite *IssueTxPositiveSuite) Test_IssueTxWithSameDataPositive() {
	versions := issue.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				for range 2 {
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

func TestIssueTxPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxPositiveSuite))
}

type IssueTxNegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *IssueTxNegativeSuite) Test_IssueTxNegative() {
	utl.WaitForHeight(&suite.BaseSuite, utl.DefaultSponsorshipActivationHeight,
		config.WaitWithTimeoutInBlocks(utl.DefaultSponsorshipActivationHeight))
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

func TestIssueTxNegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxNegativeSuite))
}
