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

type UpdateAssetInfoTxPositiveSuite struct {
	f.BaseSuite
}

func (suite *UpdateAssetInfoTxPositiveSuite) Test_UpdateAssetInfoTxReissuableTokenPositive() {
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
					updateassetinfo.SendUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Update Asset Info tx: %s", caseName, tx.TxID.String())
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateassetinfo.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxPositiveSuite) Test_UpdateAssetInfoTxNFTPositive() {
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
					updateassetinfo.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Update Asset Info tx: %s", caseName, tx.TxID.String())
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateassetinfo.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxPositiveSuite) Test_UpdateAssetInfoTxSmartAssetPositive() {
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
					updateassetinfo.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Update Asset Info tx: %s", caseName, tx.TxID.String())
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateassetinfo.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func TestUpdateAssetInfoTxPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UpdateAssetInfoTxPositiveSuite))
}

type UpdateAssetInfoTxNegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *UpdateAssetInfoTxNegativeSuite) Test_UpdateAssetInfoTxReissuableTokenNegative() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue.SendWithTestData(&suite.BaseSuite, reissuable, iv, true)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			// ***wait n blocks***
			blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
			utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						updateassetinfo.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := fmt.Sprintf("Case: %s; Update Asset Info tx: %s", caseName, tx.TxID.String())
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateassetinfo.NegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
						initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxNegativeSuite) Test_UpdateAssetInfoTxNFTNegative() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
			itx := issue.SendWithTestData(&suite.BaseSuite, nft, iv, true)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			// ***wait n blocks***
			blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
			utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						updateassetinfo.SendUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := fmt.Sprintf("Case: %s; Update Asset Info tx: %s", caseName, tx.TxID.String())
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateassetinfo.NegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
						initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxNegativeSuite) Test_UpdateAssetInfoTxSmartAssetNegative() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersionsSmartAsset(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue.SendWithTestData(&suite.BaseSuite, smart, iv, true)
			tdmatrix := testdata.GetUpdateSmartAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			// ***wait n blocks***
			blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
			utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						updateassetinfo.SendUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := fmt.Sprintf("Case: %s; Update Asset Info tx: %s", caseName, tx.TxID.String())
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateassetinfo.NegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
						initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxNegativeSuite) Test_UpdateAssetInfoTxWithoutWaitingNegative() {
	versions := updateassetinfo.GetVersions(&suite.BaseSuite)
	issueVersions := issue.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue.SendWithTestData(&suite.BaseSuite, reissuable, iv, true)
			name := "Updating Asset Info without waiting"
			tdstruct := testdata.GetUpdateAssetInfoWithoutWaitingNegativeData(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for _, td := range tdstruct {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						updateassetinfo.SendUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := fmt.Sprintf("Case: %s; Update Asset Info tx: %s", caseName, tx.TxID.String())
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateassetinfo.NegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
						initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestUpdateAssetInfoTxNegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UpdateAssetInfoTxNegativeSuite))
}
