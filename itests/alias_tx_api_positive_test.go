package itests

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	alias_utl "github.com/wavesplatform/gowaves/itests/utilities/alias_utilities"
)

type AliasTxApiPositiveSuite struct {
	f.BaseSuite
}

func (suite *AliasTxApiPositiveSuite) Test_AliasTxApiPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", v, tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", v)
			})
		}
	}
}

func (suite *AliasTxApiPositiveSuite) Test_AliasTxApiMaxValuesPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		n, _ := utl.AddNewAccount(&suite.BaseSuite, testdata.TestChainID)
		utl.TransferFunds(&suite.BaseSuite, testdata.TestChainID, 5, n, 1000_00000000)
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, n)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", v, tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", v)
			})
		}
	}
}

func TestAliasTxApiPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxApiPositiveSuite))
}
