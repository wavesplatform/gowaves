package reissue

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.ReissueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction

// MakeTxAndGetDiffBalances This function returns txID with difference balances after tx for both nodes.
func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.ReissueTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	initBalanceInAssetGo, initBalanceInAssetScala := utl.GetAssetBalance(suite, testdata.Account.Address,
		testdata.AssetID)
	tx := makeTx(suite, testdata, version, waitForTx)
	actualDiffBalanceInWaves := utl.GetActualDiffBalanceInWaves(suite, testdata.Account.Address,
		initBalanceInWavesGo, initBalanceInWavesScala)
	actualDiffBalanceInAsset := utl.GetActualDiffBalanceInAssets(suite, testdata.Account.Address,
		testdata.AssetID, initBalanceInAssetGo, initBalanceInAssetScala)
	return utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo,
			tx.WtErr.ErrWtScala, tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		utl.NewBalanceInWaves(actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala),
		utl.NewBalanceInAsset(actualDiffBalanceInAsset.BalanceInAssetGo, actualDiffBalanceInAsset.BalanceInAssetScala)
}

func NewSignReissueTransaction[T any](suite *f.BaseSuite, version byte,
	testdata testdata.ReissueTestData[T]) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedReissueWithSig(
			testdata.Account.PublicKey, testdata.AssetID, testdata.Quantity, testdata.Reissuable,
			testdata.Timestamp, testdata.Fee)
	} else {
		tx = proto.NewUnsignedReissueWithProofs(version, testdata.Account.PublicKey, testdata.AssetID,
			testdata.Quantity, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	}
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	txJSON := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Reissue Transaction JSON: %s", txJSON)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func Send[T any](suite *f.BaseSuite, testdata testdata.ReissueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignReissueTransaction(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func Broadcast[T any](suite *f.BaseSuite, testdata testdata.ReissueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignReissueTransaction(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendReissueTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.ReissueTestData[T],
	version byte, waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, Send[T])
}

func BroadcastReissueTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.ReissueTestData[T],
	version byte, waitForTx bool) (
	utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, Broadcast[T])
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.ReissueTransaction, testdata.ReissueMinVersion,
		testdata.ReissueMaxVersion).Sum
}

func PositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.ReissueTestData[testdata.ReissueExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func NegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.ReissueTestData[testdata.ReissueExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func PositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.ReissueTestData[testdata.ReissueExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func NegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.ReissueTestData[testdata.ReissueExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
		tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}
