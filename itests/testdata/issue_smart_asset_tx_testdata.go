package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

const (
	IssueSmartAssetMinVersion = 2
)

func GetPositiveAssetScriptData(suite *f.BaseSuite) map[string]IssueTestData[ExpectedValuesPositive] {
	return map[string]IssueTestData[ExpectedValuesPositive]{
		"Valid script, true as expression": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			readScript(suite, "valid_script_true_as_expression.base64"),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
		"Valid script, size 8192 bytes": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			readScript(suite, "valid_8192_bytes_3910_complexity.base64"),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
		"Script with complexity 4000": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			readScript(suite, "script_with_complexity_4000.base64"),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
	}
}

func GetNegativeAssetScriptData(suite *f.BaseSuite) map[string]IssueTestData[ExpectedValuesNegative] {
	return map[string]IssueTestData[ExpectedValuesNegative]{
		"Complexity more than 4000": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			readScript(suite, "invalid_script_complexity_more_4000.base64"),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Script is too complex",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Illegal length of script": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			utl.GetScriptBytes(suite, "AA=="),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Illegal length of script",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid content type of script": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			utl.GetScriptBytes(suite, "AAQB"),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Invalid content type of script",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
		"Invalid script version": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			utl.GetScriptBytes(suite, "CAEF"),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Invalid version of script",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
	}
}
