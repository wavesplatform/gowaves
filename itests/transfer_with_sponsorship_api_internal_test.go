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
	"github.com/wavesplatform/gowaves/itests/utilities/sponsor_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type TransferWithSponsorshipApiTxSuite struct {
	f.BaseSuite
}

func transferWithSponsorshipPositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferSponsoredExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	//RecipientSender balance in Waves does not change because of fee in sponsored asset
	//RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient
	//The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)
	//Recipient balance in Waves changes if Waves were transferred
	//Recipient Asset balance increases by the asset amount being transferred
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	//Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
	//Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}

func transferWithSponsorshipMinAssetFeePositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferSponsoredTestData[testdata.TransferSponsoredExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	//RecipientSender balance in Waves does not change because of fee in sponsored asset
	//RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient
	//The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)
	//Recipient balance in Waves changes if Waves were transferred
	//Recipient Asset balance increases by the asset amount being transferred
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	//Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
	//Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}

func transferWithSponsorshipNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferSponsoredTestData[testdata.TransferSponsoredExpectedValuesNegative],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.TransferTestData.Expected.ErrGoMsg, td.TransferTestData.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)

	//Balances of RecipientSender do not change
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)

	//Balances of Recipient do not change
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)

	//Balances of Sponsor do not change
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor set up sponsorship for this token
		sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
		//Sponsor transfers all issued sponsored tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Sponsor issues one more token
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)

		tdmatrix := testdata.GetSponsoredTransferPositiveData(&suite.BaseSuite, assetId, sponsoredAssetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				transferWithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipToOneselfApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor set up sponsorship for this token
		sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
		//Sponsor issues one more token
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)

		tdmatrix := testdata.GetSposoredTransferBySponsorAsSender(&suite.BaseSuite, sponsoredAssetId, assetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor transfers assets to himself, sponsored asset is used as fee asset
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				transferWithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestFeeInWavesAccordingMinSponsoredAssetApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor issues one more token
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)

		tdmatrix := testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(&suite.BaseSuite,
			sponsoredAssetId, assetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, true)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				transferWithSponsorshipMinAssetFeePositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipMaxValuesApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		//Fill Sponsor's Waves balance
		transfer_utilities.TransferFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, testdata.Sponsor, 100000000000000)
		//Sponsor creates a new token
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor issues one more token
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipMaxAmountPositive(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, true)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				transferWithSponsorshipMinAssetFeePositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipApiNegative() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor issues one more token
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipMaxValuesDataNegative(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, false)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				transferWithSponsorshipNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	//actualTxIds should be empty
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferWithSponsorshipApiTxSuite) TestSponsoredTransferFeeApiNegative() {
	utl.SkipLongTest(suite.T())
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)

	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, 10000000000)
		//Sponsor issues one more token
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipDataNegative(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, false)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				transferWithSponsorshipNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	//actualTxIds should be empty
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipApiSmokePositive() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	//Sponsor creates a new token
	sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor set up sponsorship for this token
	sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
	//Sponsor transfers all issued sponsored tokens to RecipientSender
	transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
	//Sponsor issues one more token
	assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		assetId, testdata.Sponsor, testdata.RecipientSender)

	tdmatrix := utl.GetRandomValueFromMap(testdata.GetSponsoredTransferPositiveData(
		&suite.BaseSuite, assetId, sponsoredAssetId))

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
			tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
			transferWithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipToOneselfApiSmokePositive() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	//Sponsor creates a new token
	sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor set up sponsorship for this token
	sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
	//Sponsor issues one more token
	assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)

	tdmatrix := utl.GetRandomValueFromMap(testdata.GetSposoredTransferBySponsorAsSender(
		&suite.BaseSuite, sponsoredAssetId, assetId))

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			//Sponsor transfers assets to himself, sponsored asset is used as fee asset
			tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
			transferWithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestFeeInWavesAccordingMinSponsoredAssetApiSmokePositive() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	//Sponsor creates a new token
	sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor issues one more token
	assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		assetId, testdata.Sponsor, testdata.RecipientSender)

	tdmatrix := utl.GetRandomValueFromMap(testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(
		&suite.BaseSuite, sponsoredAssetId, assetId))

	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			//Sponsor set up sponsorship for the token
			sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, randV,
				td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

			//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
			tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td.TransferTestData, randV, true)
			errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
			transferWithSponsorshipMinAssetFeePositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipApiSmokeNegative() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	//Sponsor creates a new token
	sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor issues one more token
	assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		assetId, testdata.Sponsor, testdata.RecipientSender)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetTransferWithSponsorshipMaxValuesDataNegative(
		&suite.BaseSuite, sponsoredAssetId, assetId))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			//Sponsor set up sponsorship for the token
			sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, randV,
				td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

			//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
			tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td.TransferTestData, randV, false)
			errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			transferWithSponsorshipNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
	//actualTxIds should be empty
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferWithSponsorshipApiTxSuite) TestSponsoredTransferFeeApiSmokeNegative() {
	versions := transfer_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	//Sponsor creates a new token
	sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, 10000000000)
	//Sponsor issues one more token
	assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
		testdata.Sponsor, utl.MaxAmount)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
	//Sponsor transfers all issued tokens to RecipientSender
	transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
		assetId, testdata.Sponsor, testdata.RecipientSender)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetTransferWithSponsorshipDataNegative(
		&suite.BaseSuite, sponsoredAssetId, assetId))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			//Sponsor set up sponsorship for the token
			sponsor_utilities.SponsorshipEnableBroadcast(&suite.BaseSuite, randV,
				td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

			//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
			tx, diffBalances := transfer_utilities.BroadcastTransferTxAndGetBalances(
				&suite.BaseSuite, td.TransferTestData, randV, false)
			errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			transferWithSponsorshipNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
		})
	}
	//actualTxIds should be empty
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferWithSponsorshipApiTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipApiTxSuite))
}
