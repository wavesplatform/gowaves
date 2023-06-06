package update_asset_info_utilities

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignedUpdateAssetInfoTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, name, description string, timestamp, fee uint64, feeAsset proto.OptionalAsset) proto.Transaction {
	tx := proto.NewUnsignedUpdateAssetInfoWithProofs(version, assetID, senderPK, name, description, timestamp, feeAsset, fee)
	err := tx.Sign(scheme, senderSK)
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("UpdateAssetInfo Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func NewSignedUpdateAssetInfoTransactionWithTestData[T any](suite *f.BaseSuite, version byte, testdata testdata.UpdateAssetInfoTestData[T]) proto.Transaction {
	return NewSignedUpdateAssetInfoTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey, testdata.Account.SecretKey,
		testdata.AssetID, testdata.AssetName, testdata.AssetDesc, testdata.Timestamp, testdata.Fee, testdata.FeeAsset)
}

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T], version byte,
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

func UpdateAssetInfoSendWithTestData[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T],
	version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignedUpdateAssetInfoTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func UpdateAssetInfoBroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T],
	version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignedUpdateAssetInfoTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendUpdateAssetInfoTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T],
	version byte, waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, UpdateAssetInfoSendWithTestData[T])
}

func BroadcastUpdateAssetInfoTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T],
	version byte, waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, UpdateAssetInfoBroadcastWithTestData[T])
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.UpdateAssetInfoTransaction, testdata.UpdateAssetInfoMinVersion, testdata.UpdateAssetInfoMaxVersion).Sum
}
