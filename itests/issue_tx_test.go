package itests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueTxSuite struct {
	f.BaseSuite
}

func (suite *IssueTxSuite) Test_IssueTxPositive() {
	versions := testdata.GetVersions()

	positive := true
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := issue_utilities.SendIssueTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)

				actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
					&suite.BaseSuite, td.Account.Address, tx.TxID)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", i, tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", i)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo,
					actualAssetBalanceScala, name, "version", i)
			})
		}
	}

}

func (suite *IssueTxSuite) Test_IssueTxWithSameDataPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				for j := 0; j < 2; j++ {
					tx, _, actualDiffBalanceInWaves := issue_utilities.SendIssueTxAndGetWavesBalances(&suite.BaseSuite,
						testdata.DataChangedTimestamp(&td), i, timeout, positive)

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

func (suite *IssueTxSuite) Test_IssueTxNegative() {
	versions := testdata.GetVersions()
	positive := false
	timeout := 1 * time.Second
	txIds := make(map[string]*crypto.Digest)
	for _, i := range versions {
		tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := issue_utilities.SendIssueTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)
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

func TestIssueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxSuite))
}
