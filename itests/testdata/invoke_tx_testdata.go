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

type ExpectedInvokeScriptDataDAppFromDAppPositive struct {
	Entries []*ExpectedInvokeScriptDataPositive
	_       struct{}
}

type ExpectedInvokeScriptDataDAppFromDAppSlicePositive struct {
	Entries []*ExpectedInvokeScriptDataSlicePositive
	_       struct{}
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
	}
}

func GetInvokeScriptWriteToStorageTestData(suite *f.BaseSuite,
	dApp config.AccountInfo) map[string]InvokeScriptTestData[ExpectedInvokeScriptDataSlicePositive] {
	return map[string]InvokeScriptTestData[ExpectedInvokeScriptDataSlicePositive]{
		"Max argument values": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("writeData",
				proto.Arguments{
					&proto.BinaryArgument{Value: utl.StrToBase16Bytes(suite.T(), "test")},
					&proto.BinaryArgument{Value: utl.StrToBase58Bytes(suite.T(), "test")},
					&proto.BinaryArgument{Value: utl.StrToBase64Bytes(suite.T(), "test")},
					&proto.BooleanArgument{Value: false},
					&proto.BooleanArgument{Value: true},
					&proto.IntegerArgument{Value: -9223372036854775808},
					&proto.IntegerArgument{Value: 9223372036854775807},
					&proto.StringArgument{Value: ""},
					&proto.StringArgument{Value: "test1"},
					&proto.StringArgument{Value: "test2"},
					&proto.StringArgument{Value: "test3"},
					&proto.StringArgument{Value: "test4"},
					&proto.StringArgument{Value: "test5"},
					&proto.StringArgument{Value: "test6"},
					&proto.StringArgument{Value: "test7"},
					&proto.StringArgument{Value: "test8"},
					&proto.StringArgument{Value: "test9"},
					&proto.StringArgument{Value: "test10"},
					&proto.StringArgument{Value: "test11"},
					&proto.StringArgument{Value: "test12"},
					&proto.StringArgument{Value: "test13"},
					&proto.StringArgument{Value: "test14"},
				}),

			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
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
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBoolean1",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.BoolToBase64Bytes(suite.T(), false)},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBoolean2",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.BoolToBase64Bytes(suite.T(), true)},
					},
					{
						Key: utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binInteger1",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.IntToBase64Bytes(suite.T(),
							-9223372036854775808)},
					},
					{
						Key: utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binInteger2",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.IntToBase64Bytes(suite.T(),
							9223372036854775807)},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString0",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString1",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test1")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString2",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test2")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString3",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test3")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString4",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test4")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString5",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test5")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString6",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test6")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString7",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test7")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString8",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test8")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString9",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test9")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString10",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test10")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString11",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test11")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString12",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test12")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString13",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test13")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString14",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test14")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_bool1",
						Value: &waves.DataEntry_BoolValue{BoolValue: false},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_bool2",
						Value: &waves.DataEntry_BoolValue{BoolValue: true},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_int1",
						Value: &waves.DataEntry_IntValue{IntValue: -9223372036854775808},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_int2",
						Value: &waves.DataEntry_IntValue{IntValue: 9223372036854775807},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str0",
						Value: &waves.DataEntry_StringValue{StringValue: ""},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str1",
						Value: &waves.DataEntry_StringValue{StringValue: "test1"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str2",
						Value: &waves.DataEntry_StringValue{StringValue: "test2"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str3",
						Value: &waves.DataEntry_StringValue{StringValue: "test3"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str4",
						Value: &waves.DataEntry_StringValue{StringValue: "test4"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str5",
						Value: &waves.DataEntry_StringValue{StringValue: "test5"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str6",
						Value: &waves.DataEntry_StringValue{StringValue: "test6"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str7",
						Value: &waves.DataEntry_StringValue{StringValue: "test7"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str8",
						Value: &waves.DataEntry_StringValue{StringValue: "test8"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str9",
						Value: &waves.DataEntry_StringValue{StringValue: "test9"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str10",
						Value: &waves.DataEntry_StringValue{StringValue: "test10"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str11",
						Value: &waves.DataEntry_StringValue{StringValue: "test11"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str12",
						Value: &waves.DataEntry_StringValue{StringValue: "test12"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str13",
						Value: &waves.DataEntry_StringValue{StringValue: "test13"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str14",
						Value: &waves.DataEntry_StringValue{StringValue: "test14"},
					},
				},
			}),
		"Update storage values": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("writeData",
				proto.Arguments{
					&proto.BinaryArgument{Value: utl.StrToBase16Bytes(suite.T(), "test_test")},
					&proto.BinaryArgument{Value: utl.StrToBase58Bytes(suite.T(), "test_test")},
					&proto.BinaryArgument{Value: utl.StrToBase64Bytes(suite.T(), "test_test")},
					&proto.BooleanArgument{Value: true},
					&proto.BooleanArgument{Value: false},
					&proto.IntegerArgument{Value: 9223372036854775807},
					&proto.IntegerArgument{Value: -9223372036854775808},
					&proto.StringArgument{Value: "test0"},
					&proto.StringArgument{Value: "test_test1"},
					&proto.StringArgument{Value: "test_test2"},
					&proto.StringArgument{Value: "test_test3"},
					&proto.StringArgument{Value: "test_test4"},
					&proto.StringArgument{Value: "test_test5"},
					&proto.StringArgument{Value: "test_test6"},
					&proto.StringArgument{Value: "test_test7"},
					&proto.StringArgument{Value: "test_test8"},
					&proto.StringArgument{Value: "test_test9"},
					&proto.StringArgument{Value: "test_test10"},
					&proto.StringArgument{Value: "test_test11"},
					&proto.StringArgument{Value: "test_test12"},
					&proto.StringArgument{Value: "test_test13"},
					&proto.StringArgument{Value: "test_test14"},
				}),

			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataSlicePositive{
				Address: dApp.Address,
				DataEntries: []*waves.DataEntry{
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBase16",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBase58",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBase64",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBoolean1",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.BoolToBase64Bytes(suite.T(), true)},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binBoolean2",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.BoolToBase64Bytes(suite.T(), false)},
					},
					{
						Key: utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binInteger1",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.IntToBase64Bytes(suite.T(),
							9223372036854775807)},
					},
					{
						Key: utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binInteger2",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: utl.IntToBase64Bytes(suite.T(),
							-9223372036854775808)},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString0",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test0")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString1",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test1")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString2",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test2")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString3",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test3")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString4",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test4")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString5",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test5")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString6",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test6")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString7",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test7")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString8",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test8")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString9",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test9")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString10",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test10")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString11",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test11")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString12",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test12")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString13",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test13")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_binString14",
						Value: &waves.DataEntry_BinaryValue{BinaryValue: []byte("test_test14")},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_bool1",
						Value: &waves.DataEntry_BoolValue{BoolValue: true},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_bool2",
						Value: &waves.DataEntry_BoolValue{BoolValue: false},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_int1",
						Value: &waves.DataEntry_IntValue{IntValue: 9223372036854775807},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_int2",
						Value: &waves.DataEntry_IntValue{IntValue: -9223372036854775808},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str0",
						Value: &waves.DataEntry_StringValue{StringValue: "test0"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str1",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test1"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str2",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test2"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str3",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test3"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str4",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test4"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str5",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test5"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str6",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test6"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str7",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test7"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str8",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test8"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str9",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test9"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str10",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test10"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str11",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test11"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str12",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test12"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str13",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test13"},
					},
					{
						Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str14",
						Value: &waves.DataEntry_StringValue{StringValue: "test_test14"},
					},
				},
			}),
	}
}

