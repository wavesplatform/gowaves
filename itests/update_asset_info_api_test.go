package itests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/update_asset_info_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type UpdateAssetInfoTxApiSuite struct {
	f.BaseSuite
}

func (suite *UpdateAssetInfoTxApiSuite) TestUpdateAssetInfoTxApiReissuableTokenPositive() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		assets := issue_utilities.GetReissuableMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, assets)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := update_asset_info_utilities.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, errMsg)
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

				assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				utl.AssetNameCheck(suite.T(), td.AssetName, assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
				utl.AssetDescriptionCheck(suite.T(), td.AssetDesc, assetDetailsGo.GetDescription(),
					assetDetailsScala.GetDescription(), errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxApiSuite) TestUpdateAssetInfoTxApiNFTPositive() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		nft := issue_utilities.GetNFTMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, nft)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := update_asset_info_utilities.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, errMsg)
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

				assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				utl.AssetNameCheck(suite.T(), td.AssetName, assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
				utl.AssetDescriptionCheck(suite.T(), td.AssetDesc, assetDetailsGo.GetDescription(),
					assetDetailsScala.GetDescription(), errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxApiSuite) TestUpdateAssetInfoTxApiSmartAssetPositive() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		smart := issue_utilities.GetSmartAssetMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateSmartAssetInfoPositiveDataMatrix(&suite.BaseSuite, smart)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := update_asset_info_utilities.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, errMsg)
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

				assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				utl.AssetNameCheck(suite.T(), td.AssetName, assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
				utl.AssetDescriptionCheck(suite.T(), td.AssetDesc, assetDetailsGo.GetDescription(),
					assetDetailsScala.GetDescription(), errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxApiSuite) TestUpdateAssetInfoTxApiReissuableTokenNegative() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	issue_versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issue_versions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, iv, waitForTx)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetailsGo, initAssetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := update_asset_info_utilities.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, !waitForTx)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()

					utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
					utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
						tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
					utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
						tx.WtErr.ErrWtScala, errMsg)
					utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
						actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
					utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
						actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

					assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					utl.AssetNameCheck(suite.T(), initAssetDetailsGo.GetName(), assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
					utl.AssetNameCheck(suite.T(), initAssetDetailsScala.GetName(), assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
					utl.AssetDescriptionCheck(suite.T(), initAssetDetailsGo.GetDescription(), assetDetailsGo.GetDescription(),
						assetDetailsScala.GetDescription(), errMsg)
					utl.AssetDescriptionCheck(suite.T(), initAssetDetailsScala.GetDescription(), assetDetailsGo.GetDescription(),
						assetDetailsScala.GetDescription(), errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxApiSuite) TestUpdateAssetInfoTxNFTApiNegative() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	issue_versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issue_versions {
			nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
			itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, nft, iv, waitForTx)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetailsGo, initAssetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := update_asset_info_utilities.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, !waitForTx)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()

					utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
					utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
						tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
					utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
						tx.WtErr.ErrWtScala, errMsg)
					utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
						actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
					utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
						actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

					assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					utl.AssetNameCheck(suite.T(), initAssetDetailsGo.GetName(), assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
					utl.AssetNameCheck(suite.T(), initAssetDetailsScala.GetName(), assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
					utl.AssetDescriptionCheck(suite.T(), initAssetDetailsGo.GetDescription(), assetDetailsGo.GetDescription(),
						assetDetailsScala.GetDescription(), errMsg)
					utl.AssetDescriptionCheck(suite.T(), initAssetDetailsScala.GetDescription(), assetDetailsGo.GetDescription(),
						assetDetailsScala.GetDescription(), errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxApiSuite) TestUpdateAssetInfoTxSmartAssetApiNegative() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	issue_versions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issue_versions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, smart, iv, waitForTx)
			tdmatrix := testdata.GetUpdateSmartAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetailsGo, initAssetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := update_asset_info_utilities.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, !waitForTx)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()

					utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
					utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
						tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
					utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
						tx.WtErr.ErrWtScala, errMsg)
					utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
						actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
					utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
						actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

					assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					utl.AssetNameCheck(suite.T(), initAssetDetailsGo.GetName(), assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
					utl.AssetNameCheck(suite.T(), initAssetDetailsScala.GetName(), assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
					utl.AssetDescriptionCheck(suite.T(), initAssetDetailsGo.GetDescription(), assetDetailsGo.GetDescription(),
						assetDetailsScala.GetDescription(), errMsg)
					utl.AssetDescriptionCheck(suite.T(), initAssetDetailsScala.GetDescription(), assetDetailsGo.GetDescription(),
						assetDetailsScala.GetDescription(), errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxApiSuite) TestUpdateAssetInfoTxApiWithoutWaitingNegative() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	issue_versions := issue_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issue_versions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, iv, waitForTx)
			name := "Broadcast Update Asset Info without waiting"
			tdstruct := testdata.GetUpdateAssetInfoWithoutWaitingNegativeData(&suite.BaseSuite, itx.TxID)
			for _, td := range tdstruct {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				initAssetDetailsGo, initAssetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := update_asset_info_utilities.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, !waitForTx)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()

					utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
						tx.WtErr.ErrWtScala, errMsg)
					utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
						actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
					utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
						actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

					assetDetailsGo, assetDetailsScala := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					utl.AssetNameCheck(suite.T(), initAssetDetailsGo.GetName(), assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
					utl.AssetNameCheck(suite.T(), initAssetDetailsScala.GetName(), assetDetailsGo.GetName(), assetDetailsScala.GetName(), errMsg)
					utl.AssetDescriptionCheck(suite.T(), initAssetDetailsGo.GetDescription(), assetDetailsGo.GetDescription(),
						assetDetailsScala.GetDescription(), errMsg)
					utl.AssetDescriptionCheck(suite.T(), initAssetDetailsScala.GetDescription(), assetDetailsGo.GetDescription(),
						assetDetailsScala.GetDescription(), errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestUpdateAssetInfoTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UpdateAssetInfoTxApiSuite))
}
