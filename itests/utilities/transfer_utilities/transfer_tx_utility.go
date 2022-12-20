package transfer_utilities

import (
	"fmt"
	"time"

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
	recipient proto.Recipient, attachment proto.Attachment, timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransaction(suite, version, scheme, senderPK, sk, amountAsset, feeAsset, timestamp, amount,
		fee, recipient, attachment)
	return utl.SendAndWaitTransaction(suite, tx, scheme, timeout, positive)
}

func TransferBroadcast(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	sk crypto.SecretKey, amountAsset, feeAsset proto.OptionalAsset, timestamp, amount, fee uint64,
	recipient proto.Recipient, attachment proto.Attachment, timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransaction(suite, version, scheme, senderPK, sk, amountAsset, feeAsset, timestamp, amount,
		fee, recipient, attachment)
	return utl.BroadcastAndWaitTransaction(suite, tx, scheme, timeout, positive)
}

func NewSignTransferTransactionWithTestData[T any](suite *f.BaseSuite, version byte,
	testdata testdata.TransferTestData[T]) proto.Transaction {
	return NewSignTransferTransaction(suite, version, testdata.ChainID, testdata.Sender.PublicKey,
		testdata.Sender.SecretKey, testdata.Asset, testdata.FeeAsset, testdata.Timestamp, testdata.Amount,
		testdata.Fee, testdata.Recipient, testdata.Attachment)
}

func TransferSendWithTestData[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout, positive)
}

func TransferBroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte,
	timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout, positive)
}

func TransferFunds(suite *f.BaseSuite, version byte, scheme proto.Scheme, from, to int, amount uint64,
	timeout time.Duration) utl.ConsideredTransaction {
	sender := utl.GetAccount(suite, from)
	recipient := utl.GetAccount(suite, to)
	tx := TransferSend(suite, version, scheme, sender.PublicKey, sender.SecretKey,
		proto.NewOptionalAssetWaves(), proto.NewOptionalAssetWaves(), utl.GetCurrentTimestampInMs(), amount, 100000,
		proto.NewRecipientFromAddress(recipient.Address), nil, timeout, true)
	return tx
}

func GetNewAccountWithFunds(suite *f.BaseSuite, version byte, scheme proto.Scheme, from int, amount uint64,
	timeout time.Duration) int {
	accNumber, _ := utl.AddNewAccount(suite, scheme)
	tx := TransferFunds(suite, version, scheme, from, accNumber, amount, timeout)
	require.NoError(suite.T(), tx.WtErr.ErrWtGo, "Reached deadline of Transfer tx in Go")
	require.NoError(suite.T(), tx.WtErr.ErrWtScala, "Reached deadline of Transfer tx in Scala")
	//waiting for changing waves balance
	err := node_client.Retry(timeout, func() error {
		var balanceErr error
		balanceGo, balanceScala := utl.GetAvailableBalanceInWaves(suite, utl.GetAccount(suite, accNumber).Address)
		if balanceScala == 0 && balanceGo == 0 {
			balanceErr = fmt.Errorf("account Waves balance is empty")
		}
		return balanceErr
	})
	require.NoError(suite.T(), err)
	return accNumber
}
