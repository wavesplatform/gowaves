package itests

import (
	"strconv"
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
		initBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)

		tx, errGo, errScala := issue_utilities.Issue(&suite.CommonIssueTxSuite, td, timeout)

		currentBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAssetBalance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, tx.ID.Bytes())

		suite.NoErrorf(errGo, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(td.Expected.WavesDiffBalance, actualDiffBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(td.Expected.AssetBalance, actualAssetBalance, "Node Go in case: \"%s\"", name)
	}
}

func (suite *IssueTxSuite) Test_IssueTxWithSameDataPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)

		tx1, errGo1, errScala1 := issue_utilities.Issue(&suite.CommonIssueTxSuite, td, timeout)
		tx2, errGo2, errScala2 := issue_utilities.Issue(&suite.CommonIssueTxSuite, testdata.DataChangedTimestamp(&td), timeout)

		currentBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAsset1Balance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, tx1.ID.Bytes())
		actualAsset2Balance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, tx2.ID.Bytes())
		// TODO(nickeskov): explain why we multiply expected value two times
		expectedDiffBalanceInWaves := 2 * td.Expected.WavesDiffBalance

		suite.NoErrorf(errGo1, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala1, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.NoErrorf(errGo2, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala2, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(expectedDiffBalanceInWaves, actualDiffBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(td.Expected.AssetBalance, actualAsset1Balance, "Node go in case: \"%s\"", name)
		suite.Equalf(td.Expected.AssetBalance, actualAsset2Balance, "Node Go in case: \"%s\"", name)
	}
}

func (suite *IssueTxSuite) Test_IssueTxNegative() {
	tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
	timeout := 3 * time.Second
	txIds := make(map[string]*crypto.Digest)

	for name, td := range tdmatrix {

		initBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)

		tx, errGo, errScala := issue_utilities.Issue(&suite.CommonIssueTxSuite, td, timeout)
		txIds[name] = tx.ID

		currentBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)
		actualBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAssetBalance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, tx.ID.Bytes())

		expectedBalanceInWaves, err := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		suite.NoErrorf(err, "failed to parse expected diff balance")
		expectedAssetBalance, err := strconv.ParseInt(td.Expected["asset balance"], 10, 64)
		suite.NoErrorf(err, "failed to parse expected asset balance")

		suite.ErrorContainsf(errGo, td.Expected["err go msg"], "Node Go in case: \"%s\"", name)
		suite.ErrorContainsf(errScala, td.Expected["err scala msg"], "Node Scala in case: \"%s\"", name)
		suite.Equalf(expectedBalanceInWaves, actualBalanceInWaves, "Expected balance in Waves Node Go in case: \"%s\"", name)
		suite.Equalf(expectedAssetBalance, actualAssetBalance, "Expected Asset balance Node Go in case: \"%s\"", name)
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
	suite.Equalf(0, len(actualTxIds), "IDs: %#v", actualTxIds)
}

func TestIssueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxSuite))
}
