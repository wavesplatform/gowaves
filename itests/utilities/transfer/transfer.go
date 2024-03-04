package transfer

import (
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/node_client"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignTransferTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, amountAsset, feeAsset proto.OptionalAsset, timestamp, amount, fee uint64,
	recipient proto.Recipient, attachment proto.Attachment) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedTransferWithSig(senderPK, amountAsset, feeAsset, timestamp, amount, fee,
			recipient, attachment)
	} else {
		tx = proto.NewUnsignedTransferWithProofs(version, senderPK, amountAsset, feeAsset, timestamp, amount,
			fee, recipient, attachment)
	}
	err := tx.Sign(scheme, senderSK)
	txJSON := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Transfer Transaction JSON after sign: %s", txJSON)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func Send(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	sk crypto.SecretKey, amountAsset, feeAsset proto.OptionalAsset, timestamp, amount, fee uint64,
	recipient proto.Recipient, attachment proto.Attachment, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransaction(suite, version, scheme, senderPK, sk, amountAsset, feeAsset, timestamp, amount,
		fee, recipient, attachment)
	return utl.SendAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func NewSignTransferTransactionWithTestData[T any](suite *f.BaseSuite, version byte,
	testdata testdata.TransferTestData[T]) proto.Transaction {
	return NewSignTransferTransaction(suite, version, testdata.ChainID, testdata.Sender.PublicKey,
		testdata.Sender.SecretKey, testdata.Asset, testdata.FeeAsset, testdata.Timestamp, testdata.Amount,
		testdata.Fee, testdata.Recipient, testdata.Attachment)
}

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction

func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T],
	version byte, waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction,
	utl.AccountsDiffBalancesTxWithSponsorship) {
	var assetDetails *client.AssetsDetail
	if testdata.FeeAsset.Present {
		assetDetails = utl.GetAssetInfo(suite, testdata.FeeAsset.ID)
	}
	address := utl.GetAddressFromRecipient(suite, testdata.Recipient)

	initBalanceWavesGoSender, initBalanceWavesScalaSender :=
		utl.GetAvailableBalanceInWaves(suite, testdata.Sender.Address)
	initBalanceAssetGoSender, initBalanceAssetScalaSender :=
		utl.GetAssetBalance(suite, testdata.Sender.Address, testdata.Asset.ID)
	initBalanceFeeAssetGoSender, initBalanceFeeAssetScalaSender :=
		utl.GetAssetBalance(suite, testdata.Sender.Address, testdata.FeeAsset.ID)

	initBalanceWavesGoRecipient, initBalanceWavesScalaRecipient :=
		utl.GetAvailableBalanceInWaves(suite, address)
	initBalanceAssetGoRecipient, initBalanceAssetScalaRecipient :=
		utl.GetAssetBalance(suite, address, testdata.Asset.ID)

	var initBalanceWavesGoSponsor, initBalanceWavesScalaSponsor,
		initBalanceAssetGoSponsor, initBalanceAssetScalaSponsor int64
	if assetDetails != nil {
		initBalanceWavesGoSponsor, initBalanceWavesScalaSponsor =
			utl.GetAvailableBalanceInWaves(suite, assetDetails.Issuer)
		initBalanceAssetGoSponsor, initBalanceAssetScalaSponsor =
			utl.GetAssetBalance(suite, assetDetails.Issuer, testdata.FeeAsset.ID)
	}

	tx := makeTx(suite, testdata, version, waitForTx)

	actualDiffBalanceWavesSender := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Sender.Address, initBalanceWavesGoSender, initBalanceWavesScalaSender)
	actualDiffBalanceAssetSender := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Sender.Address, testdata.Asset.ID, initBalanceAssetGoSender, initBalanceAssetScalaSender)
	actualDiffBalanceFeeAssetSender := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Sender.Address, testdata.FeeAsset.ID, initBalanceFeeAssetGoSender, initBalanceFeeAssetScalaSender)

	actualDiffBalanceWavesRecipient := utl.GetActualDiffBalanceInWaves(
		suite, address, initBalanceWavesGoRecipient, initBalanceWavesScalaRecipient)
	actualDiffBalanceAssetRecipient := utl.GetActualDiffBalanceInAssets(suite,
		address, testdata.Asset.ID, initBalanceAssetGoRecipient, initBalanceAssetScalaRecipient)

	var actualDiffBalanceWavesSponsor utl.BalanceInWaves
	var actualDiffBalanceAssetSponsor utl.BalanceInAsset
	if assetDetails != nil {
		actualDiffBalanceWavesSponsor = utl.GetActualDiffBalanceInWaves(suite,
			assetDetails.Issuer, initBalanceWavesGoSponsor, initBalanceWavesScalaSponsor)
		actualDiffBalanceAssetSponsor = utl.GetActualDiffBalanceInAssets(suite,
			assetDetails.Issuer, testdata.FeeAsset.ID, initBalanceAssetGoSponsor, initBalanceAssetScalaSponsor)
	}

	return utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo,
			tx.WtErr.ErrWtScala, tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		utl.NewDiffBalancesTxWithSponsorship(actualDiffBalanceWavesSender.BalanceInWavesGo,
			actualDiffBalanceWavesSender.BalanceInWavesScala, actualDiffBalanceAssetSender.BalanceInAssetGo,
			actualDiffBalanceAssetSender.BalanceInAssetScala, actualDiffBalanceFeeAssetSender.BalanceInAssetGo,
			actualDiffBalanceFeeAssetSender.BalanceInAssetScala, actualDiffBalanceWavesRecipient.BalanceInWavesGo,
			actualDiffBalanceWavesRecipient.BalanceInWavesScala, actualDiffBalanceAssetRecipient.BalanceInAssetGo,
			actualDiffBalanceAssetRecipient.BalanceInAssetScala, actualDiffBalanceWavesSponsor.BalanceInWavesGo,
			actualDiffBalanceWavesSponsor.BalanceInWavesScala, actualDiffBalanceAssetSponsor.BalanceInAssetGo,
			actualDiffBalanceAssetSponsor.BalanceInAssetScala)
}

