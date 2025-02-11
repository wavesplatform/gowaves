package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	InvokeMinVersion = 1
	InvokeMaxVersion = 2
)

type InvokeScriptTestData[T any] struct {
	Sender          config.AccountInfo
	ScriptRecipient proto.Recipient
	Call            proto.FunctionCall
	Payments        proto.ScriptPayments
	ChainID         proto.Scheme
	Fee             uint64
	FeeAsset        proto.OptionalAsset
	Timestamp       uint64
	Expected        T
}

type ExpectedInvokeScriptDataPositive struct {
	Address   proto.Address
	DataEntry *waves.DataEntry
	_         struct{}
}

type ExpectedInvokeScriptDataSlicePositive struct {
	Address     proto.Address
	DataEntries []*waves.DataEntry
	_           struct{}
}

type ExpectedInvokeScriptDataNegative struct {
	WavesDiffBalanceSender    int64
	AssetDiffBalanceSender    int64
	FeeAssetDiffBalanceSender int64
	WavesDiffBalanceRecipient int64
	AssetDiffBalanceRecipient int64
	WavesDiffBalanceSponsor   int64
	AssetDiffBalanceSponsor   int64
	ErrGoMsg                  string
	ErrScalaMsg               string
	ErrBrdCstGoMsg            string
	ErrBrdCstScalaMsg         string
	_                         struct{}
}

func NewInvokeScriptTestData[T any](senderAccount config.AccountInfo, scriptRecipient proto.Recipient,
	call proto.FunctionCall, payments proto.ScriptPayments, chainID proto.Scheme, fee uint64,
	feeAssetID proto.OptionalAsset, timestamp uint64, expected T) InvokeScriptTestData[T] {
	return InvokeScriptTestData[T]{
		Sender:          senderAccount,
		ScriptRecipient: scriptRecipient,
		Call:            call,
		Payments:        payments,
		ChainID:         chainID,
		Fee:             fee,
		FeeAsset:        feeAssetID,
		Timestamp:       timestamp,
		Expected:        expected,
	}
}

func GetInvokeScriptAccountStorageUntouchedTestData(suite *f.BaseSuite, dApp,
	account config.AccountInfo) map[string]InvokeScriptTestData[ExpectedInvokeScriptDataPositive] {
	return map[string]InvokeScriptTestData[ExpectedInvokeScriptDataPositive]{
		"Check account storage is untouched by address": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("checkStorageUntouchedByAddress",
				proto.Arguments{proto.NewStringArgument(account.Address.String())}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataPositive{
				Address: dApp.Address,
				DataEntry: &waves.DataEntry{
					Key:   "test",
					Value: &waves.DataEntry_BoolValue{BoolValue: true},
				},
			},
		),
		"Check account storage is untouched by alias": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAlias(dApp.Alias),
			proto.NewFunctionCall("checkStorageUntouchedByAlias",
				proto.Arguments{proto.NewStringArgument(account.Alias.Alias)}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			500000,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataPositive{
				Address: dApp.Address,
				DataEntry: &waves.DataEntry{
					Key:   "test",
					Value: &waves.DataEntry_BoolValue{BoolValue: true},
				},
			},
		),
	}
}

func GetInvokeScriptWriteToStorageTestData(suite *f.BaseSuite,
	dApp config.AccountInfo) map[string]InvokeScriptTestData[ExpectedInvokeScriptDataSlicePositive] {
	return map[string]InvokeScriptTestData[ExpectedInvokeScriptDataSlicePositive]{
		"Checking that data is written correctly": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("writeData",
				proto.Arguments{
					&proto.BinaryArgument{Value: utl.StrToBase16Bytes(suite.T(), "test")},
					&proto.BinaryArgument{Value: utl.StrToBase58Bytes(suite.T(), "test")},
					&proto.BinaryArgument{Value: utl.StrToBase64Bytes(suite.T(), "test")},
					&proto.BooleanArgument{Value: true},
					&proto.IntegerArgument{Value: 9223372036854775807},
					&proto.StringArgument{Value: "test string value"}}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			500000,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataSlicePositive{
				Address: dApp.Address,
				DataEntries: []*waves.DataEntry{
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBase16",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBase58",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBase64",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBoolean",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.BoolToBase64Bytes(suite.T(), true)},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binInteger",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.IntToBase64Bytes(suite.T(), 9223372036854775807)},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test string value")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_bool",
						Value: &waves.DataEntry_BoolValue{BoolValue: true},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_int",
						Value: &waves.DataEntry_IntValue{IntValue: 9223372036854775807},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str",
						Value: &waves.DataEntry_StringValue{StringValue: "test string value"},
					},
				},
			}),
	}
}
