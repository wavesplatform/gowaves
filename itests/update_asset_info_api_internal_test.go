//go:build !smoke

package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/update_asset_info"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type UpdateAssetInfoTxApiSuite struct {
	f.BaseSuite
}

func (suite *UpdateAssetInfoTxApiSuite) Test_UpdateAssetInfoTxApiReissuableTokenPositive() {
	versions := update_asset_info.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		assets := issue.GetReissuableMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, assets)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					update_asset_info.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				update_asset_info.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxApiSuite) Test_UpdateAssetInfoTxApiNFTPositive() {
	versions := update_asset_info.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		nft := issue.GetNFTMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, nft)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					update_asset_info.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				update_asset_info.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxApiSuite) Test_UpdateAssetInfoTxApiSmartAssetPositive() {
	versions := update_asset_info.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		smart := issue.GetSmartAssetMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateSmartAssetInfoPositiveDataMatrix(&suite.BaseSuite, smart)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					update_asset_info.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				update_asset_info.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxApiSuite) Test_UpdateAssetInfoTxApiReissuableTokenNegative() {
	versions := update_asset_info.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, iv, true)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						update_asset_info.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					update_asset_info.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
						actualDiffBalanceInAsset, initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxApiSuite) Test_UpdateAssetInfoTxNFTApiNegative() {
	versions := update_asset_info.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
			itx := issue.IssueBroadcastWithTestData(&suite.BaseSuite, nft, iv, true)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						update_asset_info.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					update_asset_info.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
						actualDiffBalanceInAsset, initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxApiSuite) Test_UpdateAssetInfoTxSmartAssetApiNegative() {
	versions := update_asset_info.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersionsSmartAsset(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue.IssueBroadcastWithTestData(&suite.BaseSuite, smart, iv, true)
			tdmatrix := testdata.GetUpdateSmartAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						update_asset_info.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Broadcast Update Asset Info tx: " + tx.TxID.String()
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					update_asset_info.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
						actualDiffBalanceInAsset, initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxApiSuite) Test_UpdateAssetInfoTxApiWithoutWaitingNegative() {
	versions := update_asset_info.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, iv, true)
			name := "Broadcast Update Asset Info without waiting"
			tdstruct := testdata.GetUpdateAssetInfoWithoutWaitingNegativeData(&suite.BaseSuite, itx.TxID)
			for _, td := range tdstruct {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						update_asset_info.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					update_asset_info.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
						actualDiffBalanceInAsset, initAssetDetails, assetDetails, errMsg)
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
