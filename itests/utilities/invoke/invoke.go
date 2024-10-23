package invoke

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Need to add [T any]!
func NewSignedInvokeScriptTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, scriptRecipient proto.Recipient, call proto.FunctionCall, payments proto.ScriptPayments,
	feeAsset proto.OptionalAsset, fee, timestamp uint64) proto.Transaction {
	var tx proto.Transaction
	tx = proto.NewUnsignedInvokeScriptWithProofs(version, senderPK, scriptRecipient, call, payments,
		feeAsset, fee, timestamp)
	err := tx.Sign(scheme, senderSK)
	txJSON := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Invoke script transaction json: %s", txJSON)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

type MakeTx func(suite *f.BaseSuite, testdata testdata.InvokeScriptTestData, version byte,
	waitForTx bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances(suite *f.BaseSuite, testdata testdata.InvokeScriptTestData, version byte,
	waitForTx bool, makeTx MakeTx) (utl.ConsideredTransaction, utl.BalanceInWaves) {
	initBalanceGo, initBalanceScala := utl.GetAvailableBalanceInWaves(suite, testdata.SenderAccount.Address)
	tx := makeTx(suite, testdata, version, waitForTx)
	actualDiffBalanceInWaves := utl.GetActualDiffBalanceInWaves(
		suite, testdata.SenderAccount.Address, initBalanceGo, initBalanceScala)
	return utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo,
			tx.WtErr.ErrWtScala, tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		utl.NewBalanceInWaves(actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala)
}

func NewSignedInvokeScriptTransactionWithTestData(suite *f.BaseSuite, version byte,
	testdata testdata.InvokeScriptTestData) proto.Transaction {
	return NewSignedInvokeScriptTransaction(suite, version, testdata.ChainID, testdata.SenderAccount.PublicKey,
		testdata.SenderAccount.SecretKey, testdata.ScriptRecipient, testdata.Call, testdata.Payments, testdata.FeeAsset,
		testdata.Fee, testdata.Timestamp)
}

func SendWithTestData(suite *f.BaseSuite, testdata testdata.InvokeScriptTestData, version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignedInvokeScriptTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func BroadcastWithTestData(suite *f.BaseSuite, testdata testdata.InvokeScriptTestData, version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignedInvokeScriptTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func GetVersionsInvokeScript(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.InvokeScriptTransaction, testdata.InvokeMinVersion,
		testdata.InvokeMaxVersion).Sum
}
