package itests

import (
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

type ReissueTxSuite struct {
	f.BaseSuite
}

func (suite *ReissueTxSuite) Test_ReissuePositive() {
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
				&suite.BaseSuite, td.Account.Address, itxID.Bytes())

			rtxID, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, i, timeout)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID.Bytes())
			actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
			actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

			utl.ExistenceTxInfoCheck(suite.T(), rErrGo, rErrScala, name, "Reissue: "+rtxID.String(), "Version: ", i)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo,
				actualDiffBalanceInWavesScala, name, "Version: ", i)
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo,
				actualDiffAssetBalanceScala, name, "Version: ", i)
		}
	}
}

func (suite *ReissueTxSuite) Test_ReissueMaxQuantityPositive() {
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
				&suite.BaseSuite, td.Account.Address, itxID.Bytes())

			rtxID, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, i, timeout)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID.Bytes())
			actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
			actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

			utl.ExistenceTxInfoCheck(suite.T(), rErrGo, rErrScala, name, "Reissue: "+rtxID.String(), "Version: ", i)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo,
				actualDiffBalanceInWavesScala, name, "Version: ", i)
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo,
				actualDiffAssetBalanceScala, name, "Version: ", i)
		}
	}
}

func (suite *ReissueTxSuite) Test_ReissueNFTNegative() {
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
				&suite.BaseSuite, td.Account.Address, itxID.Bytes())

			rtxID, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, i, timeout)
			txIds[name] = &rtxID

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID.Bytes())
			actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
			actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, rErrGo, rErrScala)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo,
				actualDiffBalanceInWavesScala, name, "Version: ", i)
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo,
				actualDiffAssetBalanceScala, name, "Version: ", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *ReissueTxSuite) Test_ReissueNegative() {
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
				&suite.BaseSuite, td.Account.Address, itxID.Bytes())
			rtxID, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, i, timeout)
			txIds[name] = &rtxID
			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, itxID.Bytes())
			actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
			actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, rErrGo, rErrScala, name, "Version: ", i)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo,
				actualDiffBalanceInWavesScala, name, "Version: ", i)
			utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo,
				actualDiffAssetBalanceScala, name, "Version: ", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestReissueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxSuite))
}
