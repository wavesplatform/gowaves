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

type SponsorshipTxApiSuite struct {
	f.BaseSuite
}

func sponsorshipPositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SponsorshipTestData[testdata.SponsorshipExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func sponsorshipNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SponsorshipTestData[testdata.SponsorshipExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
		tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipTxApiPositive() {
	utl.SkipLongTest(suite.T())
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetSponsorshipPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					sponsor_utilities.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Sponsorship tx: " + tx.TxID.String()
				sponsorshipPositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipTxApiMaxValues() {
	utl.SkipLongTest(suite.T())
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		n := transfer_utilities.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		itxID := issue_utilities.IssueAssetAmount(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultSenderNotMiner, utl.MaxAmount)
		transfer_utilities.TransferAssetAmount(&suite.BaseSuite, v, utl.TestChainID, itxID,
			utl.DefaultSenderNotMiner, n)
		tdmatrix := testdata.GetSponsorshipMaxValuesPositive(&suite.BaseSuite, itxID, n)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					sponsor_utilities.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, true)
				errMsg := caseName + "Broadcast Sponsorship tx: " + tx.TxID.String()
				sponsorshipPositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipDisabledTxApi() {
	utl.SkipLongTest(suite.T())
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	name := "Sponsorship Enabled/Disabled"
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		sponsorship := testdata.GetSponsorshipEnabledDisabledData(&suite.BaseSuite, itx.TxID)
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//switch on sponsorship
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsor_utilities.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, sponsorship.Enabled,
					v, true)
			errMsg := caseName + "Broadcast Sponsorship tx: " + tx.TxID.String()
			sponsorshipPositiveAPIChecks(suite.T(), tx, sponsorship.Enabled, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)

			//switch off sponsorship
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset =
				sponsor_utilities.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, sponsorship.Disabled,
					v, true)
			errMsg = caseName + "Broadcast Sponsorship Disabled tx: " + tx.TxID.String()
			sponsorshipPositiveAPIChecks(suite.T(), tx, sponsorship.Disabled, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipTxApiNegative() {
	utl.SkipLongTest(suite.T())
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetSponsorshipNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					sponsor_utilities.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, false)
				errMsg := caseName + "Broadcast Sponsorship tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				sponsorshipNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SponsorshipTxApiSuite) Test_SponsorshipForSmartAssetApiNegative() {
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, smart, v, true)
		td := testdata.GetSponsorshipForSmartAssetData(&suite.BaseSuite, itx.TxID).Enabled
		name := "Check sponsorship for smart asset"
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsor_utilities.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, false)
			errMsg := caseName + "Broadcast Sponsorship tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			sponsorshipNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipTxApiSmokePositive() {
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetSponsorshipPositiveDataMatrix(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsor_utilities.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, randV, true)
			errMsg := caseName + "Broadcast Sponsorship tx: " + tx.TxID.String()
			sponsorshipPositiveAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SponsorshipTxApiSuite) TestSponsorshipTxApiSmokeNegative() {
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	txIds := make(map[string]*crypto.Digest)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueBroadcastWithTestData(&suite.BaseSuite, reissuable, randV, true)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetSponsorshipNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsor_utilities.BroadcastSponsorshipTxAndGetBalances(&suite.BaseSuite, td, randV, false)
			errMsg := caseName + "Broadcast Sponsorship tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			sponsorshipNegativeAPIChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
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
