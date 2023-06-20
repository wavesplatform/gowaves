package set_asset_script_utilities

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignedSetAssetScriptTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, script proto.Script, fee, timestamp uint64) proto.Transaction {
	tx := proto.NewUnsignedSetAssetScriptWithProofs(version, senderPK, assetID, script, fee, timestamp)
	err := tx.Sign(scheme, senderSK)
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Set Asset Script Transaction after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func NewSignedSetAssetScriptTransactionWithTestData[T any](suite *f.BaseSuite, version byte,
	testdata testdata.SetAssetScriptTestData[T]) proto.Transaction {
	return NewSignedSetAssetScriptTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey,
		testdata.Account.SecretKey, testdata.AssetID, testdata.Script, testdata.Fee, testdata.Timestamp)
}

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.SetAssetScriptTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.SetAssetScriptTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	initBalanceInAssetGo, initBalanceInAssetScala := utl.GetAssetBalance(suite, testdata.Account.Address, testdata.AssetID)

	tx := makeTx(suite, testdata, version, waitForTx)

	actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)
	actualDiffBalanceInAssetGo, actualDiffBalanceInAssetScala := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Account.Address, testdata.AssetID, initBalanceInAssetGo, initBalanceInAssetScala)

	return utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
			tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		utl.NewBalanceInWaves(actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala),
		utl.NewBalanceInAsset(actualDiffBalanceInAssetGo, actualDiffBalanceInAssetScala)
}

func SetAssetScriptSendWithTestData[T any](suite *f.BaseSuite, testdata testdata.SetAssetScriptTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignedSetAssetScriptTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SetAssetScriptBroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.SetAssetScriptTestData[T],
	version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignedSetAssetScriptTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendSetAssetScriptTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.SetAssetScriptTestData[T],
	version byte, waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SetAssetScriptSendWithTestData[T])
}

func BroadcastSetAssetScriptTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.SetAssetScriptTestData[T],
	version byte, waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SetAssetScriptBroadcastWithTestData[T])
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.SetAssetScriptTransaction, testdata.SetAssetScriptMinVersion, testdata.SetAssetScriptMaxVersion).Sum
}
