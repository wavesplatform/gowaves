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

type IssueTxApiSuite struct {
	f.BaseSuite
}

func (suite *IssueTxApiSuite) Test_IssueTxApiPositive() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Minute
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			tx, _, actualDiffBalanceInWaves := issue_utilities.BroadcastIssueTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout)
			utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", i)

			actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, tx.TxID)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i, tx.TxID.String())
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name, "version", i)
		}
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiWithSameDataPositive() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Minute
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			for j := 0; j < 2; j++ {
				tx, _, actualDiffBalanceInWaves := issue_utilities.SendIssueTxAndGetWavesBalances(&suite.BaseSuite,
					testdata.DataChangedTimestamp(&td), i, timeout)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", i)

				actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
					&suite.BaseSuite, td.Account.Address, tx.TxID)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i, tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo,
					actualAssetBalanceScala, name, "version", i)
			}
		}
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiNegative() {
	versions := testdata.GetVersions()
	timeout := 3 * time.Second
	txIds := make(map[string]*crypto.Digest)
	for _, i := range versions {
		tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			tx, _, actualDiffBalanceInWaves := issue_utilities.SendIssueTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout)

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
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestIssueTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxApiSuite))
}
