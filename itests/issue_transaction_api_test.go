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

		brdCstTx, errWtGo, errWtScala := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)

		suite.Equalf(brdCstTx.ResponseGo.StatusCode, 200, "Status Code for Node Go Response not equal 200")
		suite.Equalf(brdCstTx.ResponseScala.StatusCode, 200, "Status Code for Node Scala Response not equal 200")

		currentBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAssetBalance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, brdCstTx.TxID.Bytes())

		suite.NoErrorf(errWtGo, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errWtScala, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(td.Expected.WavesDiffBalance, actualDiffBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(td.Expected.AssetBalance, actualAssetBalance, "Node Go in case: \"%s\"", name)
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiWithSameDataPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)

		brdCstTx1, errWtGo1, errWtScala1 := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)
		brdCstTx2, errWtGo2, errWtScala2 := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, testdata.DataChangedTimestamp(&td), timeout)

		suite.Equalf(brdCstTx1.ResponseGo.StatusCode, 200, "Status Code for Node Go Response not equal 200")
		suite.Equalf(brdCstTx2.ResponseGo.StatusCode, 200, "Status Code for Node Go Response not equal 200")
		suite.Equalf(brdCstTx1.ResponseScala.StatusCode, 200, "Status Code for Node Scala Response not equal 200")
		suite.Equalf(brdCstTx2.ResponseScala.StatusCode, 200, "Status Code for Node Scala Response not equal 200")

		currentBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAsset1Balance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, brdCstTx1.TxID.Bytes())
		actualAsset2Balance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, brdCstTx2.TxID.Bytes())
		// TODO(nickeskov): explain why we multiply expected value two times
		expectedDiffBalanceInWaves := 2 * td.Expected.WavesDiffBalance

		suite.NoErrorf(errWtGo1, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errWtScala1, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.NoErrorf(errWtGo2, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errWtScala2, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(expectedDiffBalanceInWaves, actualDiffBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(td.Expected.AssetBalance, actualAsset1Balance, "Node go in case: \"%s\"", name)
		suite.Equalf(td.Expected.AssetBalance, actualAsset2Balance, "Node Go in case: \"%s\"", name)
	}
}

func (suite *IssueTxApiSuite) Test_IssueTxApiNegative() {
	tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
	timeout := 3 * time.Second
	txIds := make(map[string]*crypto.Digest)

	for name, td := range tdmatrix {

		initBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)

		brdCstTx, errWtGo, errWtScala := issue_utilities.IssueBroadcast(&suite.CommonIssueTxSuite, td, timeout)
		suite.Equalf(500, brdCstTx.ResponseGo.StatusCode, "Case: \"%s\", Status Code for Node Go Response not equal 500", name)
		suite.Equalf(400, brdCstTx.ResponseScala.StatusCode, "Case: \"%s\", Status Code for Node Scala Response not equal 400", name)
		suite.ErrorContainsf(brdCstTx.ErrorBrdCstGo, td.Expected["err brdcst msg go"], "Node Go in case: \"%s\"", name)
		suite.ErrorContainsf(brdCstTx.ErrorBrdCstScala, td.Expected["err brdcst msg scala"], "Node Scala in case: \"%s\"", name)

		txIds[name] = &brdCstTx.TxID

		currentBalanceInWaves := utl.GetAvalibleBalanceInWavesGo(&suite.BaseSuite, td.Account.Address)
		actualBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAssetBalance := utl.GetAssetBalanceGo(&suite.BaseSuite, td.Account.Address, brdCstTx.TxID.Bytes())

		expectedBalanceInWaves, err := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		suite.NoErrorf(err, "failed to parse expected diff balance")
		expectedAssetBalance, err := strconv.ParseInt(td.Expected["asset balance"], 10, 64)
		suite.NoErrorf(err, "failed to parse expected asset balance")

		suite.ErrorContainsf(errWtGo, td.Expected["err go msg"], "Node Go in case: \"%s\"", name)
		suite.ErrorContainsf(errWtScala, td.Expected["err scala msg"], "Node Scala in case: \"%s\"", name)
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
