package alias_utilities

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration, positive bool) utl.ConsideredTransaction

// MakeTxAndGetDiffBalances This function returns txID with init balance before tx and difference balance after tx for both nodes
func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	timeout time.Duration, positive bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	initBalanceGo, initBalanceScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	tx := makeTx(suite, testdata, version, timeout, positive)
	actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceGo, initBalanceScala)
	return *utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
			tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		*utl.NewBalanceInWaves(initBalanceGo, initBalanceScala),
		*utl.NewBalanceInWaves(actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
}

func NewSignAliasTransaction[T any](suite *f.BaseSuite, version byte, testdata testdata.AliasTestData[T]) proto.Transaction {
	var tx proto.Transaction
	alias := proto.NewAlias(testdata.ChainID, testdata.Alias)
	if version == 1 {
		tx = proto.NewUnsignedCreateAliasWithSig(
			testdata.Account.PublicKey, *alias,
			testdata.Fee, testdata.Timestamp)
	} else {
		tx = proto.NewUnsignedCreateAliasWithProofs(version, testdata.Account.PublicKey, *alias,
			testdata.Fee, testdata.Timestamp)
	}
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	suite.T().Logf("Alias Transaction JSON: %s", utl.GetTransactionJsonOrErrMsg(tx))
	require.NoError(suite.T(), err)
	return tx
}

func AliasSend[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransaction(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout, positive)
}

func AliasBroadcast[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransaction(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout, positive)
}

func SendAliasTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration, positive bool) (
	utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, timeout, positive, AliasSend[T])
}

func BroadcastAliasTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration, positive bool) (
	utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, timeout, positive, AliasBroadcast[T])
}
