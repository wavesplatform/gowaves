package alias_utilities

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction

// MakeTxAndGetDiffBalances This function returns txID with init balance before tx and difference balance after tx for both nodes
func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
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

func NewSignAliasTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, accountPK crypto.PublicKey,
	accountSK crypto.SecretKey, aliasStr string, fee, timestamp uint64) proto.Transaction {
	var tx proto.Transaction
	alias := proto.NewAlias(scheme, aliasStr)
	if version == 1 {
		tx = proto.NewUnsignedCreateAliasWithSig(accountPK, *alias, fee, timestamp)
	} else {
		tx = proto.NewUnsignedCreateAliasWithProofs(version, accountPK, *alias, fee, timestamp)
	}
	err := tx.Sign(scheme, accountSK)
	suite.T().Logf("Alias Transaction JSON: %s", utl.GetTransactionJsonOrErrMsg(tx))
	require.NoError(suite.T(), err)
	return tx
}

func AliasSend(suite *f.BaseSuite, version byte, scheme proto.Scheme, accountPK crypto.PublicKey,
	accountSK crypto.SecretKey, aliasStr string, fee, timestamp uint64, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransaction(suite, version, scheme, accountPK, accountSK, aliasStr, fee, timestamp)
	return utl.SendAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func SetAliasToAccount(suite *f.BaseSuite, version byte, scheme proto.Scheme, alias string, accNumber int) {
	account := utl.GetAccount(suite, accNumber)
	AliasSend(suite, version, scheme, account.PublicKey, account.SecretKey, alias,
		100000, utl.GetCurrentTimestampInMs(), true)
}

func SetAliasToAccountByAPI(suite *f.BaseSuite, version byte, scheme proto.Scheme, alias string, accNumber int) {
	account := utl.GetAccount(suite, accNumber)
	AliasBroadcast(suite, version, scheme, account.PublicKey, account.SecretKey, alias,
		100000, utl.GetCurrentTimestampInMs(), true)
}

func AliasBroadcast(suite *f.BaseSuite, version byte, scheme proto.Scheme, accountPK crypto.PublicKey,
	accountSK crypto.SecretKey, aliasStr string, fee, timestamp uint64, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransaction(suite, version, scheme, accountPK, accountSK, aliasStr, fee, timestamp)
	return utl.BroadcastAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func NewSignAliasTransactionWithTestData[T any](suite *f.BaseSuite, version byte, testdata testdata.AliasTestData[T]) proto.Transaction {
	return NewSignAliasTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey,
		testdata.Account.SecretKey, testdata.Alias, testdata.Fee, testdata.Timestamp)
}

func AliasSendWithTestData[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func AliasBroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendAliasTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, AliasSendWithTestData[T])
}

func BroadcastAliasTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, AliasBroadcastWithTestData[T])
}
