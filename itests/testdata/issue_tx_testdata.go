package testdata

import (
	"math/rand"
	"time"

	"github.com/wavesplatform/gowaves/itests/config"
	i "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789~!|#$%^&*()_+=\\\";:/?><|][{}"
	errMsg      = "transactions does not exist"
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

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for j := range b {
		b[j] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func GetCurrentTimestampInMs() uint64 {
	return uint64(time.Now().UnixMilli())
}

func DataChangedTimestamp(td *IssueTestData) IssueTestData {
	return *NewIssueTestData(td.Account, td.AssetName, td.AssetDesc, td.Quantity, td.Decimals, td.Reissuable, td.Fee,
		GetCurrentTimestampInMs(), td.ChainID, td.Expected)
}

func GetPositiveDataMatrix(suite *i.BaseSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Min values, empty description, NFT": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(4),
			"",
			1,
			0,
			false,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "1",
			}),
		"Middle values, special symbols in desc, not reissuable": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(500),
			100000000000,
			4,
			false,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "100000000000",
			}),
		"Max values": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(16),
			RandStringBytes(1000),
			9223372036854775807,
			8,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "9223372036854775807",
			}),
	}
	return t
}

func GetNegativeDataMatrix(suite *i.BaseSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Invalid asset name (len < min)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(3),
			RandStringBytes(1),
			1,
			0,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid asset name (len > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(17),
			RandStringBytes(8),
			10000,
			2,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Empty string in asset name": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			"",
			RandStringBytes(8),
			10000,
			2,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid encoding in asset name": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			"\\u0061\\u0073\\u0073\\u0065",
			RandStringBytes(8),
			10000,
			2,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		//Error in Node Go
		/*"Invalid encoding in asset description": *NewIssueTestData(
		utl.GetAccount(suite, 2),
		RandStringBytes(8),
		"\\u0061\\u0073\\u0073\\u0065\\u0074",
		10000,
		2,
		true,
		100000000,
		GetCurrentTimestampInMs(),
		'L',
		map[string]string{
			"err go msg":         errMsg,
			"err scala msg":      errMsg,
			"waves diff balance": "0",
			"asset balance":      "0",
		}),*/
		"Invalid asset description (len > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(6),
			RandStringBytes(1001),
			10000,
			2,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token quantity (quantity < min)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			0,
			2,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token quantity (quantity > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			9223372036854775808,
			2,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid token decimals (decimals > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			100000,
			9,
			true,
			100000000,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid fee (fee > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			100000,
			8,
			true,
			9223372036854775808,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid fee (0 < fee < min)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			100000,
			8,
			true,
			10,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Invalid fee (fee = 0)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			100000,
			8,
			true,
			0,
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			100000,
			8,
			true,
			100000000,
			GetCurrentTimestampInMs()-7215000,
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			100000,
			8,
			true,
			100000000,
			GetCurrentTimestampInMs()+54160000,
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
		"Creating a token when there are not enough funds on the account balance": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			RandStringBytes(8),
			RandStringBytes(8),
			100000,
			8,
			true,
			uint64(100000000+utl.GetAvalibleBalanceInWavesGo(suite, utl.GetAccount(suite, 2).Address)),
			GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":         errMsg,
				"err scala msg":      errMsg,
				"waves diff balance": "0",
				"asset balance":      "0",
			}),
	}
	return t
}
