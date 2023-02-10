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

type IssueTestData[T any] struct {
	Account    config.AccountInfo
	AssetName  string
	AssetDesc  string
	Quantity   uint64
	Decimals   byte
	Reissuable bool
	Fee        uint64
	Timestamp  uint64
	ChainID    proto.Scheme
	Expected   T
}

type ExpectedValuesNegative struct {
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	WavesDiffBalance  int64
	AssetBalance      int64
	_                 struct{} // this field is necessary to force using explicit struct initialization
}

type ExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetBalance     int64
	_                struct{} // this field is necessary to force using explicit struct initialization
}

func NewIssueTestData[T any](account config.AccountInfo, assetName string, assetDesc string, quantity uint64, decimals byte,
	reissuable bool, fee uint64, timestamp uint64, chainID proto.Scheme, expected T) *IssueTestData[T] {
	return &IssueTestData[T]{
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

func DataChangedTimestamp[T any](td *IssueTestData[T]) IssueTestData[T] {
	return *NewIssueTestData(td.Account, td.AssetName, td.AssetDesc, td.Quantity, td.Decimals, td.Reissuable, td.Fee,
		utl.GetCurrentTimestampInMs(), td.ChainID, td.Expected)
}

type CommonIssueData struct {
	NFT        IssueTestData[ExpectedValuesPositive]
	Reissuable IssueTestData[ExpectedValuesPositive]
}

func GetCommonIssueData(suite *f.BaseSuite) CommonIssueData {
	return CommonIssueData{
		NFT: *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(4, utl.CommonSymbolSet),
			"",
			1,
			0,
			false,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     1,
			}),
		Reissuable: *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(500, utl.CommonSymbolSet),
			100000000000,
			4,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
	}
}

func GetPositiveDataMatrix(suite *f.BaseSuite) map[string]IssueTestData[ExpectedValuesPositive] {
	var t = map[string]IssueTestData[ExpectedValuesPositive]{
		"Min values, empty description, NFT": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(4, utl.CommonSymbolSet),
			"",
			1,
			0,
			false,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     1,
			}),
		"Middle values, special symbols in desc, not reissuable": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(500, utl.CommonSymbolSet),
			100000000000,
			4,
			false,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
		"Max values": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(16, utl.CommonSymbolSet),
			utl.RandStringBytes(1000, utl.CommonSymbolSet),
			utl.MaxAmount,
			8,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     utl.MaxAmount,
			}),
	}
	return t
}

func GetNegativeDataMatrix(suite *f.BaseSuite) map[string]IssueTestData[ExpectedValuesNegative] {
	var t = map[string]IssueTestData[ExpectedValuesNegative]{
		"Invalid asset name (len < min)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(3, utl.CommonSymbolSet),
			utl.RandStringBytes(1, utl.CommonSymbolSet),
			1,
			0,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: errName,
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid asset name (len > max)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(17, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			10000,
			2,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: errName,
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Empty string in asset name": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			"",
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			10000,
			2,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: errName,
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid encoding in asset name": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			"\\u0061\\u0073\\u0073\\u0065",
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			10000,
			2,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: errName,
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid asset description (len > max)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(6, utl.CommonSymbolSet),
			utl.RandStringBytes(1001, utl.CommonSymbolSet),
			10000,
			2,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Too big sequence requested",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid token quantity (quantity < min)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			0,
			2,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "non-positive amount: 0 of assets",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid token quantity (quantity > max)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.MaxAmount+1,
			2,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid token decimals (decimals > max)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			100000,
			9,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "invalid decimals value: 9, decimals should be in interval [0; 8]",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid fee (fee > max)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			100000,
			8,
			true,
			utl.MaxAmount+1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid fee (0 < fee < min)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			100000,
			8,
			true,
			10,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Fee for IssueTransaction (10 in WAVES) does not exceed minimal value of 100000000 WAVES",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid fee (fee = 0)": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			100000,
			8,
			true,
			0,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "insufficient fee",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			100000,
			8,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs()-7260000,
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 7200000ms in the past relative to previous block timestamp",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			100000,
			8,
			true,
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs()+54160000,
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 5400000ms in the future relative to block timestamp",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Creating a token when there are not enough funds on the account balance": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			100000,
			8,
			true,
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Accounts balance errors",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
	}
	return t
}
