package testdata

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	SetScriptMaxVersion = 2
	ScriptDir           = "dApp_scripts"
)

type SetScriptData struct {
	SenderAccount config.AccountInfo
	Script        proto.Script
	ChainID       proto.Scheme
	Fee           uint64
	Timestamp     uint64
}

func getDAppScript(suite *f.BaseSuite, name string) proto.Script {
	script, err := utl.ReadAndCompileRideScript(ScriptDir, name)
	require.NoError(suite.T(), err, "unable to read dApp script")
	return script
}

func NewSetScriptData(senderAccount config.AccountInfo, script proto.Script, chainID proto.Scheme,
	fee uint64, timestamp uint64) SetScriptData {
	return SetScriptData{
		SenderAccount: senderAccount,
		Script:        script,
		ChainID:       chainID,
		Fee:           fee,
		Timestamp:     timestamp,
	}
}

func GetDataForDAppAccount(suite *f.BaseSuite, account config.AccountInfo, scriptName string) SetScriptData {
	return NewSetScriptData(
		account,
		getDAppScript(suite, scriptName),
		utl.TestChainID,
		utl.SetDAppScriptFeeWaves,
		utl.GetCurrentTimestampInMs(),
	)
}
