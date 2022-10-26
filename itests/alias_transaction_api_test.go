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

type AliasTxApiSuite struct {
	f.BaseSuite
}

func (suite *AliasTxSuite) Test_AliasTxApiPositive() {
	versions := testdata.GetVersions(&suite.BaseSuite)
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			brdCstTx, errWtGo, errWtScala := alias_utl.AliasBroadcast(&suite.BaseSuite, td, i, timeout)

			utl.StatusCodesCheck(suite.T(), brdCstTx, http.StatusOK, http.StatusOK, name, "version: ", i)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			utl.ExistenceTxInfoCheck(suite.T(), errWtGo, errWtScala, name, "version:", i, brdCstTx.TxID.String())
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala,
				name, "version:", i)
		}
	}
}

func TestAliasTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxApiSuite))
}
