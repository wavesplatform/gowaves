package transfer_utilities

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
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

func TransferBroadcast(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	sk crypto.SecretKey, amountAsset, feeAsset proto.OptionalAsset, timestamp, amount, fee uint64,
	recipient proto.Recipient, attachment proto.Attachment, waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransaction(suite, version, scheme, senderPK, sk, amountAsset, feeAsset, timestamp, amount,
		fee, recipient, attachment)
	return utl.BroadcastAndWaitTransaction(suite, tx, scheme, waitForTx)
}

func NewSignTransferTransactionWithTestData[T any](suite *f.BaseSuite, version byte,
	testdata testdata.TransferTestData[T]) proto.Transaction {
	return NewSignTransferTransaction(suite, version, testdata.ChainID, testdata.Sender.PublicKey,
		testdata.Sender.SecretKey, testdata.Asset, testdata.FeeAsset, testdata.Timestamp, testdata.Amount,
		testdata.Fee, testdata.Recipient, testdata.Attachment)
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
	return accNumber
}
