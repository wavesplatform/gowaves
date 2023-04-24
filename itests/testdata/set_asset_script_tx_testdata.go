package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	SetAssetScriptMaxVersion = 3
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
	fee, timestamp uint64, chainID proto.Scheme, expected T) *SetAssetScriptTestData[T] {
	return &SetAssetScriptTestData[T]{
		Account:   account,
		AssetID:   assetID,
		Script:    script,
		Fee:       fee,
		Timestamp: timestamp,
		ChainID:   chainID,
		Expected:  expected,
	}
}

func GetSetAssetScriptPositiveData(suite *f.BaseSuite, assetID crypto.Digest) map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesPositive] {
	return map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesPositive]{
		"Valid script, true as expression": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
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
		"": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, ""),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  utl.MinSetAssetScriptFeeWaves,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
			}),
	}
}
