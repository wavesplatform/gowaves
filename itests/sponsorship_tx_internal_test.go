package itests

import (
	"math/rand"
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

type SponsorshipTxSuite struct {
	f.BaseSuite
}

func sponsorshipPositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SponsorshipTestData[testdata.SponsorshipExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func sponsorshipNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SponsorshipTestData[testdata.SponsorshipExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func (suite *SponsorshipTxSuite) TestSponsorshipTxPositive() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Sponsorship Tx tests in short mode")
	}
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetSponsorshipPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					sponsor_utilities.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Sponsorship tx: " + tx.TxID.String()
				sponsorshipPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *SponsorshipTxSuite) TestSponsorshipTxMaxValues() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Sponsorship Tx tests in short mode")
	}
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
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
					sponsor_utilities.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, waitForTx)
				errMsg := caseName + "Sponsorship tx: " + tx.TxID.String()
				sponsorshipPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *SponsorshipTxSuite) TestSponsorshipDisabledTx() {
	if testing.Short() {
		suite.T().Skip("skipping long positive Sponsorship Tx tests in short mode")
	}
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	name := "Sponsorship Enabled/Disabled"
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		sponsorship := testdata.GetSponsorshipEnabledDisabledData(&suite.BaseSuite, itx.TxID)
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//switch on sponsorship
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsor_utilities.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, sponsorship.Enabled, v, waitForTx)
			errMsg := caseName + "Sponsorship tx: " + tx.TxID.String()
			sponsorshipPositiveChecks(suite.T(), tx, sponsorship.Enabled, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)

			//switch off sponsorship
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset =
				sponsor_utilities.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, sponsorship.Disabled, v, waitForTx)
			errMsg = caseName + "Sponsorship Disabled tx: " + tx.TxID.String()
			sponsorshipPositiveChecks(suite.T(), tx, sponsorship.Disabled, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SponsorshipTxSuite) TestSponsorshipTxNegative() {
	if testing.Short() {
		suite.T().Skip("skipping long negative Sponsorship Tx tests in short mode")
	}
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)

	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetSponsorshipNegativeDataMatrix(&suite.BaseSuite, itx.TxID)

		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
					sponsor_utilities.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, !waitForTx)
				errMsg := caseName + "Sponsorship tx: " + tx.TxID.String()
				txIds[name] = &tx.TxID
				sponsorshipNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
					actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SponsorshipTxSuite) Test_SponsorshipForSmartAssetNegative() {
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		smart := testdata.GetCommonIssueData(&suite.BaseSuite).Smart
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, smart, v, waitForTx)
		td := testdata.GetSponsorshipForSmartAssetData(&suite.BaseSuite, itx.TxID).Enabled
		name := "Check sponsorship for smart asset"
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsor_utilities.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, td, v, !waitForTx)
			errMsg := caseName + "Sponsorship tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			sponsorshipNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *SponsorshipTxSuite) TestSponsorshipTxSmokePositive() {
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	waitForTx := true
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randV, waitForTx)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetSponsorshipPositiveDataMatrix(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsor_utilities.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, td, randV, waitForTx)
			errMsg := caseName + "Sponsorship tx: " + tx.TxID.String()
			sponsorshipPositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
}

func (suite *SponsorshipTxSuite) TestSponsorshipTxSmokeNegative() {
	versions := sponsor_utilities.GetVersions(&suite.BaseSuite)
	randV := versions[rand.Intn(len(versions))]
	waitForTx := true
	txIds := make(map[string]*crypto.Digest)
	reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
	itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, randV, waitForTx)
	tdmatrix := utl.GetRandomValueFromMap(testdata.GetSponsorshipNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
	for name, td := range tdmatrix {
		caseName := utl.GetTestcaseNameWithVersion(name, randV)
		suite.Run(caseName, func() {
			tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset :=
				sponsor_utilities.SendSponsorshipTxAndGetBalances(&suite.BaseSuite, td, randV, !waitForTx)
			errMsg := caseName + "Sponsorship tx: " + tx.TxID.String()
			txIds[name] = &tx.TxID
			sponsorshipNegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves,
				actualDiffBalanceInAsset, errMsg)
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestSponsorshipTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SponsorshipTxSuite))
}
