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

type SponsorshipTxApiSuite struct {
	f.BaseSuite
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipTxApiPositive() {
	versions := sponsorship.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
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
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipTxApiMaxValues() {
	versions := sponsorship.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		n := transfer.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		itxID := issue.IssuedAssetAmount(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultSenderNotMiner, utl.MaxAmount)
		transfer.TransferringAssetAmount(&suite.BaseSuite, v, utl.TestChainID, itxID,
			utl.DefaultSenderNotMiner, n)
		tdmatrix := testdata.GetSponsorshipMaxValuesPositive(&suite.BaseSuite, itxID, n)
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
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipDisabledTxApi() {
	versions := sponsorship.GetVersions(&suite.BaseSuite)
	name := "Sponsorship Enabled/Disabled"
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.BroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		sponsorshipData := testdata.GetSponsorshipEnabledDisabledData(&suite.BaseSuite, itx.TxID)
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//switch on sponsorship
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsorship.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, sponsorshipData.Enabled,
					v, true)
			errMsg := caseName + "Broadcast Sponsorship tx: " + tx.TxID.String()
			sponsorship.PositiveAPIChecks(suite.T(), tx, sponsorshipData.Enabled, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)

			//switch off sponsorship
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset =
				sponsorship.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, sponsorshipData.Disabled,
					v, true)
			errMsg = fmt.Sprintf("Case: %s; Broadcast Sponsorship tx: %s", caseName, tx.TxID.String())
			sponsorship.PositiveAPIChecks(suite.T(), tx, sponsorshipData.Disabled, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipTxApiNegative() {
	versions := sponsorship.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.BroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetSponsorshipNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					sponsorship.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, false)
				errMsg := fmt.Sprintf("Case: %s; Broadcast Sponsorship tx: %s", caseName, tx.TxID.String())
				txIds[name] = &tx.TxID
				sponsorship.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SponsorshipTxApiSuite) Test_SponsorshipForSmartAssetApiNegative() {
	versions := sponsorship.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue.BroadcastWithTestData(&suite.BaseSuite, smart, v, true)
		td := testdata.GetSponsorshipForSmartAssetData(&suite.BaseSuite, itx.TxID).Enabled
		name := "Check sponsorship for smart asset"
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsorship.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, false)
			errMsg := fmt.Sprintf("Case: %s; Broadcast Sponsorship tx: %s", caseName, tx.TxID.String())
			txIds[name] = &tx.TxID
			sponsorship.NegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestSponsorshipTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SponsorshipTxApiSuite))
}
