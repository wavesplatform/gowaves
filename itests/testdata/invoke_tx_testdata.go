package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	InvokeMinVersion = 2
	InvokeMaxVersion = 2
)

type InvokeScriptTestData[T any] struct {
	SenderAccount   config.AccountInfo
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
	DataEntry waves.DataEntry
	_         struct{}
}

type ExpectedInvokeScriptDataNegative struct {
}

func NewInvokeScriptTestData[T any](senderAccount config.AccountInfo, scriptRecipient proto.Recipient,
	call proto.FunctionCall, payments proto.ScriptPayments, chainID proto.Scheme, fee uint64,
	feeAssetID proto.OptionalAsset, timestamp uint64, expected T) InvokeScriptTestData[T] {
	return InvokeScriptTestData[T]{
		SenderAccount:   senderAccount,
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

func GetInvokeScriptAccountStorageUntouchedTestData(suite *f.BaseSuite, dAppAliasOrAddress, accountAddressOrAlias string) map[string]InvokeScriptTestData[ExpectedInvokeScriptDataPositive] {
	return map[string]InvokeScriptTestData[ExpectedInvokeScriptDataPositive]{
		/*"Check account storage is untouched by address": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(dApp.Address),
			//proto.NewFunctionCall("call", make(proto.Arguments, 0)),
			proto.NewFunctionCall("checkStorageUntouchedByAddress",
				proto.Arguments{proto.NewStringArgument(caller.Address.String())}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			500000,
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataPositive{
				Address: dApp.Address,
				DataEntry: waves.DataEntry{
					Key:   "test",
					Value: &waves.DataEntry_BoolValue{BoolValue: true},
				},
			},
		),*/
		"Check account storage is untouched by alias": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAlias(*utl.GetAliasFromString(suite, dAppAliasOrAddress, utl.TestChainID)),
			proto.NewFunctionCall("checkStorageUntouchedByAlias",
				proto.Arguments{proto.NewStringArgument(accountAddressOrAlias)}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			500000,
			utl.GetAssetByID(nil),
			utl.GetCurrentTimestampInMs(),
			ExpectedInvokeScriptDataPositive{
				Address: utl.GetAddressFromRecipient(suite, proto.NewRecipientFromAlias(*utl.GetAliasFromString(suite,
					dAppAliasOrAddress, utl.TestChainID))),
				DataEntry: waves.DataEntry{
					Key:   "test",
					Value: &waves.DataEntry_BoolValue{BoolValue: true},
				},
			},
		),
	}
}
