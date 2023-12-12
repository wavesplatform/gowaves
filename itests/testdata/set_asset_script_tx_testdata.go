package testdata

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	SetAssetScriptMaxVersion = 3
	SetAssetScriptMinVersion = 2
	AssetScriptDir           = "asset_scripts"
)

type SetAssetScriptTestData[T any] struct {
	Account   config.AccountInfo
	AssetID   crypto.Digest
	Script    proto.Script
	Fee       uint64
	Timestamp uint64
	ChainID   proto.Scheme
	Expected  T
}

type SetAssetScriptExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	_                struct{}
}

type SetAssetScriptExpectedValuesNegative struct {
	WavesDiffBalance  int64
	AssetDiffBalance  int64
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	_                 struct{}
}

func NewSetAssetScriptTestData[T any](account config.AccountInfo, assetID crypto.Digest, script proto.Script,
	fee, timestamp uint64, chainID proto.Scheme, expected T) SetAssetScriptTestData[T] {
	return SetAssetScriptTestData[T]{
		Account:   account,
		AssetID:   assetID,
		Script:    script,
		Fee:       fee,
		Timestamp: timestamp,
		ChainID:   chainID,
		Expected:  expected,
	}
}

func readScript(suite *f.BaseSuite, name string) proto.Script {
	script, err := utl.ReadScript(AssetScriptDir, name)
	require.NoError(suite.T(), err, "unable to read asset script")
	return script
}

func GetSetAssetScriptPositiveData(suite *f.BaseSuite, assetID crypto.Digest) map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesPositive] {
	return map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesPositive]{
		"Valid script, true as expression": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "valid_script_true_as_expression.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesPositive{
				WavesDiffBalance: utl.MinSetAssetScriptFeeWaves,
				AssetDiffBalance: 0,
			}),
		"Valid script, size 8192 bytes": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "valid_8192_bytes_3910_complexity.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesPositive{
				WavesDiffBalance: utl.MinSetAssetScriptFeeWaves,
				AssetDiffBalance: 0,
			}),
		"Script with complexity 4000": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "script_with_complexity_4000.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesPositive{
				WavesDiffBalance: utl.MinSetAssetScriptFeeWaves,
				AssetDiffBalance: 0,
			}),
	}
}

func GetSetAssetScriptNegativeData(suite *f.BaseSuite, assetID crypto.Digest) map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesNegative] {
	return map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesNegative]{
		"Empty script": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "empty_script.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Can't parse empty script bytes",
			}),
		"Complexity more than 4000": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "invalid_script_complexity_more_4000.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Script is too complex",
			}),
		"Illegal length of script": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "AA=="),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Illegal length of script",
			}),
		"Invalid content type of script": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "AAQB"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Invalid content type of script",
			}),
		"Invalid script version": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "CAEF"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "ScriptParseError(Invalid checksum)",
			}),
		"Asset was issued by other Account": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultRecipientNotMiner),
			assetID,
			readScript(suite, "valid_script_true_as_expression.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Asset was issued by other address",
			}),
		"Invalid fee (fee > max)": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "valid_script_true_as_expression.base64"),
			utl.MaxAmount+1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "json data validation error",
			}),
		"Invalid fee (0 < fee < min)": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "valid_script_true_as_expression.base64"),
			10,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "(10 in WAVES) does not exceed minimal value",
			}),
		"Invalid fee (fee = 0)": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "valid_script_true_as_expression.base64"),
			0,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "insufficient fee",
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "valid_script_true_as_expression.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs()-7260000,
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 7200000ms in the past relative to previous block timestamp",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "valid_script_true_as_expression.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs()+54160000,
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 5400000ms in the future relative to block timestamp",
			}),
		"Try to do sponsorship when fee more than funds on the sender balance": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			readScript(suite, "valid_script_true_as_expression.base64"),
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "negative waves balance",
			}),
		"Invalid asset ID (asset ID not exist)": NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandDigest(suite.T(), 32, utl.LettersAndDigits),
			readScript(suite, "valid_script_true_as_expression.base64"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Referenced assetId not found",
			}),
	}
}

func GetSimpleSmartAssetNegativeData(suite *f.BaseSuite, assetID crypto.Digest) SetAssetScriptTestData[SetAssetScriptExpectedValuesNegative] {
	return NewSetAssetScriptTestData(
		utl.GetAccount(suite, utl.DefaultSenderNotMiner),
		assetID,
		readScript(suite, "valid_script_true_as_expression.base64"),
		utl.MinSetAssetScriptFeeWaves,
		utl.GetCurrentTimestampInMs(),
		utl.TestChainID,
		SetAssetScriptExpectedValuesNegative{
			WavesDiffBalance:  0,
			AssetDiffBalance:  0,
			ErrGoMsg:          errMsg,
			ErrScalaMsg:       errMsg,
			ErrBrdCstGoMsg:    errBrdCstMsg,
			ErrBrdCstScalaMsg: "",
		})
}
