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
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueTxApiNegativeSuite struct {
	f.BaseSuite
}

func (suite *IssueTxApiNegativeSuite) Test_IssueTxApiNegative() {
	versions := testdata.GetVersions()
	positive := false
	timeout := 1 * time.Second
	txIds := make(map[string]*crypto.Digest)
	for _, i := range versions {
		tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := issue_utilities.BroadcastIssueTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)

				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, name, "version", i)
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, name, "version", i)
				txIds[name] = &tx.TxID

				actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
					&suite.BaseSuite, td.Account.Address, tx.TxID)

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i)
				utl.WavesDiffBalanceCheck(
					suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name, "version", i)
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 30*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestIssueTxApiNegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxApiNegativeSuite))
}
