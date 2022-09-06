package integration

import (
	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	"github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"strconv"
	"testing"
	"time"
)

type IssueTxSuite struct {
	fixtures.BaseSuite
}

func newSignIssueTransaction(suite *IssueTxSuite, testdata testdata.IssueTestData) *proto.IssueWithSig {
	tx := proto.NewUnsignedIssueWithSig(testdata.Account.PublicKey, testdata.AssetName,
		testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	suite.NoError(err, "failed to create proofs from signature")
	return tx
}

func sendAndWaitTransaction(suite *IssueTxSuite, tx *proto.IssueWithSig, timeout time.Duration) (error, error) {
	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.Conns.Reconnect(suite.T(), suite.Ports)
	suite.Conns.SendToEachNode(suite.T(), &txMsg)

	errGo, errScala := suite.Clients.WaitForTransaction(suite.T(), tx.ID, timeout)
	return errGo, errScala
}

func issue(suite *IssueTxSuite, testdata testdata.IssueTestData, timeout time.Duration) (*proto.IssueWithSig, error, error) {
	tx := newSignIssueTransaction(suite, testdata)
	errGo, errScala := sendAndWaitTransaction(suite, tx, timeout)
	return tx, errGo, errScala
}

func (suite *IssueTxSuite) Test_IssueTxPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWaves := utilities.GetAvalibleBalanceInWaves(&suite.BaseSuite, td.Account.Address)

		tx, errGo, errScala := issue(suite, td, timeout)

		currentBalanceInWaves := utilities.GetAvalibleBalanceInWaves(&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAssetBalance := utilities.GetAssetBalance(&suite.BaseSuite, td.Account.Address, tx.ID.Bytes())

		expectedDiffBalanceInWaves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expectedAssetBalance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.NoErrorf(errGo, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(expectedDiffBalanceInWaves, actualDiffBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(expectedAssetBalance, actualAssetBalance, "Node Go in case: \"%s\"", name)
	}
}

func (suite *IssueTxSuite) Test_IssueTxWithSameDataPositive() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		initBalanceInWaves := utilities.GetAvalibleBalanceInWaves(&suite.BaseSuite, td.Account.Address)

		tx1, errGo1, errScala1 := issue(suite, td, timeout)
		tx2, errGo2, errScala2 := issue(suite, testdata.DataChangedTimestamp(&td), timeout)

		currentBalanceInWaves := utilities.GetAvalibleBalanceInWaves(&suite.BaseSuite, td.Account.Address)
		actualDiffBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAsset1Balance := utilities.GetAssetBalance(&suite.BaseSuite, td.Account.Address, tx1.ID.Bytes())
		actualAsset2Balance := utilities.GetAssetBalance(&suite.BaseSuite, td.Account.Address, tx2.ID.Bytes())
		diffBalanceInWaves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expectedDiffBalanceInWaves := 2 * diffBalanceInWaves
		expectedAssetBalance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.NoErrorf(errGo1, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala1, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.NoErrorf(errGo2, "Node Go in case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(errScala2, "Node Scala in case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(expectedDiffBalanceInWaves, actualDiffBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(expectedAssetBalance, actualAsset1Balance, "Node go in case: \"%s\"", name)
		suite.Equalf(expectedAssetBalance, actualAsset2Balance, "Node Go in case: \"%s\"", name)
	}
}

func (suite *IssueTxSuite) Test_IssueTxNegative() {
	tdmatrix := testdata.GetNegativeDataMatrix(&suite.BaseSuite)
	timeout := 3 * time.Second

	var txIds []*crypto.Digest
	for name, td := range tdmatrix {
		initBalanceInWaves := utilities.GetAvalibleBalanceInWaves(&suite.BaseSuite, td.Account.Address)

		tx, errGo, errScala := issue(suite, td, timeout)
		txIds = append(txIds, tx.ID)

		currentBalanceInWaves := utilities.GetAvalibleBalanceInWaves(&suite.BaseSuite, td.Account.Address)
		actualBalanceInWaves := initBalanceInWaves - currentBalanceInWaves
		actualAssetBalance := utilities.GetAssetBalance(&suite.BaseSuite, td.Account.Address, tx.ID.Bytes())

		expectedBalanceInWaves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expectedAssetBalance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.ErrorContainsf(errGo, td.Expected["err go msg"], "Node Go in case: \"%s\"", name)
		suite.ErrorContainsf(errScala, td.Expected["err scala msg"], "Node Scala in case: \"%s\"", name)
		suite.Equalf(expectedBalanceInWaves, actualBalanceInWaves, "Node Go in case: \"%s\"", name)
		suite.Equalf(expectedAssetBalance, actualAssetBalance, "Node Go in case: \"%s\"", name)
	}
	suite.Equalf(0, len(utilities.GetInvalidTxIdsInBlockchain(&suite.BaseSuite, txIds, 15*timeout)), "IDs: %#v", txIds)
}

func TestIssueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IssueTxSuite))
}
