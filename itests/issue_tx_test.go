package itests

import (
	"testing"

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
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, _, actualDiffBalanceInWaves := issue_utilities.SendIssueTxAndGetWavesBalances(
					&suite.BaseSuite, td, v, waitForTx)

				actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
					&suite.BaseSuite, td.Account.Address, tx.TxID)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
					utl.GetTestcaseNameWithVersion(name, v), tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo,
					actualAssetBalanceScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}

}

func (suite *IssueTxSuite) Test_IssueTxWithSameDataPositive() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				for j := 0; j < 2; j++ {
					tx, _, actualDiffBalanceInWaves := issue_utilities.SendIssueTxAndGetWavesBalances(&suite.BaseSuite,
						testdata.DataChangedTimestamp(&td), v, waitForTx)

					actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
						&suite.BaseSuite, td.Account.Address, tx.TxID)

					utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
						utl.GetTestcaseNameWithVersion(name, v), tx.TxID.String())
					utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
						actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
					utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo,
						actualAssetBalanceScala, utl.GetTestcaseNameWithVersion(name, v))
				}
			})
		}
	}
}

func (suite *IssueTxSuite) Test_IssueTxNegative() {
	versions := utl.GetVersions()
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, _, actualDiffBalanceInWaves := issue_utilities.SendIssueTxAndGetWavesBalances(&suite.BaseSuite,
					td, v, !waitForTx)
				txIds[name] = &tx.TxID

				actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
					&suite.BaseSuite, td.Account.Address, tx.TxID)

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(
					suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo,
					actualAssetBalanceScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestIssueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxSuite))
}
