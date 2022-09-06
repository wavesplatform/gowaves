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

/*type IssueTestData struct {
	Account    config.AccountInfo
	AssetName  string
	AssetDesc  string
	Quantity   uint64
	Decimals   byte
	Reissuable bool
	Fee        uint64
	Timestamp  uint64
	ChainID    proto.Scheme
	Expected   map[string]string
}*/

type IssueTxSuite struct {
	fixtures.BaseSuite
}

/*func NewIssueTestData(account config.AccountInfo, assetName string, assetDesc string, quantity uint64, decimals byte, reissuable bool, fee uint64,
	timestamp uint64, chainID proto.Scheme, expected map[string]string) *IssueTestData {
	return &IssueTestData{
		Account:    account,
		AssetName:  assetName,
		AssetDesc:  assetDesc,
		Quantity:   quantity,
		Decimals:   decimals,
		Reissuable: reissuable,
		Fee:        fee,
		Timestamp:  timestamp,
		ChainID:    chainID,
		Expected:   expected,
	}
}

func getCurrentTimestampInMs() uint64 {
	return uint64(time.Now().UnixNano() / 1000000)
}

func getAccount(suite *IssueTxSuite, i int) config.AccountInfo {
	return suite.Cfg.Accounts[i]
}

func getAvalibleBalanceInWaves(suite *IssueTxSuite, address proto.WavesAddress) int64 {
	return suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func getAssetBalance(suite *IssueTxSuite, address proto.WavesAddress, id []byte) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, id).GetAmount()
}

func dataChangedTimestamp(td IssueTestData) IssueTestData {
	return *NewIssueTestData(td.Account, td.AssetName, td.AssetDesc, td.Quantity, td.Decimals, td.Reissuable, td.Fee,
		getCurrentTimestampInMs(), td.ChainID, td.Expected)
}

func getInvalidTxIdsInBlockchain(suite *IssueTxSuite, ids []*crypto.Digest, timeout time.Duration) []*crypto.Digest {
	time.Sleep(timeout)
	for _, id := range ids {
		_, _, errGo := suite.Clients.GoClients.HttpClient.TransactionInfoRaw(*id)
		_, _, errScala := suite.Clients.ScalaClients.HttpClient.TransactionInfoRaw(*id)
		if (errGo != nil) && (errScala != nil) {
			ids[0] = nil
			if len(ids) > 1 {
				copy(ids[0:], ids[1:])
				ids = ids[:len(ids)-1]
			} else if len(ids) == 1 {
				ids = nil
			}
		}
	}
	return ids
}*/

/*func getPositiveDataMatrix(suite *IssueTxSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Min values, empty description, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"",
			1,
			0,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "1",
			}),
		"Middle values, special symbols in desc, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"testtest",
			"~!|#$%^&*()_+=\\\";:/?><|\\\\][{}",
			100000000000,
			4,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "100000000000",
			}),
		"Max values, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"testtesttestest",
			"testtesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttest"+
				"testtestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestteste"+
				"sttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttes"+
				"ttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestestt"+
				"esttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestte"+
				"stesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttest"+
				"testtestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestteste"+
				"sttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttes"+
				"ttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestestt"+
				"esttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestte"+
				"sttesttesttestt",
			9223372036854775807,
			8,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "9223372036854775807",
			}),
		"NFT, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			1,
			0,
			false,
			100000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000",
				"asset balance":      "1",
			}),
		"Not reissuable, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"testtest",
			"testtesttestest",
			100000000000,
			4,
			false,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "100000000000",
			}),
	}
	return t
}

func getNegativeDataMatrix(suite *IssueTxSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Invalid asset name (len < min), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"tes",
			"t",
			1,
			0,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid asset name (len > max), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"testtesttesttestt",
			"test",
			10000,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Empty string in asset name, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"",
			"test",
			10000,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Special symbols in asset name, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"~!|#$%^&*()_+=\\\";:/?><|\\\\][{}",
			"test",
			10000,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid encoding in asset name, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"\\u0061\\u0073\\u0073\\u0065\\u0074",
			"test",
			10000,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		//Error in Node Go
		/*"Invalid encoding in asset description, not miner": *NewIssueTestData(
		getAccount(suite, 2),
		"test",
		"\\u0061\\u0073\\u0073\\u0065\\u0074",
		10000,
		2,
		true,
		100000000,
		getCurrentTimestampInMs(),
		'L',
		map[string]string{
			"err go msg":         "transactions does not exist",
			"err scala msg":      "transactions does not exist",
			"waves diff balance": "0",
			"asset balance":      "0",
		}),
		"Invalid asset description (len > max), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"testtesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttest"+
				"testtestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestteste"+
				"sttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttes"+
				"ttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestestt"+
				"esttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestte"+
				"stesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttest"+
				"testtestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestteste"+
				"sttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttes"+
				"ttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestestt"+
				"esttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestte"+
				"sttesttesttesttt",
			10000,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token quantity (quantity < min), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			0,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token quantity (quantity > max), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			9223372036854775808,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token decimals (decimals > max), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			100000,
			9,
			true,
			100000000,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid fee (fee > max), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			9223372036854775808,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid fee (fee < min), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			0,
			getCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			100000000,
			1,
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			100000000,
			9223372036854775807,
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Creating a token when there are not enough funds on the account balance": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			uint64(100000000+getAvalibleBalanceInWaves(suite, getAccount(suite, 2).Address)),
			9223372036854775807,
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
	}
	return t
}*/

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
