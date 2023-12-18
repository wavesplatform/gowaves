package itests

import (
	"math/rand"
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

func reissuePositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.ReissueTestData[testdata.ReissueExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func reissueNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.ReissueTestData[testdata.ReissueExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *ReissueTxSuite) Test_ReissuePositive() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Reissue Tx tests in short mode")
	}
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					reissue_utilities.SendReissueTxAndGetBalances(&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Reissue tx:" + tx.TxID.String()

				reissuePositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *ReissueTxSuite) Test_ReissueMaxQuantityPositive() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Reissue Tx tests in short mode")
	}
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetReissueMaxQuantityValue(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Reissue tx:" + tx.TxID.String()
				reissuePositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *ReissueTxSuite) Test_ReissueNotReissuableNegative() {
	if testing.Short() {
		suite.T().Skip("skipping long negative Reissue Tx tests in short mode")
	}
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetNotReissuableTestData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//first tx should be successful
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Reissue tx:" + tx.TxID.String()
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				//second reissue tx should be failed because of reissuable=false
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset = reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, testdata.ReissueDataChangedTimestamp(&td), v, !waitForTx)
				txIds[name] = &tx.TxID
				errMsg = caseName + "Broadcast Reissue tx2:" + tx.TxID.String()
				reissueNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxSuite) Test_ReissueNFTNegative() {
	if testing.Short() {
		suite.T().Skip("skipping long negative Reissue Tx tests in short mode")
	}
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, nft, v, waitForTx)
		tdmatrix := testdata.GetReissueNFTData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Reissue tx:" + tx.TxID.String()
				reissueNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxSuite) Test_ReissueNegative() {
	if testing.Short() {
		suite.T().Skip("skipping long negative Reissue Tx tests in short mode")
	}
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		//TODO (ipereiaslavskaia) For v1 of reissue tx negative cases for chainID will be ignored
		if v >= 2 {
			maps.Copy(tdmatrix, testdata.GetReissueChainIDNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
		}
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
					&suite.BaseSuite, td, v, !waitForTx)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Reissue tx:" + tx.TxID.String()
				reissueNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxSuite) Test_ReissueSmokePositive() {
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	waitForTx := true
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randV, waitForTx)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				reissue_utilities.SendReissueTxAndGetBalances(&suite.BaseSuite, td, randV, waitForTx)
			errMsg := caseName + "Reissue tx:" + tx.TxID.String()
			reissuePositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *ReissueTxSuite) Test_ReissueSmokeNegative() {
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randV, waitForTx)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
	//TODO (ipereiaslavskaia) For v1 of reissue tx negative cases for chainID will be ignored
	if randV >= 2 {
		maps.Copy(tdmatrix, utl.GetRandomValueFromMap(testdata.GetReissueChainIDNegativeDataMatrix(
			&suite.BaseSuite, itx.TxID)))
	}
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
				&suite.BaseSuite, td, randV, !waitForTx)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Reissue tx:" + tx.TxID.String()
			reissueNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxSuite) Test_ReissueNFTSmokeNegative() {
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, nft, randV, waitForTx)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetReissueNFTData(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := reissue_utilities.SendReissueTxAndGetBalances(
				&suite.BaseSuite, td, randV, !waitForTx)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Reissue tx:" + tx.TxID.String()
			reissueNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestReissueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxSuite))
}
