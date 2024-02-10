package updateassetinfo

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignedUpdateAssetInfoTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme,
	senderPK crypto.PublicKey, senderSK crypto.SecretKey, assetID crypto.Digest, name, description string,
	timestamp, fee uint64, feeAsset proto.OptionalAsset) proto.Transaction {
	tx := proto.NewUnsignedUpdateAssetInfoWithProofs(version, assetID, senderPK, name, description,
		timestamp, feeAsset, fee)
	err := tx.Sign(scheme, senderSK)
	txJSON := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("UpdateAssetInfo Transaction JSON after sign: %s", txJSON)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func NewSignedUpdateAssetInfoTransactionWithTestData[T any](suite *f.BaseSuite, version byte,
	testdata testdata.UpdateAssetInfoTestData[T]) proto.Transaction {
	return NewSignedUpdateAssetInfoTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey,
		testdata.Account.SecretKey, testdata.AssetID, testdata.AssetName, testdata.AssetDesc, testdata.Timestamp,
		testdata.Fee, testdata.FeeAsset)
}

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	initBalanceInAssetGo, initBalanceInAssetScala := utl.GetAssetBalance(suite, testdata.Account.Address,
		testdata.AssetID)
	tx := makeTx(suite, testdata, version, waitForTx)

	actualDiffBalanceInWaves := utl.GetActualDiffBalanceInWaves(suite, testdata.Account.Address,
		initBalanceInWavesGo, initBalanceInWavesScala)
	actualDiffBalanceInAsset := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Account.Address, testdata.AssetID, initBalanceInAssetGo, initBalanceInAssetScala)

	return utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo,
			tx.WtErr.ErrWtScala, tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		utl.NewBalanceInWaves(actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala),
		utl.NewBalanceInAsset(actualDiffBalanceInAsset.BalanceInAssetGo, actualDiffBalanceInAsset.BalanceInAssetScala)
}

func SendWithTestData[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T],
	version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignedUpdateAssetInfoTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func BroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T],
	version byte, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignedUpdateAssetInfoTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendUpdateAssetInfoTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.UpdateAssetInfoTestData[T],
	version byte, waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SendWithTestData[T])
}

func BroadcastUpdateAssetInfoTxAndGetDiffBalances[T any](suite *f.BaseSuite,
	testdata testdata.UpdateAssetInfoTestData[T], version byte, waitForTx bool) (utl.ConsideredTransaction,
	utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, BroadcastWithTestData[T])
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.UpdateAssetInfoTransaction, testdata.UpdateAssetInfoMinVersion,
		testdata.UpdateAssetInfoMaxVersion).Sum
}

func PositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.UpdateAssetInfoTestData[testdata.UpdateAssetInfoExpectedPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	assetDetails utl.AssetInfo, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
	utl.AssetNameCheck(t, td.AssetName, assetDetails.AssetInfoGo.GetName(),
		assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetDescriptionCheck(t, td.AssetDesc, assetDetails.AssetInfoGo.GetDescription(),
		assetDetails.AssetInfoScala.GetDescription(), errMsg)
}

func NegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.UpdateAssetInfoTestData[testdata.UpdateAssetInfoExpectedNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	initAssetDetails, assetDetails utl.AssetInfo, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

	utl.AssetNameCheck(t, initAssetDetails.AssetInfoGo.GetName(), assetDetails.AssetInfoGo.GetName(),
		assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetNameCheck(t, initAssetDetails.AssetInfoScala.GetName(), assetDetails.AssetInfoGo.GetName(),
		assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetDescriptionCheck(t, initAssetDetails.AssetInfoGo.GetDescription(),
		assetDetails.AssetInfoGo.GetDescription(), assetDetails.AssetInfoScala.GetDescription(), errMsg)
	utl.AssetDescriptionCheck(t, initAssetDetails.AssetInfoScala.GetDescription(),
		assetDetails.AssetInfoGo.GetDescription(), assetDetails.AssetInfoScala.GetDescription(), errMsg)
}

func PositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.UpdateAssetInfoTestData[testdata.UpdateAssetInfoExpectedPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	assetDetails utl.AssetInfo, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
	utl.AssetNameCheck(t, td.AssetName, assetDetails.AssetInfoGo.GetName(),
		assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetDescriptionCheck(t, td.AssetDesc, assetDetails.AssetInfoGo.GetDescription(),
		assetDetails.AssetInfoScala.GetDescription(), errMsg)
}

func NegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.UpdateAssetInfoTestData[testdata.UpdateAssetInfoExpectedNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	initAssetDetails, assetDetails utl.AssetInfo, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
		tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)

	utl.AssetNameCheck(t, initAssetDetails.AssetInfoGo.GetName(), assetDetails.AssetInfoGo.GetName(),
		assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetNameCheck(t, initAssetDetails.AssetInfoScala.GetName(), assetDetails.AssetInfoGo.GetName(),
		assetDetails.AssetInfoScala.GetName(), errMsg)
	utl.AssetDescriptionCheck(t, initAssetDetails.AssetInfoGo.GetDescription(),
		assetDetails.AssetInfoGo.GetDescription(), assetDetails.AssetInfoScala.GetDescription(), errMsg)
	utl.AssetDescriptionCheck(t, initAssetDetails.AssetInfoScala.GetDescription(),
		assetDetails.AssetInfoGo.GetDescription(), assetDetails.AssetInfoScala.GetDescription(), errMsg)
}
