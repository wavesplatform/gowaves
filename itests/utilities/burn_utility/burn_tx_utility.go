package burn_utility

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MadeTx[T any] func(suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, timeout time.Duration, positive bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte,
	timeout time.Duration, positive bool, makedTx MadeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves) {
	initBalanceGo, initBalanceScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	tx := makedTx(suite, testdata, version, timeout, positive)
	actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceGo, initBalanceScala)
	return *utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
			tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		*utl.NewBalanceInWaves(actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
}

func NewSignBurnTransaction[T any](suite *f.BaseSuite, version byte, testdata testdata.BurnTestData[T]) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedBurnWithSig(testdata.Account.PublicKey, testdata.AssetID, testdata.Quantity, testdata.Timestamp, testdata.Fee)
	} else {
		tx = proto.NewUnsignedBurnWithProofs(version, testdata.ChainID, testdata.Account.PublicKey, testdata.AssetID,
			testdata.Quantity, testdata.Timestamp, testdata.Fee)
	}
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Issue Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func BurnSend[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignBurnTransaction(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout, positive)
}

func BurnBroadcast[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignBurnTransaction(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout, positive)
}

func SendBurnTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, timeout time.Duration, positive bool) (
	utl.ConsideredTransaction, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, timeout, positive, BurnSend[T])
}

func BroadcastBurnTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, timeout time.Duration, positive bool) (
	utl.ConsideredTransaction, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, timeout, positive, BurnBroadcast[T])
}
