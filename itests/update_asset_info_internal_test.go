package itests

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/update_asset_info_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type UpdateAssetInfoTxSuite struct {
	f.BaseSuite
}

func updateAssetInfoPositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.UpdateAssetInfoTestData[testdata.UpdateAssetInfoExpectedPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	assetDetails utl.AssetInfo, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
	utl.AssetNameCheck(t, td.AssetName, assetDetails.AssetInfoGo.GetName(),
		assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetDescriptionCheck(t, td.AssetDesc, assetDetails.AssetInfoGo.GetDescription(),
		assetDetails.AssetInfoScala.GetDescription(), errMsg)
}

func updateAssetInfoNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.UpdateAssetInfoTestData[testdata.UpdateAssetInfoExpectedNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	initAssetDetails, assetDetails utl.AssetInfo, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

	utl.AssetNameCheck(t, initAssetDetails.AssetInfoGo.GetName(), assetDetails.AssetInfoGo.GetName(), assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetNameCheck(t, initAssetDetails.AssetInfoScala.GetName(), assetDetails.AssetInfoGo.GetName(), assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetDescriptionCheck(t, initAssetDetails.AssetInfoGo.GetDescription(), assetDetails.AssetInfoGo.GetDescription(),
		assetDetails.AssetInfoScala.GetDescription(), errMsg)
	utl.AssetDescriptionCheck(t, initAssetDetails.AssetInfoScala.GetDescription(),
		assetDetails.AssetInfoGo.GetDescription(),
		assetDetails.AssetInfoScala.GetDescription(), errMsg)
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxReissuableTokenPositive() {
	utl.SkipLongTest(suite.T())
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		assets := issue_utilities.GetReissuableMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, assets)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(
						&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateAssetInfoPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxNFTPositive() {
	utl.SkipLongTest(suite.T())
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		nft := issue_utilities.GetNFTMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, nft)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateAssetInfoPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxSmartAssetPositive() {
	utl.SkipLongTest(suite.T())
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		smart := issue_utilities.GetSmartAssetMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
		tdmatrix := testdata.GetUpdateSmartAssetInfoPositiveDataMatrix(&suite.BaseSuite, smart)
		//***wait n blocks***
		blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
		utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
				assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
				updateAssetInfoPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
					assetDetails, errMsg)
			})
		}
	}
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxReissuableTokenNegative() {
	utl.SkipLongTest(suite.T())
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	issueVersions := issue_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, iv, true)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateAssetInfoNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
						initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxNFTNegative() {
	utl.SkipLongTest(suite.T())
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	issueVersions := issue_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
			itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, nft, iv, true)
			tdmatrix := testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateAssetInfoNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
						initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxSmartAssetNegative() {
	utl.SkipLongTest(suite.T())
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	issueVersions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
			itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smart, iv, true)
			tdmatrix := testdata.GetUpdateSmartAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for name, td := range tdmatrix {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateAssetInfoNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
						initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxWithoutWaitingNegative() {
	utl.SkipLongTest(suite.T())
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	issueVersions := issue_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		for _, iv := range issueVersions {
			reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
			itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, iv, true)
			name := "Updating Asset Info without waiting"
			tdstruct := testdata.GetUpdateAssetInfoWithoutWaitingNegativeData(&suite.BaseSuite, itx.TxID)
			initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			for _, td := range tdstruct {
				caseName := utl.GetTestcaseNameWithVersion(name, v) + utl.AssetWithVersion(itx.TxID, int(iv))
				suite.Run(caseName, func() {
					tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
						update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(
							&suite.BaseSuite, td, v, false)
					txIds[name] = &tx.TxID
					errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
					assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
					updateAssetInfoNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
						initAssetDetails, assetDetails, errMsg)
				})
			}
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxReissuableTokenSmokePositive() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	assets := issue_utilities.GetReissuableMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, assets))
	//***wait n blocks***
	blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
	utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			updateAssetInfoPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxNFTSmokePositive() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	nft := issue_utilities.GetNFTMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, nft))
	//***wait n blocks***
	blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
	utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			updateAssetInfoPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxSmartAssetSmokePositive() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	smart := issue_utilities.GetSmartAssetMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetUpdateSmartAssetInfoPositiveDataMatrix(&suite.BaseSuite, smart))
	//***wait n blocks***
	blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
	utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			updateAssetInfoPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxReissuableTokenSmokeNegative() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	issueVersions := issue_utilities.GetVersions(&suite.BaseSuite)
	randIV := issueVersions[rand.Intn(len(issueVersions))]
	txIds := make(map[string]*crypto.Digest)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randIV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
	initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV) + utl.AssetWithVersion(itx.TxID, int(randIV))
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			updateAssetInfoNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				initAssetDetails, assetDetails, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxNFTSmokeNegative() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	issueVersions := issue_utilities.GetVersions(&suite.BaseSuite)
	randIV := issueVersions[rand.Intn(len(issueVersions))]
	txIds := make(map[string]*crypto.Digest)
	nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, nft, randIV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetUpdateAssetInfoNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
	initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV) + utl.AssetWithVersion(itx.TxID, int(randIV))
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			updateAssetInfoNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				initAssetDetails, assetDetails, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxSmartAssetSmokeNegative() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	issueVersions := issue_utilities.GetVersionsSmartAsset(&suite.BaseSuite)
	randIV := issueVersions[rand.Intn(len(issueVersions))]
	txIds := make(map[string]*crypto.Digest)
	smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smart, randIV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetUpdateSmartAssetInfoNegativeDataMatrix(
		&suite.BaseSuite, itx.TxID))
	initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV) + utl.AssetWithVersion(itx.TxID, int(randIV))
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			updateAssetInfoNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				initAssetDetails, assetDetails, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *UpdateAssetInfoTxSuite) TestUpdateAssetInfoTxWithoutWaitingSmokeNegative() {
	versions := update_asset_info_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	issueVersions := issue_utilities.GetVersions(&suite.BaseSuite)
	randIV := issueVersions[rand.Intn(len(issueVersions))]
	txIds := make(map[string]*crypto.Digest)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randIV, true)
	name := "Updating Asset Info without waiting"
	tdstruct := testdata.GetUpdateAssetInfoWithoutWaitingNegativeData(&suite.BaseSuite, itx.TxID)
	initAssetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
	for _, td := range tdstruct {
		caseName := utl.GetTestcaseNameWithVersion(name, randV) + utl.AssetWithVersion(itx.TxID, int(randIV))
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info_utilities.SendUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, randV, false)
			txIds[name] = &tx.TxID
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, itx.TxID)
			updateAssetInfoNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				initAssetDetails, assetDetails, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestUpdateAssetInfoTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UpdateAssetInfoTxSuite))
}
