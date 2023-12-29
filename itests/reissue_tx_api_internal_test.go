package itests

import (
	"math/rand"
	"net/http"
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

type ReissueTxApiSuite struct {
	f.BaseSuite
}

func reissuePositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.ReissueTestData[testdata.ReissueExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func reissueNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.ReissueTestData[testdata.ReissueExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
		tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Reissue tx:" + tx.TxID.String()
				reissuePositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiMaxQuantityPositive() {
	utl.SkipLongTest(suite.T())
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetReissueMaxQuantityValue(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Reissue tx:" + tx.TxID.String()
				reissuePositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueNotReissuableApiNegative() {
	utl.SkipLongTest(suite.T())
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetNotReissuableTestData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//first tx should be successful
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Reissue tx:" + tx.TxID.String()
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, errMsg)
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				//second reissue tx should be failed because of reissuable=false
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset =
					reissue_utilities.BroadcastReissueTxAndGetBalances(
						&suite.BaseSuite, testdata.ReissueDataChangedTimestamp(&td), v, false)
				errMsg = caseName + "Broadcast Reissue tx2:" + tx.TxID.String()
				txIds[name] = &tx.TxID
				reissueNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiNFTNegative() {
	utl.SkipLongTest(suite.T())
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, nft, v, true)
		tdmatrix := testdata.GetReissueNFTData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, v, false)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Broadcast Reissue tx:" + tx.TxID.String()
				reissueNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiNegative() {
	utl.SkipLongTest(suite.T())
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		//TODO (ipereiaslavskaia) For v1 of reissue tx negative cases for chainID will be ignored
		if v >= 2 {
			maps.Copy(tdmatrix, testdata.GetReissueChainIDNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
		}
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, v, false)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Broadcast Reissue tx:" + tx.TxID.String()
				reissueNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiSmokePositive() {
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Broadcast Reissue tx:" + tx.TxID.String()
			reissuePositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiSmokeNegative() {
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
	//TODO (ipereiaslavskaia) For v1 of reissue tx negative cases for chainID will be ignored
	if randV >= 2 {
		maps.Copy(tdmatrix, utl.GetRandomValueFromMap(testdata.GetReissueChainIDNegativeDataMatrix(
			&suite.BaseSuite, itx.TxID)))
	}
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Broadcast Reissue tx:" + tx.TxID.String()
			reissueNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxApiSuite) Test_ReissueTxApiNFTSmokeNegative() {
	versions := reissue_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
	itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, nft, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetReissueNFTData(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				reissue_utilities.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Broadcast Reissue tx:" + tx.TxID.String()
			reissueNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestReissueTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxApiSuite))
}
