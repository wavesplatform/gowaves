package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	integration "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

func NewIssueTestData(account config.AccountInfo, assetName string, assetDesc string, quantity uint64, decimals byte,
	reissuable bool, fee uint64, timestamp uint64, chainID proto.Scheme, expected map[string]string) *IssueTestData {
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

func DataChangedTimestamp(td *IssueTestData) IssueTestData {
	return *NewIssueTestData(td.Account, td.AssetName, td.AssetDesc, td.Quantity, td.Decimals, td.Reissuable, td.Fee,
		utilities.GetCurrentTimestampInMs(), td.ChainID, td.Expected)
}

func GetPositiveDataMatrix(suite *integration.BaseSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Min values, empty description, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"test",
			"",
			1,
			0,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "1",
			}),
		"Middle values, special symbols in desc, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"testtest",
			"~!|#$%^&*()_+=\\\";:/?><|\\\\][{}",
			100000000000,
			4,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "100000000000",
			}),
		"Max values, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
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
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "9223372036854775807",
			}),
		"NFT, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"test",
			"test",
			1,
			0,
			false,
			100000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000",
				"asset balance":      "1",
			}),
		"Not reissuable, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"testtest",
			"testtesttestest",
			100000000000,
			4,
			false,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "100000000000",
			}),
	}
	return t
}

func GetNegativeDataMatrix(suite *integration.BaseSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Invalid asset name (len < min), not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"tes",
			"t",
			1,
			0,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid asset name (len > max), not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"testtesttesttestt",
			"test",
			10000,
			2,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Empty string in asset name, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"",
			"test",
			10000,
			2,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Special symbols in asset name, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"~!|#$%^&*()_+=\\\";:/?><|\\\\][{}",
			"test",
			10000,
			2,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid encoding in asset name, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"\\u0061\\u0073\\u0073\\u0065\\u0074",
			"test",
			10000,
			2,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		//Error in Node Go
		/*"Invalid encoding in asset description, not miner": *NewIssueTestData(
		utilities.GetAccount(suite, 2),
		"test",
		"\\u0061\\u0073\\u0073\\u0065\\u0074",
		10000,
		2,
		true,
		100000000,
		utilities.GetCurrentTimestampInMs(),
		'L',
		map[string]string{
			"err go msg":         "transactions does not exist",
			"err scala msg":      "transactions does not exist",
			"waves diff balance": "0",
			"asset balance":      "0",
		}),*/
		"Invalid asset description (len > max), not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
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
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token quantity (quantity < min), not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"test",
			"test",
			0,
			2,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token quantity (quantity > max), not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"test",
			"test",
			9223372036854775808,
			2,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token decimals (decimals > max), not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"test",
			"test",
			100000,
			9,
			true,
			100000000,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid fee (fee > max), not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			9223372036854775808,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid fee (fee < min), not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			0,
			utilities.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         "transactions does not exist",
				"err scala msg":      "transactions does not exist",
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp, not miner": *NewIssueTestData(
			utilities.GetAccount(suite, 2),
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
			utilities.GetAccount(suite, 2),
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
			utilities.GetAccount(suite, 2),
			"test",
			"test",
			100000,
			8,
			true,
			uint64(100000000+utilities.GetAvalibleBalanceInWaves(suite, utilities.GetAccount(suite, 2).Address)),
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
}
