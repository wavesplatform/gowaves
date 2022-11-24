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
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AliasTxApiSuite struct {
	f.BaseSuite
}

func (suite *AliasTxApiSuite) Test_AliasTxApiPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)
			utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", i)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i, tx.TxID.String())
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
		}
	}
}

func (suite *AliasTxApiSuite) Test_AliasTxApiMaxValuesPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, int(i+1))
		for name, td := range tdmatrix {
			tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)

			utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", i)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i, tx.TxID.String())
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
		}
	}
}

func (suite *AliasTxApiSuite) Test_AliasTxApiNegative() {
	versions := testdata.GetVersions()
	positive := false
	timeout := 5 * time.Second
	txIds := make(map[string]*crypto.Digest)
	for _, i := range versions {
		tdmatrix := testdata.GetAliasNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)

			utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, name, "version: ", i)
			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
				tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, name, "version", i)
			txIds[name] = &tx.TxID

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds, "Version: %#v", i)
	}
}

func TestAliasTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxApiSuite))
}
