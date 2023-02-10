package transfer_utilities

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/node_client"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
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
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Transfer Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func TransferSend(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
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
	version byte, waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.AccountDiffBalances, utl.AccountDiffBalances) {
	address := utl.GetAddressFromRecipient(suite, testdata.Recipient)

	initBalanceWavesGoSender, initBalanceWavesScalaSender := utl.GetAvailableBalanceInWaves(suite, testdata.Sender.Address)
	initBalanceAssetGoSender, initBalanceAssetScalaSender := utl.GetAssetBalance(suite, testdata.Sender.Address, testdata.Asset.ID)

	initBalanceWavesGoRecipient, initBalanceWavesScalaRecipient := utl.GetAvailableBalanceInWaves(suite, address)
	initBalanceAssetGoRecipient, initBalanceAssetScalaRecipient := utl.GetAssetBalance(suite, address, testdata.Asset.ID)

	tx := makeTx(suite, testdata, version, waitForTx)

	actualDiffBalanceWavesGoSender, actualDiffBalanceWavesScalaSender := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Sender.Address, initBalanceWavesGoSender, initBalanceWavesScalaSender)

	actualDiffBalanceAssetGoSender, actualDiffBalanceAssetScalaSender := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Sender.Address, testdata.Asset.ID, initBalanceAssetGoSender, initBalanceAssetScalaSender)

	actualDiffBalanceWavesGoRecipient, actualDiffBalanceWavesScalaRecipient := utl.GetActualDiffBalanceInWaves(
		suite, address, initBalanceWavesGoRecipient, initBalanceWavesScalaRecipient)

	actualDiffBalanceAssetGoRecipient, actualDiffBalanceAssetScalaRecipient := utl.GetActualDiffBalanceInAssets(suite,
		address, testdata.Asset.ID, initBalanceAssetGoRecipient, initBalanceAssetScalaRecipient)
	return *utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo,
			tx.WtErr.ErrWtScala, tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		*utl.NewDiffBalances(actualDiffBalanceWavesGoSender, actualDiffBalanceWavesScalaSender,
			actualDiffBalanceAssetGoSender, actualDiffBalanceAssetScalaSender),
		*utl.NewDiffBalances(actualDiffBalanceWavesGoRecipient, actualDiffBalanceWavesScalaRecipient,
			actualDiffBalanceAssetGoRecipient, actualDiffBalanceAssetScalaRecipient)
}

func TransferSendWithTestData[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func TransferBroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendTransferTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.AccountDiffBalances, utl.AccountDiffBalances) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, TransferSendWithTestData[T])
}

func BroadcastTransferTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.AccountDiffBalances, utl.AccountDiffBalances) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, TransferBroadcastWithTestData[T])
}

func TransferFunds(suite *f.BaseSuite, version byte, scheme proto.Scheme, from, to int, amount uint64) utl.ConsideredTransaction {
	sender := utl.GetAccount(suite, from)
	recipient := utl.GetAccount(suite, to)
	tx := TransferSend(suite, version, scheme, sender.PublicKey, sender.SecretKey,
		proto.NewOptionalAssetWaves(), proto.NewOptionalAssetWaves(), utl.GetCurrentTimestampInMs(), amount, 100000,
		proto.NewRecipientFromAddress(recipient.Address), nil, true)
	return tx
}

func GetNewAccountWithFunds(suite *f.BaseSuite, version byte, scheme proto.Scheme, from int, amount uint64) int {
	accNumber, _ := utl.AddNewAccount(suite, scheme)
	tx := TransferFunds(suite, version, scheme, from, accNumber, amount)
	require.NoError(suite.T(), tx.WtErr.ErrWtGo, "Reached deadline of Transfer tx in Go")
	require.NoError(suite.T(), tx.WtErr.ErrWtScala, "Reached deadline of Transfer tx in Scala")
	//waiting for changing waves balance
	err := node_client.Retry(5*time.Second, func() error {
		var balanceErr error
		balanceGo, balanceScala := utl.GetAvailableBalanceInWaves(suite, utl.GetAccount(suite, accNumber).Address)
		if balanceScala == 0 && balanceGo == 0 {
			balanceErr = errors.New("account Waves balance is empty")
		}
		return balanceErr
	})
	require.NoError(suite.T(), err)
	return accNumber
}

// amount of Asset that transfered from one account to another, by default it will be all amount of Asset
func TransferAssetAmount(suite *f.BaseSuite, version byte, scheme proto.Scheme, assetId crypto.Digest,
	from, to int, assetAmount ...uint64) {
	var amount, currentAmount uint64
	currentAmount = uint64(utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, from).Address, assetId))
	if len(assetAmount) == 1 && assetAmount[0] <= currentAmount {
		amount = assetAmount[0]
	} else {
		amount = currentAmount
	}
	tx := TransferSend(suite, version, scheme, utl.GetAccount(suite, from).PublicKey, utl.GetAccount(suite, from).SecretKey,
		*proto.NewOptionalAssetFromDigest(assetId), proto.NewOptionalAssetWaves(), utl.GetCurrentTimestampInMs(), amount,
		100000, proto.NewRecipientFromAddress(utl.GetAccount(suite, to).Address), nil, true)
	require.NoError(suite.T(), tx.WtErr.ErrWtGo, "Reached deadline of Transfer tx in Go")
	require.NoError(suite.T(), tx.WtErr.ErrWtScala, "Reached deadline of Transfer tx in Scala")
}
