package itests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/burn_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
)

type BurnTxSuite struct {
	f.BaseSuite
}

func (suite *BurnTxSuite) Test_BurnTxPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueSend(&suite.BaseSuite, issuedata["reissuable"], v, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala,
			"Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetBurnPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn_utilities.SendBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, positive)
				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "Burn: "+tx.TxID.String(), "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", v)
			})
		}
	}
}

func TestBurnTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BurnTxSuite))
}
