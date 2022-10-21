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
	issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
	timeout := 30 * time.Second
	itx, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["reissuable"], timeout)
	utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itx.ID.String())

	tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, *itx.ID)
	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())

		rtx, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, timeout)

		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
		actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

		utl.ExistenceTxInfoCheck(suite.T(), rErrGo, rErrScala, name, rtx.ID.String())
		utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo, actualDiffAssetBalanceScala, name)
	}
}

func (suite *ReissueTxSuite) Test_ReissueMaxQuantityPositive() {
	issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
	timeout := 15 * time.Second
	itx, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["reissuable"], timeout)
	utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itx.ID.String())

	tdmatrix := testdata.GetReissueMaxQuantityValue(&suite.BaseSuite, *itx.ID)
	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())

		rtx, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, timeout)

		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
		actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

		utl.ExistenceTxInfoCheck(suite.T(), rErrGo, rErrScala, name, rtx.ID.String())
		utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo, actualDiffAssetBalanceScala, name)
	}
}

func (suite *ReissueTxSuite) Test_ReissueNFTNegative() {
	issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
	timeout := 15 * time.Second
	itx, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["NFT"], timeout)
	utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itx.ID.String())

	tdmatrix := testdata.GetReissueNFTData(&suite.BaseSuite, *itx.ID)
	txIds := make(map[string]*crypto.Digest)
	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())

		rtx, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, timeout)
		txIds[name] = rtx.ID

		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
		actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

		utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, rErrGo, rErrScala)
		utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo, actualDiffAssetBalanceScala, name)
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, timeout, timeout)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxSuite) Test_ReissueNegative() {
	issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
	timeout := 3 * time.Second
	itx, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["reissuable"], 5*timeout)
	utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itx.ID.String())

	tdmatrix := testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, *itx.ID)
	txIds := make(map[string]*crypto.Digest)
	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		rtx, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, timeout)
		txIds[name] = rtx.ID
		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		actualDiffAssetBalanceGo := currentAssetBalanceGo - initAssetBalanceGo
		actualDiffAssetBalanceScala := currentAssetBalanceScala - initAssetBalanceScala

		utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, rErrGo, rErrScala)
		utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceGo, actualDiffAssetBalanceScala, name)
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 5*timeout, timeout)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestReissueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxSuite))
}
