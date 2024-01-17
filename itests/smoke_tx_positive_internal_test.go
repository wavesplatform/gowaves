//go:build smoke

package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/alias"
	"github.com/wavesplatform/gowaves/itests/utilities/burn"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/reissue"
	"github.com/wavesplatform/gowaves/itests/utilities/set_asset_script"
	"github.com/wavesplatform/gowaves/itests/utilities/sponsorship"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"github.com/wavesplatform/gowaves/itests/utilities/update_asset_info"
	"golang.org/x/exp/maps"
)

type SmokeTxPositiveSuite struct {
	f.BaseSuite
}

func (suite *SmokeTxPositiveSuite) Test_AliasSmokePositive() {
	v := byte(testdata.AliasMaxVersion)
	tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, _, actualDiffBalanceInWaves := alias.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td,
				v, true)
			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)
			errMsg := caseName + "Alias Tx: " + tx.TxID.String()
			alias.PositiveChecks(suite.T(), tx, td, addrByAliasGo, addrByAliasScala, actualDiffBalanceInWaves, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) Test_BurnTxSmokePositive() {
	v := byte(testdata.BurnMaxVersion)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, true)
	tdmatrix := testdata.GetBurnPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn.SendBurnTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Burn tx" + tx.TxID.String()
			burn.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) Test_IssueSmartAssetSmokePositive() {
	v := byte(testdata.IssueMaxVersion)
	tdmatrix := testdata.GetPositiveAssetScriptData(&suite.BaseSuite)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue.SendIssueTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
			errMsg := caseName + "Issue smart asset tx:" + tx.TxID.String()
			issue.SmartAssetPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails.AssetInfoGo.Script.ScriptBytes, assetDetails.AssetInfoScala.Script.ScriptBytes, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) Test_IssueTxSmokePositive() {
	v := byte(testdata.IssueMaxVersion)
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue.SendIssueTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Issue tx:" + tx.TxID.String()
			issue.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) Test_ReissueSmokePositive() {
	v := byte(testdata.ReissueMaxVersion)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, true)
	tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				reissue.SendReissueTxAndGetBalances(&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Reissue tx:" + tx.TxID.String()
			reissue.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) Test_SetAssetScriptSmokePositive() {
	v := byte(testdata.SetAssetScriptMaxVersion)
	smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
	itx := issue.IssueSendWithTestData(&suite.BaseSuite, smartAsset, v, true)
	tdmatrix := testdata.GetSetAssetScriptPositiveData(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				set_asset_script.SendSetAssetScriptTxAndGetBalances(&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Set Asset Script tx: " + tx.TxID.String()
			set_asset_script.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) TestSponsorshipTxSmokePositive() {
	v := byte(testdata.SponsorshipMaxVersion)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, true)
	tdmatrix := testdata.GetSponsorshipPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsorship.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Sponsorship tx: " + tx.TxID.String()
			sponsorship.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) Test_TransferTxSmokePositive() {
	v := byte(testdata.TransferMaxVersion)
	aliasStr := utl.RandStringBytes(15, testdata.AliasSymbolSet)
	alias.SetAliasToAccount(&suite.BaseSuite, v, utl.TestChainID, aliasStr, utl.DefaultRecipientNotMiner)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, true)
	tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, aliasStr)
	if v <= 2 {
		maps.Copy(tdmatrix, testdata.GetTransferChainIDDataBinaryVersions(
			&suite.BaseSuite, itx.TxID))
	}
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, diffBalances := transfer.SendTransferTxAndGetBalances(
				&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, true)
			errMsg := caseName + "Transfer tx: " + tx.TxID.String()
			transfer.PositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) TestTransferWithSponsorshipSmokePositive() {
	v := byte(testdata.TransferMaxVersion)
	//Sponsor creates a new token
	sponsoredAssetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor set up sponsorship for this token
	sponsorship.SponsorshipEnableSend(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
	//Sponsor transfers all issued sponsored tokens to RecipientSender
	transfer.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
	//Sponsor issues one more token
	assetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		assetId, testdata.Sponsor, testdata.RecipientSender)
	//Test data
	tdmatrix := testdata.GetSponsoredTransferPositiveData(&suite.BaseSuite,
		assetId, sponsoredAssetId)

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
			tx, diffBalances := transfer.SendTransferTxAndGetBalances(&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Transfer with Sponsorship: " + tx.TxID.String()
			//Checks
			transfer.WithSponsorshipPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) TestTransferWithSponsorshipToOneselfSmokePositive() {
	v := byte(testdata.TransferMaxVersion)
	//Sponsor creates a new token
	sponsoredAssetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor set up sponsorship for this token
	sponsorship.SponsorshipEnableSend(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
	//Sponsor issues one more token
	assetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Test data
	tdmatrix := testdata.GetSposoredTransferBySponsorAsSender(&suite.BaseSuite,
		sponsoredAssetId, assetId)

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//Sponsor transfers assets to himself, sponsored asset is used as fee asset
			//Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
			//Sponsor asset balance does not change
			tx, diffBalances := transfer.SendTransferTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Transfer with Sponsorship: " + tx.TxID.String()
			//Checks
			transfer.WithSponsorshipPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) TestFeeInWavesAccordingMinSponsoredAssetSmokePositive() {
	v := byte(testdata.TransferMaxVersion)
	//Sponsor creates a new token
	sponsoredAssetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor issues one more token
	assetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		assetId, testdata.Sponsor, testdata.RecipientSender)
	//Test data
	tdmatrix := testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(
		&suite.BaseSuite, sponsoredAssetId, assetId)

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//Sponsor set up sponsorship for the token
			sponsorship.SponsorshipEnableSend(&suite.BaseSuite, v,
				td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)
			//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
			tx, diffBalances := transfer.SendTransferTxAndGetBalances(
				&suite.BaseSuite, td.TransferTestData, v, true)
			errMsg := caseName + "Transfer with Sponsorship: " + tx.TxID.String()
			transfer.WithSponsorshipMinAssetFeePositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) TestUpdateAssetInfoTxReissuableTokenSmokePositive() {
	v := byte(testdata.UpdateAssetInfoMaxVersion)
	assets := issue.GetReissuableMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
	tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, assets)
	//***wait n blocks***
	blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
	utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			update_asset_info.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) TestUpdateAssetInfoTxNFTSmokePositive() {
	v := byte(testdata.UpdateAssetInfoMaxVersion)
	nft := issue.GetNFTMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
	tdmatrix := testdata.GetUpdateAssetInfoPositiveDataMatrix(&suite.BaseSuite, nft)
	//***wait n blocks***
	blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
	utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			update_asset_info.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func (suite *SmokeTxPositiveSuite) TestUpdateAssetInfoTxSmartAssetSmokePositive() {
	v := byte(testdata.UpdateAssetInfoMaxVersion)
	smart := issue.GetSmartAssetMatrix(&suite.BaseSuite, testdata.PositiveCasesCount)
	tdmatrix := testdata.GetUpdateSmartAssetInfoPositiveDataMatrix(&suite.BaseSuite, smart)
	//***wait n blocks***
	blocksToWait := suite.Cfg.BlockchainSettings.MinUpdateAssetInfoInterval
	utl.WaitForHeight(&suite.BaseSuite, utl.GetHeight(&suite.BaseSuite)+blocksToWait)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				update_asset_info.SendUpdateAssetInfoTxAndGetDiffBalances(&suite.BaseSuite, td, v, true)
			errMsg := caseName + "Updating Asset Info tx: " + tx.TxID.String()
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			update_asset_info.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func TestSmokeTxPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SmokeTxPositiveSuite))
}
