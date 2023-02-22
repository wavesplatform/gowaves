package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/sponsor_utilities"
)

type SponsorshipTxSuite struct {
	f.BaseSuite
}

func (suite *SponsorshipTxSuite) TestSponsorshipTxPositive() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetSponsorshipPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := sponsor_utilities.SendSponsorshipTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *SponsorshipTxSuite) TestSponsorshipTxMaxValues() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSendWithTestData(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetSponsorshipMaxValuesPositive(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := sponsor_utilities.SendSponsorshipTxAndGetBalances(
					&suite.BaseSuite, td, v, waitForTx)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, "Sponsorship: "+tx.TxID.String(),
					utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func TestSponsorshipTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SponsorshipTxSuite))
}
