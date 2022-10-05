package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	errMsg       = "transactions does not exist"
	errBrdCstMsg = "Error is unknown"
	errName      = "invalid name"
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
		utl.GetCurrentTimestampInMs(), td.ChainID, td.Expected)
}

func GetPositiveDataMatrix(suite *f.BaseSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Min values, empty description, NFT": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(4),
			"",
			1,
			0,
			false,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "1",
			}),
		"Middle values, special symbols in desc, not reissuable": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(500),
			100000000000,
			4,
			false,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "100000000000",
			}),
		"Max values": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(16),
			utl.RandStringBytes(1000),
			9223372036854775807,
			8,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"waves diff balance": "100000000",
				"asset balance":      "9223372036854775807",
			}),
	}
	return t
}

func GetNegativeDataMatrix(suite *f.BaseSuite) map[string]IssueTestData {
	var t = map[string]IssueTestData{
		"Invalid asset name (len < min)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(3),
			utl.RandStringBytes(1),
			1,
			0,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": errName,
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Invalid asset name (len > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(17),
			utl.RandStringBytes(8),
			10000,
			2,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": errName,
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Empty string in asset name": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			"",
			utl.RandStringBytes(8),
			10000,
			2,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": errName,
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Invalid encoding in asset name": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			"\\u0061\\u0073\\u0073\\u0065",
			utl.RandStringBytes(8),
			10000,
			2,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": errName,
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		//Error in Node Go
		/*"Invalid encoding in asset description": *NewIssueTestData(
		utl.GetAccount(suite, 2),
		utl.RandStringBytes(8),
		"\\u0061\\u0073\\u0073\\u0065\\u0074",
		10000,
		2,
		true,
		100000000,
		utl.GetCurrentTimestampInMs(),
		'L',
		map[string]string{
			"err go msg":         errMsg,
			"err scala msg":      errMsg,
		    "err brdcst msg go":    errBrdCstMsg,
		    "err brdcst msg scala": "",
			"waves diff balance": "0",
			"asset balance":      "0",
		}),*/
		"Invalid asset description (len > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(6),
			utl.RandStringBytes(1001),
			10000,
			2,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "Too big sequence requested",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Invalid token quantity (quantity < min)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			0,
			2,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "non-positive amount: 0 of assets",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Invalid token quantity (quantity > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			9223372036854775808,
			2,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "failed to parse json message",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Invalid token decimals (decimals > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			100000,
			9,
			true,
			100000000,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "invalid decimals value: 9, decimals should be in interval [0; 8]",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Invalid fee (fee > max)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			100000,
			8,
			true,
			9223372036854775808,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "failed to parse json message",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Invalid fee (0 < fee < min)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			100000,
			8,
			true,
			10,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "Fee for IssueTransaction (10 in WAVES) does not exceed minimal value of 100000000 WAVES",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Invalid fee (fee = 0)": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			100000,
			8,
			true,
			0,
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "insufficient fee",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			100000,
			8,
			true,
			100000000,
			utl.GetCurrentTimestampInMs()-7215000,
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "is more than 7200000ms in the past relative to previous block timestamp",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			100000,
			8,
			true,
			100000000,
			utl.GetCurrentTimestampInMs()+54160000,
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "is more than 5400000ms in the future relative to block timestamp",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
		"Creating a token when there are not enough funds on the account balance": *NewIssueTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(8),
			utl.RandStringBytes(8),
			100000,
			8,
			true,
			uint64(100000000+utl.GetAvalibleBalanceInWavesGo(suite, utl.GetAccount(suite, 2).Address)),
			utl.GetCurrentTimestampInMs(),
			'L',
			map[string]string{
				"err go msg":           errMsg,
				"err scala msg":        errMsg,
				"err brdcst msg go":    errBrdCstMsg,
				"err brdcst msg scala": "Accounts balance errors",
				"waves diff balance":   "0",
				"asset balance":        "0",
			}),
	}
	return t
}
