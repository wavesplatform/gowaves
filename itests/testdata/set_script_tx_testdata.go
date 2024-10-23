package testdata

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	SetScriptMinVersion = 2
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

/*func readDAppScript(suite *f.BaseSuite, name string) proto.Script {
	script, err := utl.ReadScript(ScriptDir, name)
	require.NoError(suite.T(), err, "unable to read dApp script")
	return script
}*/

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
		10000000,
		utl.GetCurrentTimestampInMs(),
	)
}

func GetSetScriptDataMatrix(suite *f.BaseSuite) map[string]SetScriptData {
	return map[string]SetScriptData{
		"Check invoke script tx": NewSetScriptData(
			utl.GetAccount(suite, utl.DefaultRecipientNotMiner),
			//readDAppScript(suite, "issue_0_assets.base64"),
			getDAppScript(suite, "src1.ride"),
			utl.TestChainID,
			10000000,
			utl.GetCurrentTimestampInMs(),
		),
	}
}
