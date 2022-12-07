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
	positive := true
	for _, i := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueSend(&suite.BaseSuite, issuedata["reissuable"], i, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version: ", i)
		tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(&suite.BaseSuite, td, i, timeout, positive)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "Reissue: "+tx.TxID.String(), "Version: ", i)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
				actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", i)
		}
	}
}

func (suite *ReissueTxSuite) Test_ReissueMaxQuantityPositive() {
	versions := testdata.GetVersions()
	timeout := 45 * time.Second
	positive := true
	for _, i := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueSend(&suite.BaseSuite, issuedata["reissuable"], i, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version", i)
		tdmatrix := testdata.GetReissueMaxQuantityValue(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(&suite.BaseSuite, td, i, timeout, positive)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "Reissue: "+tx.TxID.String(), "Version: ", i)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
				actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", i)
		}
	}
}

func (suite *ReissueTxSuite) Test_ReissueNFTNegative() {
	versions := testdata.GetVersions()
	timeout := 15 * time.Second
	positive := false
	for _, i := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueSend(&suite.BaseSuite, issuedata["NFT"], i, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version", i)
		tdmatrix := testdata.GetReissueNFTData(&suite.BaseSuite, itx.TxID)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(&suite.BaseSuite, td, i, timeout, positive)
			txIds[name] = &tx.TxID

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
				actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *ReissueTxSuite) Test_ReissueNegative() {
	versions := testdata.GetVersions()
	timeout := 5 * time.Second
	positive := false
	for _, i := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueSend(&suite.BaseSuite, issuedata["reissuable"], i, 9*timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala, "Issue: "+itx.TxID.String(), "Version", i)
		tdmatrix := testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(&suite.BaseSuite, td, i, timeout, positive)
			txIds[name] = &tx.TxID

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, name, "Version: ", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
				actualDiffBalanceInAsset.BalanceInAssetScala, name, "Version: ", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestReissueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxSuite))
}
