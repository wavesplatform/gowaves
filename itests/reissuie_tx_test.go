package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reissue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"golang.org/x/exp/maps"
)

type ReissueTxSuite struct {
	f.BaseSuite
}

func (suite *ReissueTxSuite) Test_ReissuePositive() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Reissue: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *ReissueTxSuite) Test_ReissueMaxQuantityPositive() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetReissueMaxQuantityValue(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Reissue: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})

		}
	}
}

func (suite *ReissueTxSuite) Test_ReissueNotReissuableNegative() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetNotReissuableTestData(&suite.BaseSuite, itx.TxID)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//first tx should be successful
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Reissue: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.Positive.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.Positive.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//second reissue tx should be failed because of reissuable=false
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset = reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, testdata.ReissueDataChangedTimestamp(&td), v, !waitForTx)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.Negative.ErrGoMsg, td.Expected.Negative.ErrScalaMsg,
					tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.Negative.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.Negative.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *ReissueTxSuite) Test_ReissueNFTNegative() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, nft, v, waitForTx)
		tdmatrix := testdata.GetReissueNFTData(&suite.BaseSuite, itx.TxID)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *ReissueTxSuite) Test_ReissueNegative() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		//TODO (ipereiaslavskaia) For v1 of reissue tx negative cases for chainID will be ignored
		if v >= 2 {
			maps.Copy(tdmatrix, testdata.GetReissueChainIDNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
		}
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestReissueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxSuite))
}