func SendWithTestData[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func BroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendTransferTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.AccountsDiffBalancesTxWithSponsorship) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, SendWithTestData[T])
}

func BroadcastTransferTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.AccountsDiffBalancesTxWithSponsorship) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, BroadcastWithTestData[T])
}

func TransferringFunds(suite *f.BaseSuite, version byte, scheme proto.Scheme, from, to int,
	amount uint64) utl.ConsideredTransaction {
	sender := utl.GetAccount(suite, from)
	recipient := utl.GetAccount(suite, to)
	tx := Send(suite, version, scheme, sender.PublicKey, sender.SecretKey,
		proto.NewOptionalAssetWaves(), proto.NewOptionalAssetWaves(), utl.GetCurrentTimestampInMs(), amount,
		utl.MinTxFeeWaves, proto.NewRecipientFromAddress(recipient.Address), nil, true)
	return tx
}

func GetNewAccountWithFunds(suite *f.BaseSuite, version byte, scheme proto.Scheme, from int, amount uint64) int {
	accNumber, _ := utl.AddNewAccount(suite, scheme)
	tx := TransferringFunds(suite, version, scheme, from, accNumber, amount)
	require.NoError(suite.T(), tx.WtErr.ErrWtGo, "Reached deadline of Transfer tx in Go")
	require.NoError(suite.T(), tx.WtErr.ErrWtScala, "Reached deadline of Transfer tx in Scala")
	// Waiting for changing waves balance.
	err := node_client.Retry(utl.DefaultTimeInterval, func() error {
		balanceGo, balanceScala := utl.GetAvailableBalanceInWaves(suite, utl.GetAccount(suite, accNumber).Address)
		if balanceScala == 0 && balanceGo == 0 {
			return errors.New("account Waves balance is empty")
		}
		return nil
	})
	require.NoError(suite.T(), err)
	return accNumber
}

// TransferringAssetAmount - Amount of Asset that transferred from one account to another,
// by default it will be all amount of Asset.
func TransferringAssetAmount(suite *f.BaseSuite, version byte, scheme proto.Scheme, assetID crypto.Digest,
	from, to int, assetAmount ...uint64) {
	var amount, currentAmount uint64
	currentAmount = uint64(utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, from).Address, assetID))
	if len(assetAmount) == 1 && assetAmount[0] <= currentAmount {
		amount = assetAmount[0]
	} else {
		amount = currentAmount
	}
	tx := Send(suite, version, scheme, utl.GetAccount(suite, from).PublicKey,
		utl.GetAccount(suite, from).SecretKey, *proto.NewOptionalAssetFromDigest(assetID),
		proto.NewOptionalAssetWaves(), utl.GetCurrentTimestampInMs(), amount, utl.MinTxFeeWaves,
		proto.NewRecipientFromAddress(utl.GetAccount(suite, to).Address), nil, true)
	require.NoError(suite.T(), tx.WtErr.ErrWtGo, "Reached deadline of Transfer tx in Go")
	require.NoError(suite.T(), tx.WtErr.ErrWtScala, "Reached deadline of Transfer tx in Scala")
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.TransferTransaction, testdata.TransferMinVersion,
		testdata.TransferMaxVersion).Sum
}

func PositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, errMsg)
}

func NegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesNegative],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, errMsg)
}

func BaseNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func PositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, errMsg)
}

func NegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesNegative],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
		tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala, errMsg)
}

func BaseNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferExpectedValuesNegative],
	actualDiffBalanceInWaves utl.BalanceInWaves, actualDiffBalanceInAsset utl.BalanceInAsset, errMsg string) {
	utl.ErrorMessageCheck(t, td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
		actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
		actualDiffBalanceInAsset.BalanceInAssetScala, errMsg)
}

func WithSponsorshipPositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferSponsoredExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	// RecipientSender balance in Waves does not change because of fee in sponsored asset.
	// RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient.
	// The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee.
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)
	// Recipient balance in Waves changes if Waves were transferred.
	// Recipient Asset balance increases by the asset amount being transferred.
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	// Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee.
	// Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender.
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}

func WithSponsorshipMinAssetFeePositiveChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferSponsoredTestData[testdata.TransferSponsoredExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	// RecipientSender balance in Waves does not change because of fee in sponsored asset.
	// RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient.
	// The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)
	// Recipient balance in Waves changes if Waves were transferred.
	// Recipient Asset balance increases by the asset amount being transferred.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	// Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee.
	// Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}

func WithSponsorshipNegativeChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferSponsoredTestData[testdata.TransferSponsoredExpectedValuesNegative],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.ErrorMessageCheck(t, td.TransferTestData.Expected.ErrGoMsg, td.TransferTestData.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

	// Balances of RecipientSender do not change.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)

	// Balances of Recipient do not change.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)

	// Balances of Sponsor do not change.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}

func WithSponsorshipPositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferTestData[testdata.TransferSponsoredExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	// RecipientSender balance in Waves does not change because of fee in sponsored asset.
	// RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient.
	// The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee.
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)
	// Recipient balance in Waves changes if Waves were transferred.
	// Recipient Asset balance increases by the asset amount being transferred.
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	// Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee.
	// Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender.
	utl.WavesDiffBalanceCheck(t, td.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}

func WithSponsorshipMinAssetFeePositiveAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferSponsoredTestData[testdata.TransferSponsoredExpectedValuesPositive],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusOK, http.StatusOK, tx, errMsg)
	utl.TxInfoCheck(t, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
	// RecipientSender balance in Waves does not change because of fee in sponsored asset.
	// RecipientSender balance of tokens (waves) is reduced by the amount of tokens that transferred to Recipient.
	// The RecipientSender's balance of tokens specified as an asset fee is reduced by the amount of the fee.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)
	// Recipient balance in Waves changes if Waves were transferred.
	// Recipient Asset balance increases by the asset amount being transferred.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
	// Sponsor balance in Waves decreases by amount feeInWaves = feeInSponsoredAsset × 0,001 / minSponsoredAssetFee.
	// Sponsor Asset balance increases by amount of fee in sponsored asset that was used by RecipientSender.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)
	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}

func WithSponsorshipNegativeAPIChecks(t *testing.T, tx utl.ConsideredTransaction,
	td testdata.TransferSponsoredTestData[testdata.TransferSponsoredExpectedValuesNegative],
	diffBalances utl.AccountsDiffBalancesTxWithSponsorship, errMsg string) {
	utl.StatusCodesCheck(t, http.StatusInternalServerError, http.StatusBadRequest, tx, errMsg)
	utl.ErrorMessageCheck(t, td.TransferTestData.Expected.ErrGoMsg, td.TransferTestData.Expected.ErrScalaMsg,
		tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

	// Balances of RecipientSender do not change.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSender.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.FeeAssetDiffBalanceSender,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSender.DiffBalanceFeeAsset.BalanceInAssetScala,
		errMsg)

	// Balances of Recipient do not change.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceRecipient,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesRecipient.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)

	// Balances of Sponsor do not change.
	utl.WavesDiffBalanceCheck(t, td.TransferTestData.Expected.WavesDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceWaves.BalanceInWavesScala,
		errMsg)

	utl.AssetDiffBalanceCheck(t, td.TransferTestData.Expected.AssetDiffBalanceSponsor,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetGo,
		diffBalances.DiffBalancesSponsor.DiffBalanceAsset.BalanceInAssetScala,
		errMsg)
}
