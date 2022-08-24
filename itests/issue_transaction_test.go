package integration_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"strconv"
	"time"
)

type IssueTestData struct {
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
}

func NewIssueTestData(account config.AccountInfo, assetName string, assetDesc string, quantity uint64, decimals byte, reissuable bool, fee uint64,
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

func getAccount(suite *ItestSuite, i int) config.AccountInfo {
	return suite.cfg.Accounts[i]
}

func getAvalibleBalanceInWaves(suite *ItestSuite, address proto.WavesAddress) int64 {
	return suite.clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func getAssetBalance(suite *ItestSuite, address proto.WavesAddress, id []byte) int64 {
	return suite.clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, id).GetAmount()
}

func testDataChangedTimestamp(td IssueTestData) IssueTestData {
	return *NewIssueTestData(td.Account, td.AssetName, td.AssetDesc, td.Quantity, td.Decimals, td.Reissuable, td.Fee,
		getCurrentTimestampInMs(), td.ChainID, td.Expected)
}

func getPositiveDataMatrix(suite *ItestSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Min values, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"t",
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
		"Middle values, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"testtest",
			"testtesttestest",
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

func getNegativeDataMatrix(suite *ItestSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Invalid name (len < min), not miner": *NewIssueTestData(
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
	}
	return t
}

func newSignIssueTransaction(suite *ItestSuite, testdata IssueTestData) *proto.IssueWithSig {
	tx := proto.NewUnsignedIssueWithSig(testdata.Account.PublicKey, testdata.AssetName,
		testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	suite.NoError(err, "failed to create proofs from signature")
	return tx
}

func sendAndWaitTransaction(suite *ItestSuite, tx *proto.IssueWithSig, timeout time.Duration) (error, error) {
	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.conns.SendToEachNode(suite.T(), &txMsg)

	err_go, err_scala := suite.clients.WaitForTransaction(suite.T(), tx.ID, timeout)
	return err_go, err_scala
}

func issue(suite *ItestSuite, testdata IssueTestData, timeout time.Duration) (*proto.IssueWithSig, error, error) {
	tx := newSignIssueTransaction(suite, testdata)
	err_go, err_scala := sendAndWaitTransaction(suite, tx, timeout)
	return tx, err_go, err_scala
}

func (suite *ItestSuite) Test_IssueTxPositive() {
	testdata := getPositiveDataMatrix(suite)
	timeout := 1 * time.Minute
	for _, td := range testdata {
		init_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)

		tx, err_go, err_scala := issue(suite, td, timeout)

		current_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)
		actual_diff_balance_in_waves := init_balance_in_waves - current_balance_in_waves
		actual_asset_balance := getAssetBalance(suite, td.Account.Address, tx.ID.Bytes())

		expected_diff_balance_in_waves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expected_asset_balance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.NoError(err_go, "Failed to get TransactionInfo from go node")
		suite.NoError(err_scala, "Failed to get TransactionInfo from scala node")
		suite.Equal(expected_diff_balance_in_waves, actual_diff_balance_in_waves)
		suite.Equal(expected_asset_balance, actual_asset_balance)
	}
}

func (suite *ItestSuite) Test_IssueTxWithSameDataPositive() {
	testdata := getPositiveDataMatrix(suite)
	timeout := 1 * time.Minute
	for _, td := range testdata {
		init_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)

		tx1, err_go1, err_scala1 := issue(suite, td, timeout)
		tx2, err_go2, err_scala2 := issue(suite, testDataChangedTimestamp(td), timeout)

		current_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)
		actual_diff_balance_in_waves := init_balance_in_waves - current_balance_in_waves
		actual_asset_1_balance := getAssetBalance(suite, td.Account.Address, tx1.ID.Bytes())
		actual_asset_2_balance := getAssetBalance(suite, td.Account.Address, tx2.ID.Bytes())
		diff_balance_in_waves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expected_diff_balance_in_waves := 2 * diff_balance_in_waves
		expected_asset_balance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.NoError(err_go1, "Failed to get TransactionInfo from go node")
		suite.NoError(err_scala1, "Failed to get TransactionInfo from scala node")
		suite.NoError(err_go2, "Failed to get TransactionInfo from go node")
		suite.NoError(err_scala2, "Failed to get TransactionInfo from scala node")
		suite.Equal(expected_diff_balance_in_waves, actual_diff_balance_in_waves)
		suite.Equal(expected_asset_balance, actual_asset_1_balance)
		suite.Equal(expected_asset_balance, actual_asset_2_balance)
	}
}

func (suite *ItestSuite) Test_IssueTxNegative() {
	testdata := getNegativeDataMatrix(suite)
	timeout := 5 * time.Second
	for _, td := range testdata {
		init_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)

		tx, err_go, err_scala := issue(suite, td, timeout)

		current_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)
		actual_balance_in_waves := init_balance_in_waves - current_balance_in_waves
		actual_asset_balance := getAssetBalance(suite, td.Account.Address, tx.ID.Bytes())

		expected_balance_in_waves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expected_asset_balance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		assert.EqualError(suite.T(), err_go, td.Expected["err go msg"])
		assert.EqualError(suite.T(), err_scala, td.Expected["err scala msg"])
		suite.Equal(expected_balance_in_waves, actual_balance_in_waves)
		suite.Equal(expected_asset_balance, actual_asset_balance)
	}
}
