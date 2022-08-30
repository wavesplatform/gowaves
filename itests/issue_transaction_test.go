package integration_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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

func dataChangedTimestamp(td IssueTestData) IssueTestData {
	return *NewIssueTestData(td.Account, td.AssetName, td.AssetDesc, td.Quantity, td.Decimals, td.Reissuable, td.Fee,
		getCurrentTimestampInMs(), td.ChainID, td.Expected)
}

func getInvalidTxIdsInBlockchain(suite *ItestSuite, ids []*crypto.Digest, timeout time.Duration) []*crypto.Digest {
	time.Sleep(timeout)
	for _, id := range ids {
		_, _, err_go := suite.clients.GoClients.HttpClient.TransactionInfoRaw(*id)
		_, _, err_scala := suite.clients.ScalaClients.HttpClient.TransactionInfoRaw(*id)
		if (err_go != nil) && (err_scala != nil) {
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
			"testtesttesttest",
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid encoding in asset name, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"0061 0073 0073 0065 0074",
			"test",
			10000,
			2,
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
		"Invalid encoding in asset description, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"0061 0073 0073 0065 0074",
			10000,
			2,
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
		"Special symbols in asset description, not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"~!|#$%^&*()_+=\\\";:/?><|\\\\][{}",
			10000,
			2,
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Empty string in asset description (len < min), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"",
			10000,
			2,
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
		"Invalid chain ID (0), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			10000,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			0,
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
		"Invalid chain ID (256), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			10000,
			2,
			true,
			100000000,
			getCurrentTimestampInMs(),
			256,
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token quantity (negative), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			-1,
			2,
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
		"Invalid token decimals (negative), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			10000,
			-1,
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
		"Invalid token decimals (float), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			100000,
			1.5,
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid fee (negative), not miner": *NewIssueTestData(
			getAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			-1,
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
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
				"err go msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}: " +
					"Invalid status code: expect 200 got 404",
				"err scala msg": "reached retry deadline: " +
					"{\"error\":311,\"message\":\"transactions does not exist\"}\n: " +
					"Invalid status code: expect 200 got 404",
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
	for name, td := range testdata {
		init_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)

		tx, err_go, err_scala := issue(suite, td, timeout)

		current_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)
		actual_diff_balance_in_waves := init_balance_in_waves - current_balance_in_waves
		actual_asset_balance := getAssetBalance(suite, td.Account.Address, tx.ID.Bytes())

		expected_diff_balance_in_waves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expected_asset_balance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.NoErrorf(err_go, "In case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(err_scala, "In case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(expected_diff_balance_in_waves, actual_diff_balance_in_waves, "In case: \"%s\"", name)
		suite.Equalf(expected_asset_balance, actual_asset_balance, "In case: \"%s\"", name)
	}
}

func (suite *ItestSuite) Test_IssueTxWithSameDataPositive() {
	testdata := getPositiveDataMatrix(suite)
	timeout := 1 * time.Minute
	for name, td := range testdata {
		init_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)

		tx1, err_go1, err_scala1 := issue(suite, td, timeout)
		tx2, err_go2, err_scala2 := issue(suite, dataChangedTimestamp(td), timeout)

		current_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)
		actual_diff_balance_in_waves := init_balance_in_waves - current_balance_in_waves
		actual_asset_1_balance := getAssetBalance(suite, td.Account.Address, tx1.ID.Bytes())
		actual_asset_2_balance := getAssetBalance(suite, td.Account.Address, tx2.ID.Bytes())
		diff_balance_in_waves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expected_diff_balance_in_waves := 2 * diff_balance_in_waves
		expected_asset_balance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		suite.NoErrorf(err_go1, "In case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(err_scala1, "In case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.NoErrorf(err_go2, "In case: \"%s\": Failed to get TransactionInfo from go node", name)
		suite.NoErrorf(err_scala2, "In case: \"%s\": Failed to get TransactionInfo from scala node", name)
		suite.Equalf(expected_diff_balance_in_waves, actual_diff_balance_in_waves, "In case: \"%s\"", name)
		suite.Equalf(expected_asset_balance, actual_asset_1_balance, "In case: \"%s\"", name)
		suite.Equalf(expected_asset_balance, actual_asset_2_balance, "In case: \"%s\"", name)
	}
}

func (suite *ItestSuite) Test_IssueTxNegative() {
	testdata := getNegativeDataMatrix(suite)
	timeout := 2 * time.Second
	var tx_ids []*crypto.Digest
	for name, td := range testdata {
		init_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)

		tx, err_go, err_scala := issue(suite, td, timeout)
		tx_ids = append(tx_ids, tx.ID)

		current_balance_in_waves := getAvalibleBalanceInWaves(suite, td.Account.Address)
		actual_balance_in_waves := init_balance_in_waves - current_balance_in_waves
		actual_asset_balance := getAssetBalance(suite, td.Account.Address, tx.ID.Bytes())

		expected_balance_in_waves, _ := strconv.ParseInt(td.Expected["waves diff balance"], 10, 64)
		expected_asset_balance, _ := strconv.ParseInt(td.Expected["asset balance"], 10, 64)

		assert.EqualErrorf(suite.T(), err_go, td.Expected["err go msg"], "In case: \"%s\"", name)
		assert.EqualErrorf(suite.T(), err_scala, td.Expected["err scala msg"], "In case: \"%s\"", name)
		suite.Equalf(expected_balance_in_waves, actual_balance_in_waves, "In case: \"%s\"", name)
		suite.Equalf(expected_asset_balance, actual_asset_balance, "In case: \"%s\"", name)
	}
	suite.Equalf(0, len(getInvalidTxIdsInBlockchain(suite, tx_ids, 45*timeout)), "IDs: %#v", tx_ids)
}
