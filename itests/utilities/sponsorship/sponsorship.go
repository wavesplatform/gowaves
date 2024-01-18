package sponsorship

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

func NewSignSponsorshipTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, minSponsoredAssetFee, fee, timestamp uint64) proto.Transaction {
	tx := proto.NewUnsignedSponsorshipWithProofs(version, senderPK, assetID, minSponsoredAssetFee, fee, timestamp)
	err := tx.Sign(scheme, senderSK)
	txJSON := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Sponsorship Transaction JSON after sign: %s", txJSON)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func Send(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, minAssetFee, fee, timestamp uint64,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransaction(suite, version, scheme, senderPK, senderSK, assetID, minAssetFee,
		fee, timestamp)
	return utl.SendAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func Broadcast(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, minAssetFee, fee, timestamp uint64,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransaction(suite, version, scheme, senderPK, senderSK, assetID, minAssetFee,
		fee, timestamp)
	return utl.BroadcastAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func NewSignSponsorshipTransactionWithTestData[T any](suite *f.BaseSuite, version byte,
	testdata testdata.SponsorshipTestData[T]) proto.Transaction {
	return NewSignSponsorshipTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey,
		testdata.Account.SecretKey, testdata.AssetID, testdata.MinSponsoredAssetFee, testdata.Fee, testdata.Timestamp)
}

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitFor bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
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

func SendWithTestData[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func BroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendSponsorshipTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SendWithTestData[T])
}

func BroadcastSponsorshipTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T],
	version byte, waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, BroadcastWithTestData[T])
}

func EnableSend(suite *f.BaseSuite, version byte, scheme proto.Scheme, assetID crypto.Digest,
	minAssetFee uint64) {
	assetDetails := utl.GetAssetInfo(suite, assetID)
	issuer := utl.MustGetAccountByAddress(suite, assetDetails.Issuer)
	Send(suite, version, scheme, issuer.PublicKey, issuer.SecretKey, assetID, minAssetFee,
		utl.MinTxFeeWaves, utl.GetCurrentTimestampInMs(), true)
}

func EnableBroadcast(suite *f.BaseSuite, version byte, scheme proto.Scheme, assetID crypto.Digest,
	minAssetFee uint64) {
	assetDetails := utl.GetAssetInfo(suite, assetID)
	issuer := utl.MustGetAccountByAddress(suite, assetDetails.Issuer)
	Broadcast(suite, version, scheme, issuer.PublicKey, issuer.SecretKey, assetID, minAssetFee,
		utl.MinTxFeeWaves, utl.GetCurrentTimestampInMs(), true)
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.SponsorshipTransaction, testdata.SponsorshipMinVersion,
		testdata.SponsorshipMaxVersion).Sum
}

func PositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SponsorshipTestData[testdata.SponsorshipExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func NegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SponsorshipTestData[testdata.SponsorshipExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func PositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SponsorshipTestData[testdata.SponsorshipExpectedValuesPositive],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func NegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.SponsorshipTestData[testdata.SponsorshipExpectedValuesNegative],
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
