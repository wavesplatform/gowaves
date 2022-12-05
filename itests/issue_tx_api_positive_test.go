package itests

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
)

type IssueTxApiPositiveSuite struct {
	f.BaseSuite
}

func (suite *IssueTxApiPositiveSuite) Test_IssueTxApiPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := issue_utilities.BroadcastIssueTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", i)

				actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
					&suite.BaseSuite, td.Account.Address, tx.TxID)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i, tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name, "version", i)
			})
		}
	}
}

func (suite *IssueTxApiPositiveSuite) Test_IssueTxApiWithSameDataPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				for j := 0; j < 2; j++ {
					tx, _, actualDiffBalanceInWaves := issue_utilities.BroadcastIssueTxAndGetWavesBalances(&suite.BaseSuite,
						testdata.DataChangedTimestamp(&td), i, timeout, positive)
					utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", i)

					actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
						&suite.BaseSuite, td.Account.Address, tx.TxID)

					utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i, tx.TxID.String())
					utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
						actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
					utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo,
						actualAssetBalanceScala, name, "version", i)
				}
			})
		}
	}
}

func TestIssueTxApiPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxApiPositiveSuite))
}
