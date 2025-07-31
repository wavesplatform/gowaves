package alias

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	errMsg = "alias transaction failed"
)

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction

// MakeTxAndGetDiffBalances This function returns txID with init balance before tx and difference balance
// after tx for both nodes.
func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	initBalanceGo, initBalanceScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	tx := makeTx(suite, testdata, version, waitForTx)
	actualDiffBalanceInWaves := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceGo, initBalanceScala)
	return utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo,
			tx.WtErr.ErrWtScala, tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		utl.NewBalanceInWaves(initBalanceGo, initBalanceScala),
		utl.NewBalanceInWaves(actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala)
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

func SetAliasToAccount(suite *f.BaseSuite, version byte, scheme proto.Scheme, alias string, account *config.AccountInfo,
	fee uint64) {
	tx := AliasSend(suite, version, scheme, account.PublicKey, account.SecretKey, alias,
		fee, utl.GetCurrentTimestampInMs(), true)
	utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	account.Alias = *utl.GetAliasFromString(suite, alias, scheme)
}

func SetAliasToAccountByAPI(suite *f.BaseSuite, version byte, scheme proto.Scheme, alias string,
	account config.AccountInfo, fee uint64) {
	tx := AliasBroadcast(suite, version, scheme, account.PublicKey, account.SecretKey, alias,
		fee, utl.GetCurrentTimestampInMs(), true)
	utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
}

func AliasBroadcast(suite *f.BaseSuite, version byte, scheme proto.Scheme, accountPK crypto.PublicKey,
	accountSK crypto.SecretKey, aliasStr string, fee, timestamp uint64, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransaction(suite, version, scheme, accountPK, accountSK, aliasStr, fee, timestamp)
	return utl.BroadcastAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func NewSignAliasTransactionWithTestData[T any](suite *f.BaseSuite, version byte,
	testdata testdata.AliasTestData[T]) proto.Transaction {
	return NewSignAliasTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey,
		testdata.Account.SecretKey, testdata.Alias, testdata.Fee, testdata.Timestamp)
}

func SendWithTestData[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func BroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignAliasTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendAliasTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SendWithTestData[T])
}

func BroadcastAliasTxAndGetWavesBalances[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInWaves) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, BroadcastWithTestData[T])
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.CreateAliasTransaction, testdata.AliasMinVersion,
		testdata.AliasMaxVersion).Sum
}

func PositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.AliasTestData[testdata.AliasExpectedValuesPositive], addrByAliasGo, addrByAliasScala []byte,
	actualDiffBalanceInWaves utl.BalanceInWaves, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.AddressByAliasCheck(t, td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala,
		errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
}

func NegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.AliasTestData[testdata.AliasExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
}

func PositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.AliasTestData[testdata.AliasExpectedValuesPositive], addrByAliasGo, addrByAliasScala []byte,
	actualDiffBalanceInWaves utl.BalanceInWaves, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.AddressByAliasCheck(t, td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala,
		errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
}

func NegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.AliasTestData[testdata.AliasExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
		tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
}
