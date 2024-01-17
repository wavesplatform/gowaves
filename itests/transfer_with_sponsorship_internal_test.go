//go:build !smoke

package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/sponsorship"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type TransferWithSponsorshipTxSuite struct {
	f.BaseSuite
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
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
		tdmatrix := testdata.GetSponsoredTransferPositiveData(&suite.BaseSuite, assetId, sponsoredAssetId)

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
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipToOneselfPositive() {
	waitForTx := true
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
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
		tdmatrix := testdata.GetSposoredTransferBySponsorAsSender(&suite.BaseSuite, sponsoredAssetId, assetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor transfers assets to himself, sponsored asset is used as fee asset
				//Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee
				//Sponsor asset balance does not change
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Transfer with Sponsorship: " + tx.TxID.String()
				//Checks
				transfer.WithSponsorshipPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxSuite) TestFeeInWavesAccordingMinSponsoredAssetPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
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
		tdmatrix := testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(&suite.BaseSuite,
			sponsoredAssetId, assetId)

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
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipMaxValuesPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		//Fill Sponsor's Waves balance
		transfer.TransferFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, testdata.Sponsor, 100000000000000)
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
		tdmatrix := testdata.GetTransferWithSponsorshipMaxAmountPositive(&suite.BaseSuite, sponsoredAssetId, assetId)

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
}

func (suite *TransferWithSponsorshipTxSuite) TestTransferWithSponsorshipNegative() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
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
		tdmatrix := testdata.GetTransferWithSponsorshipMaxValuesDataNegative(&suite.BaseSuite, sponsoredAssetId, assetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsorship.SponsorshipEnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, false)
				errMsg := caseName + "Transfer with Sponsorship: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				transfer.WithSponsorshipNegativeChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	//actualTxIds should be empty
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferWithSponsorshipTxSuite) TestSponsoredTransferFeeNegative() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion,
			utl.TestChainID, testdata.Sponsor, 10000000000)
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
		tdmatrix := testdata.GetTransferWithSponsorshipDataNegative(&suite.BaseSuite, sponsoredAssetId, assetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsorship.SponsorshipEnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)
				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, !waitForTx)
				errMsg := caseName + "Transfer with Sponsorship: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				transfer.WithSponsorshipNegativeChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	//actualTxIds should be empty
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferWithSponsorshipTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipTxSuite))
}
