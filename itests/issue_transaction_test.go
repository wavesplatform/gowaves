package integration_test

import (
	"fmt"
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
				"err msg":            "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
	}
	return t
}

func createAndSendTransaction(suite *ItestSuite, testdata IssueTestData) (*proto.IssueWithSig, error, error) {
	tx := proto.NewUnsignedIssueWithSig(testdata.Account.PublicKey, testdata.AssetName,
		testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	suite.NoError(err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.conns.SendToEachNode(suite.T(), &txMsg)

	err_go, err_scala := suite.clients.WaitForTransaction(suite.T(), tx.ID, 1*time.Minute)
	return tx, err_go, err_scala
}

func (suite *ItestSuite) Test_IssueTxPositive() {
	testdata := getPositiveDataMatrix(suite)
	for _, td := range testdata {
		init_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)

		tx, err_go, err_scala := createAndSendTransaction(suite, td)
		suite.NoError(err_go, "Failed to get TransactionInfo from go node")
		suite.NoError(err_scala, "Failed to get TransactionInfo from scala node")

		current_balance_in_waves := getAvalibleBalanceInWaves(suite, getAccount(suite, 2).Address)
		actual_balance_in_waves := init_balance_in_waves - current_balance_in_waves
		actual_asset_balance := getAssetBalance(suite, getAccount(suite, 2).Address, tx.ID.Bytes())

		expected_balance_in_waves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expected_asset_balance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.Equal(expected_balance_in_waves, actual_balance_in_waves)
		suite.Equal(expected_asset_balance, actual_asset_balance)
	}
}

func (suite *ItestSuite) Test_IssueTxNegative() {
	testdata := getNegativeDataMatrix(suite)
	for _, td := range testdata {
		init_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)

		tx, err_go, err_scala := createAndSendTransaction(suite, td)
		if err_go != nil {
			fmt.Println(err_go)
		}
		if err_scala != nil {
			fmt.Println(err_scala)
		}

		current_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)
		actual_balance_in_waves := init_balance_in_waves - current_balance_in_waves
		actual_asset_balance := getAssetBalance(suite, td.Account.Address, tx.ID.Bytes())

		expected_balance_in_waves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expected_asset_balance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.Equal(expected_balance_in_waves, actual_balance_in_waves)
		suite.Equal(expected_asset_balance, actual_asset_balance)
	}
}
