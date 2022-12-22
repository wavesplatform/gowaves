package issue_utilities

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction

// MakeTxAndGetDiffBalances This function returns txID with init balance before tx and difference balance after tx for both nodes
func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	initBalanceGo, initBalanceScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	tx := makeTx(suite, testdata, version, waitForTx)
	actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceGo, initBalanceScala)
	return *utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
			tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		*utl.NewBalanceInWaves(initBalanceGo, initBalanceScala),
		*utl.NewBalanceInWaves(actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
}
func NewSignIssueTransaction[T any](suite *f.BaseSuite, version byte, testdata testdata.IssueTestData[T]) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedIssueWithSig(testdata.Account.PublicKey, testdata.AssetName,
			testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	} else {
		tx = proto.NewUnsignedIssueWithProofs(version, testdata.ChainID, testdata.Account.PublicKey, testdata.AssetName,
			testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, nil, testdata.Timestamp, testdata.Fee)
	}
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Issue Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func IssueSend[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignIssueTransaction(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func IssueBroadcast[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignIssueTransaction(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendIssueTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, IssueSend[T])
}

func BroadcastIssueTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, IssueBroadcast[T])
}
