//go:build smoke

package itests

import (
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/alias"
	"github.com/wavesplatform/gowaves/itests/utilities/burn"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/reissue"
	"github.com/wavesplatform/gowaves/itests/utilities/setassetscript"
	"github.com/wavesplatform/gowaves/itests/utilities/sponsorship"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"github.com/wavesplatform/gowaves/itests/utilities/updateassetinfo"
)

type SmokeTxAPIPositiveSuite struct {
	f.BaseSuite
}

func (suite *SmokeTxAPIPositiveSuite) Test_AliasTxAPISmokePositive() {
	v := byte(testdata.AliasMaxVersion)
	tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, _, actualDiffBalanceInWaves := alias.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite,
				td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Alias Tx: %s", caseName, tx.TxID.String())
			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)
			alias.PositiveAPIChecks(suite.T(), tx, td, addrByAliasGo, addrByAliasScala,
				actualDiffBalanceInWaves, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) Test_BurnTxAPISmokePositive() {
	v := byte(testdata.BurnMaxVersion)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue.BroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
	tdmatrix := testdata.GetBurnPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn.BroadcastBurnTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Burn tx: %s", caseName, tx.TxID.String())
			burn.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) Test_IssueSmartAssetAPISmokePositive() {
	v := byte(testdata.IssueMaxVersion)
	tdmatrix := testdata.GetPositiveAssetScriptData(&suite.BaseSuite)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue.BroadcastIssueTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, tx.TxID)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Issue smart asset tx: %s", caseName, tx.TxID.String())
			issue.SmartAssetPositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails.AssetInfoGo.Script.ScriptBytes, assetDetails.AssetInfoScala.Script.ScriptBytes, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) Test_IssueTxAPISmokePositive() {
	v := byte(testdata.IssueMaxVersion)
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := issue.BroadcastIssueTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Issue tx: %s", caseName, tx.TxID.String())
			issue.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) Test_ReissueTxAPISmokePositive() {
	v := byte(testdata.ReissueMaxVersion)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue.BroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
	tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				reissue.BroadcastReissueTxAndGetBalances(&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Reissue tx: %s", caseName, tx.TxID.String())
			reissue.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) Test_SetAssetScriptAPISmokePositive() {
	v := byte(testdata.SetAssetScriptMaxVersion)
	smartAsset := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
	itx := issue.BroadcastWithTestData(&suite.BaseSuite, smartAsset, v, true)
	tdmatrix := testdata.GetSetAssetScriptPositiveData(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				setassetscript.BroadcastSetAssetScriptTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Set Asset Script tx: %s", caseName, tx.TxID.String())
			setassetscript.APIPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) TestSponsorshipTxAPISmokePositive() {
	v := byte(testdata.SponsorshipMaxVersion)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue.BroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
	tdmatrix := testdata.GetSponsorshipPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsorship.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Sponsorship tx: %s", caseName, tx.TxID.String())
			sponsorship.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) Test_TransferTxAPISmokePositive() {
	v := byte(testdata.TransferMaxVersion)
	aliasStr := utl.RandStringBytes(15, testdata.AliasSymbolSet)
	alias.SetAliasToAccountByAPI(&suite.BaseSuite, v, utl.TestChainID, aliasStr,
		utl.DefaultRecipientNotMiner)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue.BroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
	tdmatrix := testdata.GetTransferPositiveData(&suite.BaseSuite, itx.TxID, aliasStr)
	if v <= 2 {
		maps.Copy(tdmatrix, testdata.GetTransferChainIDDataBinaryVersions(
			&suite.BaseSuite, itx.TxID))
	}
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, testdata.TransferDataChangedTimestamp(&td), v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer tx: %s", caseName, tx.TxID.String())
			transfer.PositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) TestTransferWithSponsorshipAPISmokePositive() {
	v := byte(testdata.TransferMaxVersion)
	//Sponsor creates a new token
	sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor set up sponsorship for this token
	sponsorship.EnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
		sponsoredAssetID, testdata.DefaultMinSponsoredAssetFee)
	//Sponsor transfers all issued sponsored tokens to RecipientSender
	transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		sponsoredAssetID, testdata.Sponsor, testdata.RecipientSender)
	//Sponsor issues one more token
	assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		assetID, testdata.Sponsor, testdata.RecipientSender)

	tdmatrix := testdata.GetSponsoredTransferPositiveData(
		&suite.BaseSuite, assetID, sponsoredAssetID)

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
			tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
				caseName, tx.TxID.String())
			transfer.WithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) TestTransferWithSponsorshipToOneselfAPISmokePositive() {
	v := byte(testdata.TransferMaxVersion)
	//Sponsor creates a new token
	sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor set up sponsorship for this token
	sponsorship.EnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
		sponsoredAssetID, testdata.DefaultMinSponsoredAssetFee)
	//Sponsor issues one more token
	assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)

	tdmatrix := testdata.GetSposoredTransferBySponsorAsSender(
		&suite.BaseSuite, sponsoredAssetID, assetID)

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//Sponsor transfers assets to himself, sponsored asset is used as fee asset
			tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
				caseName, tx.TxID.String())
			transfer.WithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) TestFeeInWavesAccordingMinSponsoredAssetAPISmokePositive() {
	v := byte(testdata.TransferMaxVersion)
	//Sponsor creates a new token
	sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor issues one more token
	assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		sponsoredAssetID, testdata.Sponsor, testdata.RecipientSender)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		assetID, testdata.Sponsor, testdata.RecipientSender)

	tdmatrix := testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(
		&suite.BaseSuite, sponsoredAssetID, assetID)

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//Sponsor set up sponsorship for the token
			sponsorship.EnableBroadcast(&suite.BaseSuite, v,
				td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)

			//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
			tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td.TransferTestData, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
				caseName, tx.TxID.String())
			transfer.WithSponsorshipMinAssetFeePositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) TestUpdateAssetInfoTxAPIReissuableTokenSmokePositive() {
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
				updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s", caseName, tx.TxID.String())
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			updateassetinfo.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) TestUpdateAssetInfoTxAPINFTSmokePositive() {
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
				updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s", caseName, tx.TxID.String())
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			updateassetinfo.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func (suite *SmokeTxAPIPositiveSuite) TestUpdateAssetInfoTxAPISmartAssetSmokePositive() {
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
				updateassetinfo.BroadcastUpdateAssetInfoTxAndGetDiffBalances(
					&suite.BaseSuite, td, v, true)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Update Asset Info tx: %s", caseName, tx.TxID.String())
			assetDetails := utl.GetAssetInfoGrpc(&suite.BaseSuite, td.AssetID)
			updateassetinfo.PositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset,
				assetDetails, errMsg)
		})
	}
}

func TestSmokeTxAPIPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SmokeTxAPIPositiveSuite))
}