func GetInvokeScriptWriteToStorageStringTestData(suite *f.BaseSuite, version byte,
	dApp config.AccountInfo) map[string]InvokeScriptTestData[ExpectedInvokeScriptDataPositive] {
	var maxStrVal string
	if version == 1 {
		maxStrVal = utl.EscapeSeq + utl.Umlauts + utl.SymbolSet + utl.RusLetters +
			utl.RandStringBytes(4572, utl.LettersAndDigits)
	} else {
		maxStrVal = utl.EscapeSeq + utl.Umlauts + utl.SymbolSet + utl.RusLetters +
			utl.RandStringBytes(4695, utl.LettersAndDigits)
	}
	return map[string]InvokeScriptTestData[ExpectedInvokeScriptDataPositive]{
		"Max value for string variable": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("writeMaxValueString",
				proto.Arguments{
					&proto.StringArgument{Value: maxStrVal}}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataPositive{
				Address: dApp.Address,
				DataEntry: &waves.DataEntry{
					Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str",
					Value: &waves.DataEntry_StringValue{StringValue: maxStrVal},
				},
			}),
		"Max value for function name and variable name in dApp": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall(
				"maxValueForFunctionNameleViTzlwxjekBw9tUO5BONNIXSLhEoQJ2Y0voSups40mBcWxngZ6y7jBPw7x2JMs45W"+
					"6Ea8JH9UWrCR2DGRaQf0q9g7JbgRUfiTD3mb0mbQ7yov0xI0LQ7HrmkqKLI90lOVHR76dydswEDG47nJwL6y7OBBZVLV"+
					"gZPV8fIaApY1hE8VwskQ7WmVjqRwyfAxEQJW9LlPB25gplswvIg47HjOVDWnbDV2nmyh2qXMB",
				proto.Arguments{
					&proto.StringArgument{Value: "test"}}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataPositive{
				Address: dApp.Address,
				DataEntry: &waves.DataEntry{
					Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str",
					Value: &waves.DataEntry_StringValue{StringValue: "test"},
				},
			}),
	}
}

