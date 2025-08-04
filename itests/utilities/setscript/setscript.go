package setscript

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignedSetScriptTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, script []byte, fee, timestamp uint64) proto.Transaction {
	tx := proto.NewUnsignedSetScriptWithProofs(version, senderPK, script, fee, timestamp)
	err := tx.Sign(scheme, senderSK)
	txJSON := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Set Script Transaction:\n%s", txJSON)
	require.NoError(suite.T(), err, "failed to create proofs for set script tx from signature")
	return tx
}

func NewSignedSetScriptTransactionWithTestData(suite *f.BaseSuite, version byte,
	testdata testdata.SetScriptData) proto.Transaction {
	return NewSignedSetScriptTransaction(suite, version, testdata.ChainID, testdata.SenderAccount.PublicKey,
		testdata.SenderAccount.SecretKey, testdata.Script, testdata.Fee, testdata.Timestamp)
}

func CreateDAppAccount(suite *f.BaseSuite, from int, amount uint64,
	scriptName string) config.AccountInfo {
	accNumber := transfer.GetNewAccountWithFunds(suite, testdata.TransferMaxVersion, utl.TestChainID, from, amount)
	td := testdata.GetDataForDAppAccount(suite, utl.GetAccount(suite, accNumber), scriptName)
	tx := NewSignedSetScriptTransactionWithTestData(suite, testdata.SetScriptMaxVersion, td)
	utl.SendAndWaitTransaction(suite, tx, td.ChainID, true)
	return utl.GetAccount(suite, accNumber)
}
