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
	"github.com/wavesplatform/gowaves/itests/utilities/sponsorship"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type TransferWithSponsorshipAPITxPositiveSuite struct {
	f.BaseSuite
}

func (suite *TransferWithSponsorshipAPITxPositiveSuite) TestTransferWithSponsorshipAPIPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor set up sponsorship for this token.
		sponsorship.EnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.DefaultMinSponsoredAssetFee)
		// Sponsor transfers all issued sponsored tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.Sponsor, testdata.RecipientSender)
		// Sponsor issues one more token.
		assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetID, testdata.Sponsor, testdata.RecipientSender)

		tdmatrix := testdata.GetSponsoredTransferPositiveData(&suite.BaseSuite, assetID, sponsoredAssetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
					caseName, tx.TxID.String())
				transfer.WithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipAPITxPositiveSuite) TestTransferWithSponsorshipToOneselfAPIPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor set up sponsorship for this token.
		sponsorship.EnableBroadcast(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.DefaultMinSponsoredAssetFee)
		// Sponsor issues one more token.
		assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)

		tdmatrix := testdata.GetSposoredTransferBySponsorAsSender(&suite.BaseSuite, sponsoredAssetID, assetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor transfers assets to himself, sponsored asset is used as fee asset.
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
					caseName, tx.TxID.String())
				transfer.WithSponsorshipPositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipAPITxPositiveSuite) TestFeeInWavesAccordingMinSponsoredAssetAPIPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor issues one more token.
		assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.Sponsor, testdata.RecipientSender)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetID, testdata.Sponsor, testdata.RecipientSender)

		tdmatrix := testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(&suite.BaseSuite,
			sponsoredAssetID, assetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor set up sponsorship for the token.
				sponsorship.EnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)

				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, true)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
					caseName, tx.TxID.String())
				transfer.WithSponsorshipMinAssetFeePositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipAPITxPositiveSuite) TestTransferWithSponsorshipMaxValuesAPIPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		// Fill Sponsor's Waves balance.
		transfer.TransferringFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, testdata.Sponsor, 100000000000000)
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor issues one more token.
		assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.Sponsor, testdata.RecipientSender)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetID, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipMaxAmountPositive(&suite.BaseSuite, sponsoredAssetID, assetID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor set up sponsorship for the token.
				sponsorship.EnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)

				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, true)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
					caseName, tx.TxID.String())
				transfer.WithSponsorshipMinAssetFeePositiveAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func TestTransferWithSponsorshipAPITxPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipAPITxPositiveSuite))
}

type TransferWithSponsorshipAPITxNegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *TransferWithSponsorshipAPITxNegativeSuite) TestTransferWithSponsorshipAPINegative() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor issues one more token.
		assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.Sponsor, testdata.RecipientSender)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetID, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipMaxValuesDataNegative(&suite.BaseSuite, sponsoredAssetID, assetID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor set up sponsorship for the token.
				sponsorship.EnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)

				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, false)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
					caseName, tx.TxID.String())
				txIds[name] = &tx.TxID
				transfer.WithSponsorshipNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	// actualTxIds should be empty.
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferWithSponsorshipAPITxNegativeSuite) TestSponsoredTransferFeeAPINegative() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)

	for _, v := range versions {
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, 10000000000)
		// Sponsor issues one more token.
		assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.Sponsor, testdata.RecipientSender)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetID, testdata.Sponsor, testdata.RecipientSender)
		tdmatrix := testdata.GetTransferWithSponsorshipDataNegative(&suite.BaseSuite, sponsoredAssetID, assetID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor set up sponsorship for the token.
				sponsorship.EnableBroadcast(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)

				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.BroadcastTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, false)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Transfer with Sponsorship tx: %s",
					caseName, tx.TxID.String())
				txIds[name] = &tx.TxID
				transfer.WithSponsorshipNegativeAPIChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	// actualTxIds should be empty.
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferWithSponsorshipAPITxNegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipAPITxNegativeSuite))
}
