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

type TransferWithSponsorshipTxPositiveSuite struct {
	f.BaseSuite
}

func (suite *TransferWithSponsorshipTxPositiveSuite) TestTransferWithSponsorshipPositive() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor set up sponsorship for this token.
		sponsorship.EnableSend(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
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
		// Test data.
		tdmatrix := testdata.GetSponsoredTransferPositiveData(&suite.BaseSuite, assetID, sponsoredAssetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Transfer with Sponsorship tx: %s", caseName, tx.TxID.String())
				// Checks.
				transfer.WithSponsorshipPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxPositiveSuite) TestTransferWithSponsorshipToOneselfPositive() {
	waitForTx := true
	versions := transfer.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor set up sponsorship for this token.
		sponsorship.EnableSend(&suite.BaseSuite, testdata.SponsorshipMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.DefaultMinSponsoredAssetFee)
		// Sponsor issues one more token.
		assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Test data.
		tdmatrix := testdata.GetSposoredTransferBySponsorAsSender(&suite.BaseSuite, sponsoredAssetID, assetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor transfers assets to himself, sponsored asset is used as fee asset.
				// Sponsor balance in Waves decreases by amount
				// feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee.
				// Sponsor asset balance does not change.
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)
				errMsg := fmt.Sprintf("Case: %s; Transfer with Sponsorship tx: %s", caseName, tx.TxID.String())
				// Checks
				transfer.WithSponsorshipPositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxPositiveSuite) TestFeeInWavesAccordingMinSponsoredAssetPositive() {
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
		// Test data.
		tdmatrix := testdata.GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(&suite.BaseSuite,
			sponsoredAssetID, assetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor set up sponsorship for the token.
				sponsorship.EnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)
				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, true)
				errMsg := fmt.Sprintf("Case: %s; Transfer with Sponsorship tx: %s", caseName, tx.TxID.String())
				transfer.WithSponsorshipMinAssetFeePositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func (suite *TransferWithSponsorshipTxPositiveSuite) TestTransferWithSponsorshipMaxValuesPositive() {
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
		// Test data.
		tdmatrix := testdata.GetTransferWithSponsorshipMaxAmountPositive(&suite.BaseSuite, sponsoredAssetID, assetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor set up sponsorship for the token.
				sponsorship.EnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)
				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, true)
				errMsg := fmt.Sprintf("Case: %s; Transfer with Sponsorship tx: %s", caseName, tx.TxID.String())
				transfer.WithSponsorshipMinAssetFeePositiveChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
}

func TestTransferWithSponsorshipTxPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipTxPositiveSuite))
}

type TransferWithSponsorshipTxNegativeSuite struct {
	f.BaseNegativeSuite
}

func (suite *TransferWithSponsorshipTxNegativeSuite) TestTransferWithSponsorshipNegative() {
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
		// Test data.
		tdmatrix := testdata.GetTransferWithSponsorshipMaxValuesDataNegative(&suite.BaseSuite, sponsoredAssetID, assetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor set up sponsorship for the token.
				sponsorship.EnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)

				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, false)
				errMsg := fmt.Sprintf("Case: %s; Transfer with Sponsorship tx: %s", caseName, tx.TxID.String())
				txIds[name] = &tx.TxID
				transfer.WithSponsorshipNegativeChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	// actualTxIds should be empty.
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *TransferWithSponsorshipTxNegativeSuite) TestSponsoredTransferFeeNegative() {
	versions := transfer.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		// Sponsor creates a new token.
		sponsoredAssetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion,
			utl.TestChainID, testdata.Sponsor, 10000000000)
		// Sponsor issues one more token.
		assetID := issue.IssuedAssetAmount(&suite.BaseSuite, testdata.IssueMaxVersion, utl.TestChainID,
			testdata.Sponsor, utl.MaxAmount)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			sponsoredAssetID, testdata.Sponsor, testdata.RecipientSender)
		// Sponsor transfers all issued tokens to RecipientSender.
		transfer.TransferringAssetAmount(&suite.BaseSuite, testdata.TransferMaxVersion, utl.TestChainID,
			assetID, testdata.Sponsor, testdata.RecipientSender)
		// Test data.
		tdmatrix := testdata.GetTransferWithSponsorshipDataNegative(&suite.BaseSuite, sponsoredAssetID, assetID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				// Sponsor set up sponsorship for the token.
				sponsorship.EnableSend(&suite.BaseSuite, v,
					td.TransferTestData.ChainID, sponsoredAssetID, td.MinSponsoredAssetFee)
				// RecipientSender transfers assets to Recipient specifying fee in the sponsored asset.
				tx, diffBalances := transfer.SendTransferTxAndGetBalances(
					&suite.BaseSuite, td.TransferTestData, v, !waitForTx)
				errMsg := fmt.Sprintf("Case: %s; Transfer with Sponsorship tx: %s", caseName, tx.TxID.String())
				txIds[name] = &tx.TxID
				transfer.WithSponsorshipNegativeChecks(suite.T(), tx, td, diffBalances, errMsg)
			})
		}
	}
	// actualTxIds should be empty.
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestTransferWithSponsorshipTxNegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferWithSponsorshipTxNegativeSuite))
}