func GetInvokeScriptDAppFromDAppTestData(suite *f.BaseSuite, version byte,
	dAppProxy1, dAppProxy2,
	dAppTarget config.AccountInfo) map[string]InvokeScriptTestData[ExpectedInvokeScriptDataDAppFromDAppSlicePositive] {
	var maxStrVal string
	if version == 1 {
		maxStrVal = utl.EscapeSeq + utl.Umlauts + utl.SymbolSet + utl.RusLetters +
			utl.RandStringBytes(4502, utl.LettersAndDigits)
	} else {
		maxStrVal = utl.EscapeSeq + utl.Umlauts + utl.SymbolSet + utl.RusLetters +
			utl.RandStringBytes(4625, utl.LettersAndDigits)
	}
	return map[string]InvokeScriptTestData[ExpectedInvokeScriptDataDAppFromDAppSlicePositive]{
		"Total size of invoke 15 KB": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dAppProxy1.Address),
			proto.NewFunctionCall("callProxy",
				proto.Arguments{
					&proto.StringArgument{Value: dAppProxy2.Address.String()},
					&proto.StringArgument{Value: dAppTarget.Address.String()},
					&proto.StringArgument{Value: maxStrVal}}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataDAppFromDAppSlicePositive{
				Entries: []*ExpectedInvokeScriptDataSlicePositive{
					{
						Address: dAppProxy1.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: maxStrVal},
							},
						},
					},
					{
						Address: dAppProxy2.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dAppProxy1.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: maxStrVal},
							},
						},
					},
					{
						Address: dAppTarget.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dAppProxy2.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: maxStrVal},
							},
						},
					},
				},
			}),
		"Total count of Data Entries is 100": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dAppProxy1.Address),
			proto.NewFunctionCall("writeMaxActions",
				proto.Arguments{
					&proto.StringArgument{Value: dAppProxy2.Address.String()},
					&proto.StringArgument{Value: dAppTarget.Address.String()},
					&proto.StringArgument{Value: "test"},
					&proto.IntegerArgument{Value: 0},
					&proto.BooleanArgument{Value: true},
					&proto.BinaryArgument{Value: []byte{1, 2, 3}},
				}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataDAppFromDAppSlicePositive{
				Entries: []*ExpectedInvokeScriptDataSlicePositive{
					{
						Address: dAppProxy1.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
					{
						Address: dAppProxy2.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dAppProxy1.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
					{
						Address: dAppTarget.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dAppProxy2.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
				},
			},
		),
	}
}

func GetInvokeDAppTargetFromDAppProxy(suite *f.BaseSuite, version byte,
	dAppProxy,
	dAppTarget config.AccountInfo) map[string]InvokeScriptTestData[ExpectedInvokeScriptDataDAppFromDAppSlicePositive] {
	return map[string]InvokeScriptTestData[ExpectedInvokeScriptDataDAppFromDAppSlicePositive]{
		"Total count of Data Entries is 100": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dAppProxy.Address),
			proto.NewFunctionCall("writeMaxActions",
				proto.Arguments{
					&proto.StringArgument{Value: dAppTarget.Address.String()},
					&proto.StringArgument{Value: "test"},
					&proto.IntegerArgument{Value: 0},
					&proto.BooleanArgument{Value: true},
					&proto.BinaryArgument{Value: []byte{1, 2, 3}},
				}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataDAppFromDAppSlicePositive{
				Entries: []*ExpectedInvokeScriptDataSlicePositive{
					{
						Address: dAppProxy.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
					{
						Address: dAppTarget.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dAppProxy.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
				},
			},
		),
	}
}

func GetInvokeScriptDAppRecursiveTestData(suite *f.BaseSuite, version byte,
	dApp config.AccountInfo) map[string]InvokeScriptTestData[ExpectedInvokeScriptDataDAppFromDAppSlicePositive] {

	return map[string]InvokeScriptTestData[ExpectedInvokeScriptDataDAppFromDAppSlicePositive]{

		"Total count of Data Entries is 100": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("writeMaxActions",
				proto.Arguments{
					&proto.StringArgument{Value: "test"},
					&proto.IntegerArgument{Value: 0},
					&proto.BooleanArgument{Value: true},
					&proto.BinaryArgument{Value: []byte{1, 2, 3}},
				}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataDAppFromDAppSlicePositive{
				Entries: []*ExpectedInvokeScriptDataSlicePositive{
					{
						Address: dApp.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dApp.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
				},
			},
		),
		"Invoke DApp from DApp 100 times (recursive)": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("recursiveCall",
				proto.Arguments{
					&proto.IntegerArgument{Value: 100},
					&proto.StringArgument{Value: "test"},
				}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataDAppFromDAppSlicePositive{
				Entries: []*ExpectedInvokeScriptDataSlicePositive{
					{
						Address: dApp.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dApp.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
				},
			},
		),
		"Invoke DApp from DApp 100 times (in one script)": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("call1",
				proto.Arguments{
					&proto.StringArgument{Value: "test"},
				}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataDAppFromDAppSlicePositive{
				Entries: []*ExpectedInvokeScriptDataSlicePositive{
					{
						Address: dApp.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dApp.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
				},
			},
		),
		"Invoke DApp from DApp 100 times": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			proto.NewFunctionCall("call1",
				proto.Arguments{
					&proto.StringArgument{Value: "test"},
				}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			utl.MinTxFeeWavesDApp,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataDAppFromDAppSlicePositive{
				Entries: []*ExpectedInvokeScriptDataSlicePositive{
					{
						Address: dApp.Address,
						DataEntries: []*waves.DataEntry{
							{
								Key:   dApp.Address.String() + "_str",
								Value: &waves.DataEntry_StringValue{StringValue: "test"},
							},
						},
					},
				},
			},
		),
	}
}
