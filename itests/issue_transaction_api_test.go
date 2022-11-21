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

type IssueTxApiSuite struct {
	f.BaseSuite
}

func (suite *IssueTxApiSuite) Test_IssueTxApiPositive() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Minute
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			brdCstTx, errWtGo, errWtScala := issue_utilities.IssueBroadcast(&suite.BaseSuite, td, i, timeout)

			utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, brdCstTx, name, "version", i)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, brdCstTx.TxID)

			utl.ExistenceTxInfoCheck(suite.T(), errWtGo, errWtScala, name, "version", i, brdCstTx.TxID.String())
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name, "version", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name, "version", i)
		}
	}
}

/*func (suite *IssueTxApiSuite) Test_IssueTxApiWithSameDataPositive() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Minute
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			brdCstTx1, errWtGo1, errWtScala1 := issue_utilities.IssueBroadcast(&suite.BaseSuite, td, i, timeout)
			brdCstTx2, errWtGo2, errWtScala2 := issue_utilities.IssueBroadcast(
				&suite.BaseSuite, testdata.DataChangedTimestamp(&td), i, timeout)

			utl.StatusCodesCheck(suite.T(), brdCstTx1, http.StatusOK, http.StatusOK, name, "version", i)
			utl.StatusCodesCheck(suite.T(), brdCstTx2, http.StatusOK, http.StatusOK, name, "version", i)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			actualAsset1BalanceGo, actualAsset1BalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, brdCstTx1.TxID)
			actualAsset2BalanceGo, actualAsset2BalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, brdCstTx2.TxID)
			//Since the issue transaction is called twice, the expected balance difference also is doubled.
			expectedDiffBalanceInWaves := 2 * td.Expected.WavesDiffBalance

			utl.ExistenceTxInfoCheck(suite.T(), errWtGo1, errWtScala1, name, "version", i, brdCstTx1.TxID.String())
			utl.ExistenceTxInfoCheck(suite.T(), errWtGo2, errWtScala2, name, "version", i, brdCstTx2.TxID.String())
			utl.WavesDiffBalanceCheck(
				suite.T(), expectedDiffBalanceInWaves, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAsset1BalanceGo, actualAsset1BalanceScala, name, "version", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAsset2BalanceGo, actualAsset2BalanceScala, name, "version", i)
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
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite,
				td.Account.Address)

			brdCstTx, errWtGo, errWtScala := issue_utilities.IssueBroadcast(&suite.BaseSuite, td, i, timeout)

			utl.StatusCodesCheck(suite.T(), brdCstTx, http.StatusInternalServerError, http.StatusBadRequest, name, "version", i)
			utl.ErrorMessageCheck(
				suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
				brdCstTx.ErrorBrdCstGo, brdCstTx.ErrorBrdCstScala, name, "version", i)

			txIds[name] = &brdCstTx.TxID

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, brdCstTx.TxID)

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, errWtGo, errWtScala, name, "version", i)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name, "version", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name, "version", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}*/

func TestIssueTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxApiSuite))
}
