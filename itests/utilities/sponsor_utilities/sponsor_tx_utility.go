package sponsor_utilities

import (
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
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Sponsorship Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func SponsorshipSend(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, minAssetFee, fee, timestamp uint64,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransaction(suite, version, scheme, senderPK, senderSK, assetID, minAssetFee, fee, timestamp)
	return utl.SendAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func SponsorshipBroadcast(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, minAssetFee, fee, timestamp uint64,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransaction(suite, version, scheme, senderPK, senderSK, assetID, minAssetFee, fee, timestamp)
	return utl.BroadcastAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func NewSignSponsorshipTransactionWithTestData[T any](suite *f.BaseSuite, version byte, testdata testdata.SponsorshipTestData[T]) proto.Transaction {
	return NewSignSponsorshipTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey, testdata.Account.SecretKey,
		testdata.AssetID, testdata.MinSponsoredAssetFee, testdata.Fee, testdata.Timestamp)
}

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitFor bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {

	initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	initBalanceInAssetGo, initBalanceInAssetScala := utl.GetAssetBalance(suite, testdata.Account.Address, testdata.AssetID)

	tx := makeTx(suite, testdata, version, waitForTx)

	actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)
	actualDiffBalanceInAssetGo, actualDiffBalanceInAssetScala := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Account.Address, testdata.AssetID, initBalanceInAssetGo, initBalanceInAssetScala)

	return *utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
			tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		*utl.NewBalanceInWaves(actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala),
		*utl.NewBalanceInAsset(actualDiffBalanceInAssetGo, actualDiffBalanceInAssetScala)
}

func SponsorshipSendWithTestData[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SponsorshipBroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendSponsorshipTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SponsorshipSendWithTestData[T])
}

func BroadcastSponsorshipTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.SponsorshipTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SponsorshipBroadcastWithTestData[T])
}

func SponsorshipEnableSend(suite *f.BaseSuite, version byte, scheme proto.Scheme, assetId crypto.Digest, minAssetFee uint64) {
	assetDetails := utl.GetAssetInfo(suite, assetId)
	issuer := utl.MustGetAccountByAddress(suite, assetDetails.Issuer)
	SponsorshipSend(suite, version, scheme, issuer.PublicKey, issuer.SecretKey, assetId, minAssetFee,
		utl.MinTxFeeWaves, utl.GetCurrentTimestampInMs(), true)
}

func SponsorshipEnableBroadcast(suite *f.BaseSuite, version byte, scheme proto.Scheme, assetId crypto.Digest, minAssetFee uint64) {
	assetDetails := utl.GetAssetInfo(suite, assetId)
	issuer := utl.MustGetAccountByAddress(suite, assetDetails.Issuer)
	SponsorshipBroadcast(suite, version, scheme, issuer.PublicKey, issuer.SecretKey, assetId, minAssetFee,
		utl.MinTxFeeWaves, utl.GetCurrentTimestampInMs(), true)
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.SponsorshipTransaction, testdata.SponsorshipMinVersion, testdata.SponsorshipMaxVersion).Sum
}
