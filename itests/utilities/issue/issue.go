package issue

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignIssueTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, name, description string, quantity, timestamp, fee uint64, decimals byte,
	reissuable bool, script proto.Script) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedIssueWithSig(senderPK, name, description, quantity, decimals, reissuable, timestamp, fee)
	} else {
		tx = proto.NewUnsignedIssueWithProofs(version, senderPK, name, description, quantity, decimals,
			reissuable, script, timestamp, fee)
	}
	err := tx.Sign(scheme, senderSK)
	txJSON := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Issue Transaction JSON after sign: %s", txJSON)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func Send(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, name, description string, quantity, timestamp, fee uint64, decimals byte,
	reissuable, waitForTx bool, script proto.Script) utl.ConsideredTransaction {
	tx := NewSignIssueTransaction(suite, version, scheme, senderPK, senderSK, name, description, quantity,
		timestamp, fee, decimals, reissuable, script)
	return utl.SendAndWaitTransaction(suite, tx, scheme, waitForTx)
}

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction

// MakeTxAndGetDiffBalances This function returns txID with init balance before tx
// and difference balance after tx for both nodes.
func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	initBalanceGo, initBalanceScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	tx := makeTx(suite, testdata, version, waitForTx)
	actualDiffBalanceInWaves := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceGo, initBalanceScala)
	actualDiffBalanceInAsset := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Account.Address, tx.TxID, 0, 0)
	return utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo,
			tx.WtErr.ErrWtScala, tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		utl.NewBalanceInWaves(actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala),
		utl.NewBalanceInAsset(actualDiffBalanceInAsset.BalanceInAssetGo, actualDiffBalanceInAsset.BalanceInAssetScala)
}

func NewSignIssueTransactionWithTestData[T any](suite *f.BaseSuite, version byte,
	testdata testdata.IssueTestData[T]) proto.Transaction {
	return NewSignIssueTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey,
		testdata.Account.SecretKey, testdata.AssetName, testdata.AssetDesc, testdata.Quantity, testdata.Timestamp,
		testdata.Fee, testdata.Decimals, testdata.Reissuable, testdata.Script)
}

func SendWithTestData[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignIssueTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func BroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignIssueTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendIssueTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SendWithTestData[T])
}

func BroadcastIssueTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, BroadcastWithTestData[T])
}

func IssuedAssetAmount(suite *f.BaseSuite, version byte, scheme proto.Scheme, accountNumber int,
	assetAmount ...uint64) crypto.Digest {
	var amount uint64
	if len(assetAmount) == 1 {
		amount = assetAmount[0]
	} else {
		amount = 1
	}
	tx := Send(suite, version, scheme, utl.GetAccount(suite, accountNumber).PublicKey,
		utl.GetAccount(suite, accountNumber).SecretKey, "Asset", "Common Asset for testing", amount,
		utl.GetCurrentTimestampInMs(), utl.MinIssueFeeWaves, utl.MaxDecimals, true, true, nil)
	if tx.WtErr.ErrWtScala != nil || tx.WtErr.ErrWtGo != nil {
		suite.FailNowf("failed to issue assets", "asset: %s, amount: %d, with errors: Scala: %s, Go: %S",
			tx.TxID, amount, tx.WtErr.ErrWtScala, tx.WtErr.ErrWtGo)
	}
	return tx.TxID
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.IssueTransaction, testdata.IssueMinVersion,
		testdata.IssueMaxVersion).Sum
}

func GetVersionsSmartAsset(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.IssueTransaction, testdata.IssueSmartAssetMinVersion,
		testdata.IssueMaxVersion).Sum
}

func GetSmartAssetMatrix(suite *f.BaseSuite, casesCount int) [][]crypto.Digest {
	smartAssetData := testdata.GetCommonIssueData(suite).Smart
	issueVersions := GetVersionsSmartAsset(suite)
	return getAssetsMatrix(suite, smartAssetData, issueVersions, casesCount)
}

func GetNFTMatrix(suite *f.BaseSuite, casesCount int) [][]crypto.Digest {
	nftData := testdata.GetCommonIssueData(suite).NFT
	issueVersions := GetVersions(suite)
	return getAssetsMatrix(suite, nftData, issueVersions, casesCount)
}

func GetReissuableMatrix(suite *f.BaseSuite, casesCount int) [][]crypto.Digest {
	reissuableData := testdata.GetCommonIssueData(suite).Reissuable
	issueVersions := GetVersions(suite)
	return getAssetsMatrix(suite, reissuableData, issueVersions, casesCount)
}

// getAssetsMatrix issues [len(issueVersions)][casesCount]crypto.Digests assets.
func getAssetsMatrix(suite *f.BaseSuite, data testdata.IssueTestData[testdata.ExpectedValuesPositive],
	issueVersions []byte, casesCount int) [][]crypto.Digest {
	txIds := make(map[string]*crypto.Digest)
	matrix := make([][]crypto.Digest, len(issueVersions))
	for i := 0; i < len(issueVersions); i++ {
		matrix[i] = make([]crypto.Digest, casesCount)
		for j := 0; j < casesCount; j++ {
			itx := SendWithTestData(suite, testdata.DataChangedTimestamp(&data), issueVersions[i], false)
			matrix[i][j] = itx.TxID
			name := fmt.Sprintf("i: %d, j: %d", i, j)
			txIds[name] = &itx.TxID
		}
	}
	actualTxIDs := utl.GetTxIdsInBlockchain(suite, txIds)
	suite.Lenf(actualTxIDs, len(txIds)*2, "IDs: %#v", actualTxIDs)
	return matrix
}

func PositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func NegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func PositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func NegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
		tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func SmartAssetPositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	actualScriptGo, actualScriptScala []byte, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
	utl.AssetScriptCheck(t, td.Script, actualScriptGo, actualScriptScala, errMsg)
}

func SmartAssetNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
		tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func SmartAssetPositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset,
	actualScriptGo, actualScriptScala []byte, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
	utl.AssetScriptCheck(t, td.Script, actualScriptGo, actualScriptScala, errMsg)
}

func SmartAssetNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.IssueTestData[testdata.ExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}
