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

type TransferWithSponsorshipApiTxSuite struct {
	f.BaseSuite
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipApiPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor set up sponsorship for this token
		sponsorship.SponsorshipEnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
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

		tdmatrix := testdata.GetSponsoredTransferPositiveData(&suite.BaseSuite, assetId, sponsoredAssetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				transfer.WithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipToOneselfApiPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor set up sponsorship for this token
		sponsorship.SponsorshipEnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.DefaultMinSponsoredAssetFee)
		//Sponsor issues one more token
		assetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)

		tdmatrix := testdata.GetSposoredTransferBySponsorAsSender(&suite.BaseSuite, sponsoredAssetId, assetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor transfers assets to himself, sponsored asset is used as fee asset
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				transfer.WithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestFeeInWavesAccordingMinSponsoredAssetApiPositive() {
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

		tdmatrix := testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(&suite.BaseSuite,
			sponsoredAssetId, assetId)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsorship.SponsorshipEnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, true)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				transfer.WithSponsorshipMinAssetFeePositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipMaxValuesApiPositive() {
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
		tdmatrix := testdata.GetTransferWithSponsorshipMaxAmountPositive(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsorship.SponsorshipEnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, true)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				transfer.WithSponsorshipMinAssetFeePositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipApiTxSuite) TestTransferWithSponsorshipApiNegative() {
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
		tdmatrix := testdata.GetTransferWithSponsorshipMaxValuesDataNegative(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsorship.SponsorshipEnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, false)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				transfer.WithSponsorshipNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	//actualTxIds should be empty
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferWithSponsorshipApiTxSuite) TestSponsoredTransferFeeApiNegative() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)

	for _, v := range versions {
		//Sponsor creates a new token
		sponsoredAssetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, 10000000000)
		//Sponsor issues one more token
		assetId := issue.IssueAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetId, testdata.Sponsor, testdata.RecipientSender)
		//Sponsor transfers all issued tokens to RecipientSender
		transfer.TransferAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetId, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipDataNegative(&suite.BaseSuite, sponsoredAssetId, assetId)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//Sponsor set up sponsorship for the token
				sponsorship.SponsorshipEnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetId, td.MinSponsoredAssetFee)

				//RecipientSender transfers assets to Recipient specifying fee in the sponsored asset
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, false)
				errMsg := caseName + "Broadcast Transfer with Sponsorship tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				transfer.WithSponsorshipNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	//actualTxIds should be empty
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferWithSponsorshipApiTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipApiTxSuite))
}
