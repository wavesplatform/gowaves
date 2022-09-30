package itests

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type IssueTxApiSuite struct {
	issue_utilities.CommonIssueTxSuite
}

func (suite *IssueTxApiSuite) Test_IssueTxApiPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)

		tx, respGo, errGo, respScala, errScala := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)

		suite.Equalf(respGo.StatusCode, 200, "Status Code for Node Go Response not equal 200")
		suite.Equalf(respScala.StatusCode, 200, "Status Code for Node Scala Response not equal 200")

		currentBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAssetBalance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, tx.ID.Bytes())

		expectedDiffBalanceInWaves, err := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		suite.NoErrorf(err, "failed to parse expected diff balance")
		expectedAssetBalance, err := strconv.ParseInt(td.Expected["asset balance"], 10, 64)
		suite.NoErrorf(err, "failed to parse expected asset balance")

		suite.NoErrorf(errGo, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(expectedDiffBalanceInWaves, actualDiffBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(expectedAssetBalance, actualAssetBalance, "Node Go in case: \"%s\"", name)

	}
}

func (suite *IssueTxSuite) Test_IssueTxApiWithSameDataPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)

		tx1, respGo1, errGo1, respScala1, errScala1 := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)
		tx2, respGo2, errGo2, respScala2, errScala2 := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, testdata.DataChangedTimestamp(&td), timeout)

		suite.Equalf(respGo1.StatusCode, 200, "Status Code for Node Go Response not equal 200")
		suite.Equalf(respGo2.StatusCode, 200, "Status Code for Node Go Response not equal 200")
		suite.Equalf(respScala1.StatusCode, 200, "Status Code for Node Scala Response not equal 200")
		suite.Equalf(respScala2.StatusCode, 200, "Status Code for Node Scala Response not equal 200")

		currentBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAsset1Balance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, tx1.ID.Bytes())
		actualAsset2Balance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, tx2.ID.Bytes())
		diffBalanceInWaves, err := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		suite.NoErrorf(err, "failed to parse expected diff balance")
		expectedDiffBalanceInWaves := 2 * diffBalanceInWaves
		expectedAssetBalance, err := strconv.ParseInt(td.Expected["asset balance"], 10, 64)
		suite.NoErrorf(err, "failed to parse expected asset balance")

		suite.NoErrorf(errGo1, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala1, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.NoErrorf(errGo2, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala2, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(expectedDiffBalanceInWaves, actualDiffBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(expectedAssetBalance, actualAsset1Balance, "Node go in case: \"%s\"", name)
		suite.Equalf(expectedAssetBalance, actualAsset2Balance, "Node Go in case: \"%s\"", name)
	}
}

func (suite *IssueTxSuite) Test_IssueTxApiNegative() {
	tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
	timeout := 3 * time.Second
	txIds := make(map[string]*crypto.Digest)

	for name, td := range tdmatrix {

		initBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)

		tx, respGo, errGo, respScala, errScala := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)
		suite.Equalf(respGo.StatusCode, 200, "Status Code for Node Go Response not equal 200")
		suite.Equalf(respScala.StatusCode, 200, "Status Code for Node Scala Response not equal 200")

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

func TestIssueTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxApiSuite))
}
