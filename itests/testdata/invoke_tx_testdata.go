package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	InvokeMinVersion = 2
	InvokeMaxVersion = 2
)

type InvokeScriptTestData struct {
	SenderAccount   config.AccountInfo
	ScriptRecipient proto.Recipient
	Call            proto.FunctionCall
	Payments        proto.ScriptPayments
	ChainID         proto.Scheme
	Fee             uint64
	FeeAsset        proto.OptionalAsset
	Timestamp       uint64
}

func NewInvokeScriptTestData(senderAccount config.AccountInfo, scriptRecipient proto.Recipient, call proto.FunctionCall,
	payments proto.ScriptPayments, chainID proto.Scheme, fee uint64, timestamp uint64) InvokeScriptTestData {
	return InvokeScriptTestData{
		SenderAccount:   senderAccount,
		ScriptRecipient: scriptRecipient,
		Call:            call,
		Payments:        payments,
		ChainID:         chainID,
		Fee:             fee,
		Timestamp:       timestamp,
	}
}

func CheckInvokeDAppFromDApp(suite *f.BaseSuite, recipientDApp, dApp2 config.AccountInfo) map[string]InvokeScriptTestData {
	return map[string]InvokeScriptTestData{
		"Check invoke dApp from dApp": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(recipientDApp.Address),
			proto.NewFunctionCall("invokeNext",
				proto.Arguments{&proto.IntegerArgument{Value: 5_0000_0000},
					&proto.StringArgument{dApp2.Address.String()}}),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			10000000,
			utl.GetCurrentTimestampInMs(),
		),
	}
}

func GetInvokeScriptTestData(suite *f.BaseSuite) map[string]InvokeScriptTestData {
	return map[string]InvokeScriptTestData{
		"Check Invoke Script Tx": NewInvokeScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			proto.NewFunctionCall("call", make(proto.Arguments, 0)),
			make(proto.ScriptPayments, 0),
			utl.TestChainID,
			100500000,
			utl.GetCurrentTimestampInMs(),
		),
	}
}
