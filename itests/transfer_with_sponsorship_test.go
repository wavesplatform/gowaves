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
		//предусловия
		//Аккаунт Sponsor выпускает токен
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor делает выпущенный токен спонсорским (нужны ли тестовые данные для спонсорства? разные minAssetFee)
		sponsor_utilities.SponsorshipOnSend(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
		//Аккаунт Sponsor переводит все выпущенные спонсорские на счет Аккаунта RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Аккаунт Sponsor выпускает еще один токен
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor переводит все выпущенные токены на счет Аккаунта RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)

		tdmatrix := testdata.GetTransferSponsoredPositiveData(&suite.BaseSuite, assetId, sponsoredAssetId)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Аккаунт RecipientSender переводит ассеты на Аккаунт Recipient, указывая в транзакции в качестве fee спонсорский ассет
				tx, diffBalancesSender, diffBalancesRecipient, diffBalancesSponsor := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer with Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта RecipientSender баланс Waves не изменяется на комиссию, так как комиссия в спонсорском ассете
				//У Аккаунта RecipientSender уменьшается баланс токенов (waves), которые он переводит Аккаунту Recipient, на переводимое количество
				//У Аккаунта RecipientSender уменьшается баланс токенов, указанных в качестве комиссии, на величину комиссии
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.FeeAssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Recipient баланс Waves меняется только если ему переводят Waves
				//у Аккаунта Recipient баланс токенов увеличивается на переводимое количество
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Sponsor списывается со счета количество waves, равное feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//У Аккаунта Sponsor увеличивается баланс токенов на комиссию за транзакцию перевода Аккаунта RecipientSender
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipToOneselfPositive() {
	//предусловия
	waitForTx := true
	versions := transfer_utilities.GetVersions()
	for _, v := range versions {
		//предусловия
		//Аккаунт Sponsor выпускает токен
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor делает выпущенный токен спонсорским (нужны ли тестовые данные для спонсорства? разные minAssetFee)
		sponsor_utilities.SponsorshipOnSend(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
		//Аккаунт Sponsor выпускает еще один токен
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)

		tdmatrix := testdata.GetTransferWithSponsorshipToOneselfData(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Аккаунт Sponsor переводит ассеты себе, указывая в транзакции в качестве fee спонсорский ассет
				tx, diffBalancesSender, diffBalancesRecipient, diffBalancesSponsor := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer with Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.FeeAssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Sponsor списывается со счета количество waves, равное feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//У Аккаунта Sponsor не меняется баланс токенов
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestFeeInWavesAccordingMinSponsoredAssetPositive() {
	//предусловия
	waitForTx := true
	versions := transfer_utilities.GetVersions()
	for _, v := range versions {
		//предусловия
		//Аккаунт Sponsor выпускает токен
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor выпускает еще один токен
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor переводит все выпущенные спонсорские на счет Аккаунта RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Аккаунт Sponsor переводит все выпущенные токены на счет Аккаунта RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(&suite.BaseSuite,
			sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Аккаунт Sponsor делает выпущенный токен спонсорским
				sponsor_utilities.SponsorshipOnSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//Аккаунт RecipientSender переводит ассеты на Аккаунт Recipient, указывая в транзакции в качестве fee спонсорский ассет
				tx, diffBalancesSender, diffBalancesRecipient, diffBalancesSponsor := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer with Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта RecipientSender баланс Waves не изменяется на комиссию, так как комиссия в спонсорском ассете
				//У Аккаунта RecipientSender уменьшается баланс токенов (waves), которые он переводит Аккаунту Recipient, на переводимое количество
				//У Аккаунта RecipientSender уменьшается баланс токенов, указанных в качестве комиссии, на величину комиссии
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Recipient баланс Waves меняется только если ему переводят Waves
				//у Аккаунта Recipient баланс токенов увеличивается на переводимое количество
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Sponsor списывается со счета количество waves, равное feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//У Аккаунта Sponsor увеличивается баланс токенов на комиссию за транзакцию перевода Аккаунта RecipientSender
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipMaxValuesPositive() {
	versions := transfer_utilities.GetVersions()
	waitForTx := true
	for _, v := range versions {
		//пополняем баланс спонсора
		transfer_utilities.TransferFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, testdata.Sponsor, 100000000000000)
		//Аккаунт Sponsor выпускает токен
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor выпускает еще один токен
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor переводит все выпущенные спонсорские на счет Аккаунта RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Аккаунт Sponsor переводит все выпущенные токены на счет Аккаунта RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipMaxAmountPositive(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Аккаунт Sponsor делает выпущенный токен спонсорским
				sponsor_utilities.SponsorshipOnSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//Аккаунт RecipientSender переводит ассеты на Аккаунт Recipient, указывая в транзакции в качестве fee спонсорский ассет
				tx, diffBalancesSender, diffBalancesRecipient, diffBalancesSponsor := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Transfer with Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта RecipientSender баланс Waves не изменяется на комиссию, так как комиссия в спонсорском ассете
				//У Аккаунта RecipientSender уменьшается баланс токенов (waves), которые он переводит Аккаунту Recipient, на переводимое количество
				//У Аккаунта RecipientSender уменьшается баланс токенов, указанных в качестве комиссии, на величину комиссии
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Recipient баланс Waves меняется только если ему переводят Waves
				//у Аккаунта Recipient баланс токенов увеличивается на переводимое количество
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Sponsor списывается со счета количество waves, равное feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//У Аккаунта Sponsor увеличивается баланс токенов на комиссию за транзакцию перевода Аккаунта RecipientSender
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipNegative() {
	versions := transfer_utilities.GetVersions()
	waitForTx := true
	for _, v := range versions {
		//Аккаунт Sponsor выпускает токен
		sponsoredAssetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor выпускает еще один токен
		assetId := issue_utilities.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Аккаунт Sponsor переводит все выпущенные спонсорские на счет Аккаунта RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Аккаунт Sponsor переводит все выпущенные токены на счет Аккаунта RecipientSender
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipDataNegative(&suite.BaseSuite, sponsoredAssetId, assetId)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//Аккаунт Sponsor делает выпущенный токен спонсорским
				sponsor_utilities.SponsorshipOnSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//Аккаунт RecipientSender переводит ассеты на Аккаунт Recipient, указывая в транзакции в качестве fee спонсорский ассет
				tx, diffBalancesSender, diffBalancesRecipient, diffBalancesSponsor := transfer_utilities.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, !waitForTx)
				txIds[name] = &tx.TxID

				//У Аккаунта RecipientSender баланс Waves не изменяется на комиссию, так как комиссия в спонсорском ассете
				//У Аккаунта RecipientSender уменьшается баланс токенов (waves), которые он переводит Аккаунту Recipient, на переводимое количество
				//У Аккаунта RecipientSender уменьшается баланс токенов, указанных в качестве комиссии, на величину комиссии
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSender,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
					diffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Recipient баланс Waves меняется только если ему переводят Waves
				//у Аккаунта Recipient баланс токенов увеличивается на переводимое количество
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceRecipient,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				//У Аккаунта Sponsor списывается со счета количество waves, равное feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//У Аккаунта Sponsor увеличивается баланс токенов на комиссию за транзакцию перевода Аккаунта RecipientSender
				utl.WavesDiffBalanceCheck(suite.T(), td.TransferTestData.Expected.WavesDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
					diffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.AssetBalanceCheck(suite.T(), td.TransferTestData.Expected.AssetDiffBalanceSponsor,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
					diffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))

				utl.ErrorMessageCheck(suite.T(), td.TransferTestData.Expected.ErrGoMsg, td.TransferTestData.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestTransferWithSponsorshipTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipTxSuite))
}
