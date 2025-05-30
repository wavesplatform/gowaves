//go:build !smoke

package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/updateassetinfo"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type UpdateAssetInfoTxAPIPositiveSuite struct {
	f.BaseSuite
}

func (suite *UpdateAssetInfoTxAPIPositiveSuite) Test_UpdateAssetInfoTxAPIReissuableTokenPositive() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		assets := issue.GetReissuableMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, assets)
		// ***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s", caseName, tx.TxID.String())
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateassetinfo.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxAPIPositiveSuite) Test_UpdateAssetInfoTxAPINFTPositive() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		nft := issue.GetNFTMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, nft)
		// ***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s", caseName, tx.TxID.String())
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateassetinfo.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxAPIPositiveSuite) Test_UpdateAssetInfoTxAPISmartAssetPositive() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		smart := issue.GetSmartAssetMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateSmartAssetInfoPositiveDataMatrix(&suite.BaseSuite, smart)
		// ***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s", caseName, tx.TxID.String())
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateassetinfo.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, assetDetails, errMsg)
			})
		}
	}
}

func TestUpdateAssetInfoTxAPIPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UpdateAssetInfoTxAPIPositiveSuite))
}

type UpdateAssetInfoTxAPINegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *UpdateAssetInfoTxAPINegativeSuite) Test_UpdateAssetInfoTxAPIReissuableTokenNegative() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue.BroadcastWithTestData(&suite.BaseSuite, reissuable, iv, true)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			// ***wait n blocks***
			blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
			utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s",
						caseName, tx.TxID.String())
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateassetinfo.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
						actualDiffBalanceInAsset, initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxAPINegativeSuite) Test_UpdateAssetInfoTxNFTAPINegative() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
			itx := issue.BroadcastWithTestData(&suite.BaseSuite, nft, iv, true)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			// ***wait n blocks***
			blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
			utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s",
						caseName, tx.TxID.String())
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateassetinfo.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
						actualDiffBalanceInAsset, initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxAPINegativeSuite) Test_UpdateAssetInfoTxSmartAssetAPINegative() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersionsSmartAsset(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue.BroadcastWithTestData(&suite.BaseSuite, smart, iv, true)
			tdmatrix := testdata.GetUpdateSmartAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			// ***wait n blocks***
			blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
			utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s",
						caseName, tx.TxID.String())
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateassetinfo.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
						actualDiffBalanceInAsset, initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxAPINegativeSuite) Test_UpdateAssetInfoTxAPIWithoutWaitingNegative() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue.BroadcastWithTestData(&suite.BaseSuite, reissuable, iv, true)
			name := "Broadcast Update Asset Info without waiting"
			tdstruct := testdata.GetUpdateAssetInfoWithoutWaitingNegativeData(&suite.BaseSuite, itx.TxID)
			for _, td := range tdstruct {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s",
						caseName, tx.TxID.String())
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateassetinfo.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
						actualDiffBalanceInAsset, initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestUpdateAssetInfoTxAPINegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UpdateAssetInfoTxAPINegativeSuite))
}
