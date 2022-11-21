package issue_utilities

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MadeTx[T any] func(suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, timeout time.Duration) utl.ConsideredTransaction

// MakeTxAndGetDiffBalances This function returns txID with init balance before tx and difference balance after tx for both nodes
func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	timeout time.Duration, makedTx MadeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	initBalanceGo, initBalanceScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	tx := makedTx(suite, testdata, version, timeout)
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
	suite.T().Logf("Issue Transaction JSON: %s", utl.GetTransactionJsonOrErrMsg(tx))
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func IssueSend[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, timeout time.Duration) utl.ConsideredTransaction {
	tx := NewSignIssueTransaction(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
}

/*func IssueBroadcast[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, timeout time.Duration) utl.ConsideredTransaction {
	tx := NewSignIssueTransaction(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
}*/

func SendIssueTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, timeout time.Duration) (
	utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, timeout, IssueSend[T])
}

/*func BroadcastIssueTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, timeout time.Duration) (
	utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, timeout, IssueBroadcast[T])
}*/

func IssueBroadcast[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, timeout time.Duration) (
	utl.BroadcastedTransaction, error, error) {
	tx := NewSignIssueTransaction(suite, version, testdata)
	brdCstTx, errGo, errScala := utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return brdCstTx, errGo, errScala
}
