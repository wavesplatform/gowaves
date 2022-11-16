package itests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"

	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueTxSuite struct {
	issue_utilities.CommonIssueTxSuite
}

func (suite *IssueTxSuite) Test_IssueTxPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)

		tx, errGo, errScala := issue_utilities.Issue(&suite.CommonIssueTxSuite, td, timeout)

		actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
			&suite.BaseSuite, td.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)

		actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, *tx.ID)

		utl.ExistenceTxInfoCheck(suite.T(), errGo, errScala, name)
		utl.WavesDiffBalanceCheck(
			suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name)
	}
}

func (suite *IssueTxSuite) Test_IssueTxWithSameDataPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)

		tx1, errGo1, errScala1 := issue_utilities.Issue(&suite.CommonIssueTxSuite, td, timeout)
		tx2, errGo2, errScala2 := issue_utilities.Issue(
			&suite.CommonIssueTxSuite, testdata.DataChangedTimestamp(&td), timeout)

		actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
			&suite.BaseSuite, td.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)

		actualAsset1BalanceGo, actualAsset1BalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, *tx1.ID)
		actualAsset2BalanceGo, actualAsset2BalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, *tx2.ID)
		//Since the issue transaction is called twice, the expected balance difference also is doubled.
		expectedDiffBalanceInWaves := 2 * td.Expected.WavesDiffBalance

		utl.ExistenceTxInfoCheck(suite.T(), errGo1, errScala1, name)
		utl.ExistenceTxInfoCheck(suite.T(), errGo2, errScala2, name)
		utl.WavesDiffBalanceCheck(
			suite.T(), expectedDiffBalanceInWaves, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
		utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAsset1BalanceGo, actualAsset1BalanceScala, name)
		utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAsset2BalanceGo, actualAsset2BalanceScala, name)
	}
}

func (suite *IssueTxSuite) Test_IssueTxNegative() {
	tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
	timeout := 3 * time.Second
	txIds := make(map[string]*crypto.Digest)

	for name, td := range tdmatrix {

		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)

		tx, errGo, errScala := issue_utilities.Issue(&suite.CommonIssueTxSuite, td, timeout)
		txIds[name] = tx.ID

		actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
			&suite.BaseSuite, td.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)

		actualAssetBalanceGo, actualAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, *tx.ID)

		utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, errGo, errScala, name)
		utl.WavesDiffBalanceCheck(
			suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		utl.AssetBalanceCheck(suite.T(), td.Expected.AssetBalance, actualAssetBalanceGo, actualAssetBalanceScala, name)
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestIssueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxSuite))
}
