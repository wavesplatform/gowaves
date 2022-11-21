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
)

type ReissueTxApiSuite struct {
	f.BaseSuite
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiPositive() {
	versions := testdata.GetVersions()
	timeout := 45 * time.Second
	for _, i := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itxID, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["reissuable"], i, timeout)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itxID.String(), "Version: ", i)
		tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itxID)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID)

			brdCstTx, errWtGo, errWtScala := reissue_utilities.ReissueBroadcast(&suite.BaseSuite, td, i, timeout)

			utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, brdCstTx, name, "version", i)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID)
			actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
			actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

			utl.ExistenceTxInfoCheck(suite.T(), errWtGo, errWtScala, name, "Reissue: "+brdCstTx.TxID.String(), "Version: ", i)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo,
				actualDiffBalanceInWavesScala, name, "Version: ", i)
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo,
				actualDiffAssetBalanceScala, name, "Version: ", i)
		}
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiMaxQuantityPositive() {
	versions := testdata.GetVersions()
	timeout := 45 * time.Second
	for _, i := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itxID, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["reissuable"], i, timeout)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itxID.String(), "Version", i)
		tdmatrix := testdata.GetReissueMaxQuantityValue(&suite.BaseSuite, itxID)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID)

			brdCstTx, errWtGo, errWtScala := reissue_utilities.ReissueBroadcast(&suite.BaseSuite, td, i, timeout)

			utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, brdCstTx, name, "version", i)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID)
			actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
			actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

			utl.ExistenceTxInfoCheck(suite.T(), errWtGo, errWtScala, name, "Reissue: "+brdCstTx.TxID.String(), "Version: ", i)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo,
				actualDiffBalanceInWavesScala, name, "Version: ", i)
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo,
				actualDiffAssetBalanceScala, name, "Version: ", i)
		}
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiNFTNegative() {
	versions := testdata.GetVersions()
	timeout := 15 * time.Second
	for _, i := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itxID, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["NFT"], i, timeout)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itxID.String(), "version", i)
		tdmatrix := testdata.GetReissueNFTData(&suite.BaseSuite, itxID)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID)

			brdCstTx, errWtGo, errWtScala := reissue_utilities.ReissueBroadcast(&suite.BaseSuite, td, i, timeout)
			utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, brdCstTx, name, "version", i)
			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
				brdCstTx.ErrorBrdCstGo, brdCstTx.ErrorBrdCstScala, name, "version", i)
			txIds[name] = &brdCstTx.TxID

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID)
			actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
			actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, errWtGo, errWtScala)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo,
				actualDiffBalanceInWavesScala, name, "Version: ", i)
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo,
				actualDiffAssetBalanceScala, name, "Version: ", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiNegative() {
	versions := testdata.GetVersions()
	timeout := 5 * time.Second
	for _, i := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itxID, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["reissuable"], i, 9*timeout)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itxID.String(), "Version: ", i)
		tdmatrix := testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, itxID)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID)

			brdCstTx, errWtGo, errWtScala := reissue_utilities.ReissueBroadcast(&suite.BaseSuite, td, i, timeout)
			utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, brdCstTx, name, "version", i)
			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
				brdCstTx.ErrorBrdCstGo, brdCstTx.ErrorBrdCstScala, name, "version", i)
			txIds[name] = &brdCstTx.TxID

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID)
			actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
			actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, errWtGo, errWtScala, name, "Version: ", i)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo,
				actualDiffBalanceInWavesScala, name, "Version: ", i)
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo,
				actualDiffAssetBalanceScala, name, "Version: ", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestReissueTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxApiSuite))
}
