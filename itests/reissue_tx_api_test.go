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
	"github.com/wavesplatform/gowaves/itests/utilities/reissue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"golang.org/x/exp/maps"
)

type ReissueTxApiSuite struct {
	f.BaseSuite
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiPositive() {
	versions := testdata.GetVersions()
	timeout := 30 * time.Second
	positive := true
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["reissuable"], v, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, v, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "Reissue: "+tx.TxID.String(), "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", v)
			})
		}
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiMaxQuantityPositive() {
	versions := testdata.GetVersions()
	timeout := 30 * time.Second
	positive := true
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["reissuable"], v, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetReissueMaxQuantityValue(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, v, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "Reissue: "+tx.TxID.String(), "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", v)
			})
		}
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueNotReissuableApiNegative() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Second
	positive := true
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["reissuable"], v, 15*timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetNotReissuableTestData(&suite.BaseSuite, itx.TxID)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				//first tx should be successful
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.BroadcastReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, 15*timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "Reissue: "+tx.TxID.String(), "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.Positive.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.Positive.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", v)

				//second reissue tx should be failed because of reissuable=false
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset = reissue_utilities.BroadcastReissueTxAndGetBalances(
					&suite.BaseSuite, testdata.ReissueDataChangedTimestamp(&td), v, timeout, !positive)
				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, name, "version", v)
				utl.ErrorMessageCheck(suite.T(), td.Expected.Negative.ErrBrdCstGoMsg, td.Expected.Negative.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, name, "version", v)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.Negative.ErrGoMsg, td.Expected.Negative.ErrScalaMsg, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.Negative.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.Negative.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", v)
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 30*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiNFTNegative() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Second
	positive := true
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["NFT"], v, 15*timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetReissueNFTData(&suite.BaseSuite, itx.TxID)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.BroadcastReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, !positive)
				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, name, "version", v)
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, name, "version", v)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", v)
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 30*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiNegative() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Second
	positive := true
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["reissuable"], v, 15*timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		//TODO (ipereiaslavskaia) For v1 of reissue tx negative cases for chainID will be ignored
		if v >= 2 {
			maps.Copy(tdmatrix, testdata.GetReissueChainIDNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
		}
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.BroadcastReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, !positive)
				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, name, "version", v)
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, name, "version", v)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", v)
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 30*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestReissueTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxApiSuite))
}
