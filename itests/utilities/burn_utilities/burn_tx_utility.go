package burn_utilities

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	initBalanceInAssetGo, initBalanceInAssetScala := utl.GetAssetBalance(suite, testdata.Account.Address, testdata.AssetID)
	tx := makeTx(suite, testdata, version, waitForTx)
	actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)
	actuallDiffBalanceInAssetGo, actualDiffBalanceInAssetScala := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Account.Address, testdata.AssetID, initBalanceInAssetGo, initBalanceInAssetScala)
	return *utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
			tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		*utl.NewBalanceInWaves(actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala),
		*utl.NewBalanceInAsset(actuallDiffBalanceInAssetGo, actualDiffBalanceInAssetScala)
}

func NewSignBurnTransaction[T any](suite *f.BaseSuite, version byte, testdata testdata.BurnTestData[T]) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedBurnWithSig(testdata.Account.PublicKey, testdata.AssetID, testdata.Quantity, testdata.Timestamp, testdata.Fee)
	} else {
		tx = proto.NewUnsignedBurnWithProofs(version, testdata.Account.PublicKey, testdata.AssetID, testdata.Quantity, testdata.Timestamp, testdata.Fee)
	}
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Burn Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func BurnSend[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignBurnTransaction(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func BurnBroadcast[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignBurnTransaction(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendBurnTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, waitForTx bool) (
	utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, BurnSend[T])
}

func BroadcastBurnTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.BurnTestData[T], version byte, waitForTx bool) (
	utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, BurnBroadcast[T])
}
