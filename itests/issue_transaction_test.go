package itests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueTxSuite struct {
	f.BaseSuite
}

func (suite *IssueTxSuite) Test_IssueTxPositive() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Minute
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			txID, errGo, errScala := issue_utilities.Issue(&suite.BaseSuite, td, i, timeout)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, txID)

			utl.ExistenceTxInfoCheck(suite.T(), errGo, errScala, name, "version", i, txID.String())
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name, "version", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name, "version", i)
		}
	}

}

func (suite *IssueTxSuite) Test_IssueTxWithSameDataPositive() {
	versions := testdata.GetVersions()
	timeout := 1 * time.Minute
	for _, i := range versions {
		tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			txID1, errGo1, errScala1 := issue_utilities.Issue(&suite.BaseSuite, td, i, timeout)
			txID2, errGo2, errScala2 := issue_utilities.Issue(
				&suite.BaseSuite, testdata.DataChangedTimestamp(&td), i, timeout)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			actualAsset1BalanceGo, actualAsset1BalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, txID1)
			actualAsset2BalanceGo, actualAsset2BalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, txID2)
			//Since the issue transaction is called twice, the expected balance difference also is doubled.
			expectedDiffBalanceInWaves := 2 * td.Expected.WavesDiffBalance

			utl.ExistenceTxInfoCheck(suite.T(), errGo1, errScala1, name, "version", i, txID1.String())
			utl.ExistenceTxInfoCheck(suite.T(), errGo2, errScala2, name, "version", i, txID2.String())
			utl.WavesDiffBalanceCheck(
				suite.T(), expectedDiffBalanceInWaves, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAsset1BalanceGo, actualAsset1BalanceScala, name, "version", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAsset2BalanceGo, actualAsset2BalanceScala, name, "version", i)
		}
	}
}

func (suite *IssueTxSuite) Test_IssueTxNegative() {
	versions := testdata.GetVersions()
	timeout := 3 * time.Second
	txIds := make(map[string]*crypto.Digest)
	for _, i := range versions {
		tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			txID, errGo, errScala := issue_utilities.Issue(&suite.BaseSuite, td, i, timeout)
			txIds[name] = &txID

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
				&suite.BaseSuite, td.Account.Address, txID)

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, errGo, errScala, name, "version", i)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name, "version", i)
			utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name, "version", i)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestIssueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxSuite))
}
