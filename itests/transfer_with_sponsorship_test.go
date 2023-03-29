package itests

import (
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

type TransferWithSponsorshipTxSuite struct {
	f.BaseSuite
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipPositive() {
	waitForTx := true
	versions := transfer_utilities.GetVersions()
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor set up sponsorship for this token
		sponsor_utilities.SponsorshipEnableSend(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
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
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer with Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				//RecipientSender balance in Waves does not change because of fee in sponsored asset
				//RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient
				//The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.FeeAssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Recipient balance in Waves changes if Waves were transferred
				//Recipient Asset balance increases by the asset amount being transferred
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipToOneselfPositive() {
	waitForTx := true
	versions := transfer_utilities.GetVersions()
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor set up sponsorship for this token
		sponsor_utilities.SponsorshipEnableSend(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
		//Sponsor issues one more token
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)

		tdmatrix := testdata.GetSposoredTransferBySponsorAsSender(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Sponsor transfers assets to himself, sponsored asset is used as fee asset
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer with Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.FeeAssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//Sponsor asset balance does not change
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestFeeInWavesAccordingMinSponsoredAssetPositive() {
	waitForTx := true
	versions := transfer_utilities.GetVersions()
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
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Sponsor set up sponsorship for the token
				sponsor_utilities.SponsorshipEnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer with Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				//RecipientSender balance in Waves does not change because of fee in sponsored asset
				//RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient
				//The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Recipient balance in Waves changes if Waves were transferred
				//Recipient Asset balance increases by the asset amount being transferred
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipMaxValuesPositive() {
	versions := transfer_utilities.GetVersions()
	waitForTx := true
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
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Sponsor set up sponsorship for the token
				sponsor_utilities.SponsorshipEnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer with Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				///RecipientSender balance in Waves does not change because of fee in sponsored asset
				//RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient
				//The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Recipient balance in Waves changes if Waves were transferred
				//Recipient Asset balance increases by the asset amount being transferred
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipNegative() {
	versions := transfer_utilities.GetVersions()
	waitForTx := true
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
		txIds := make(map[string]*crypto.Digest)

		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Sponsor set up sponsorship for the token
				sponsor_utilities.SponsorshipEnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, !waitForTx)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.TransferTestData.Expected.ErrGoMsg, td.TransferTestData.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v))

				//Balances of RecipientSender do not change
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Balances of Recipient do not change
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Balances of Sponsor do not change
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

			})
		}
		//actualTxIds should be empty
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestSponsoredTransferFeeNegative() {
	versions := transfer_utilities.GetVersions()
	waitForTx := true
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
		txIds := make(map[string]*crypto.Digest)

		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Sponsor set up sponsorship for the token
				sponsor_utilities.SponsorshipEnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, !waitForTx)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.TransferTestData.Expected.ErrGoMsg, td.TransferTestData.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v))

				//Balances of RecipientSender do not change
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Balances of Recipient do not change
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceRecipient,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//Balances of Sponsor do not change
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSponsor,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
					utl.GetTestcaseNameWithVersion(name, v))

			})
		}
		//actualTxIds should be empty
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestTransferWithSponsorshipTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipTxSuite))
}
