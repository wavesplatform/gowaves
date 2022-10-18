package itests

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueTxApiSuite struct {
	issue_utilities.CommonIssueTxSuite
}

func (suite *IssueTxApiSuite) Test_IssueTxApiPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)

		brdCstTx, errWtGo, errWtScala := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)

		utl.StatusCodesCheck(suite.T(), brdCstTx, http.StatusOK, http.StatusOK, name)

		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

		actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, brdCstTx.TxID.Bytes())

		utl.ExistenceTxInfoCheck(suite.T(), errWtGo, errWtScala, name)
		utl.WavesDiffBalanceCheck(
			suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name)
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiWithSameDataPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)

		brdCstTx1, errWtGo1, errWtScala1 := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)
		brdCstTx2, errWtGo2, errWtScala2 := issue_utilities.IssueBroadcast(
			&suite.CommonIssueTxSuite, testdata.DataChangedTimestamp(&td), timeout)

		utl.StatusCodesCheck(suite.T(), brdCstTx1, http.StatusOK, http.StatusOK, name)
		utl.StatusCodesCheck(suite.T(), brdCstTx2, http.StatusOK, http.StatusOK, name)

		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

		actualAsset1BalanceGo, actualAsset1BalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, brdCstTx1.TxID.Bytes())
		actualAsset2BalanceGo, actualAsset2BalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, brdCstTx2.TxID.Bytes())
		//Since the issue transaction is called twice, the expected balance difference also is doubled.
		expectedDiffBalanceInWaves := 2 * td.Expected.WavesDiffBalance

		utl.ExistenceTxInfoCheck(suite.T(), errWtGo1, errWtScala1, name)
		utl.ExistenceTxInfoCheck(suite.T(), errWtGo2, errWtScala2, name)
		utl.WavesDiffBalanceCheck(
			suite.T(), expectedDiffBalanceInWaves, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAsset1BalanceGo, actualAsset1BalanceScala, name)
		utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAsset2BalanceGo, actualAsset2BalanceScala, name)
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiNegative() {
	tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
	timeout := 3 * time.Second
	txIds := make(map[string]*crypto.Digest)

	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite,
			td.Account.Address)

		brdCstTx, errWtGo, errWtScala := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)

		utl.StatusCodesCheck(suite.T(), brdCstTx, http.StatusInternalServerError, http.StatusBadRequest, name)
		utl.ErrorMessageCheck(
			suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
			brdCstTx.ErrorBrdCstGo, brdCstTx.ErrorBrdCstScala, name)

		txIds[name] = &brdCstTx.TxID

		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, brdCstTx.TxID.Bytes())

		utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, errWtGo, errWtScala, name)
		utl.WavesDiffBalanceCheck(
			suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
		utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala)
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxApiSuite))
}
