package transfer_utilities

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignTransferTransaction[T any](suite *f.BaseSuite, version byte, testdata testdata.TransferTestData[T]) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedTransferWithSig(testdata.Sender.PublicKey, testdata.Asset, testdata.FeeAsset,
			testdata.Timestamp, testdata.Amount, testdata.Fee, testdata.Recipient, testdata.Attachment)
	} else {
		tx = proto.NewUnsignedTransferWithProofs(version, testdata.Sender.PublicKey, testdata.Asset, testdata.FeeAsset,
			testdata.Timestamp, testdata.Amount, testdata.Fee, testdata.Recipient, testdata.Attachment)
	}
	err := tx.Sign(testdata.ChainID, testdata.Sender.SecretKey)
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Transfer Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func TransferSend[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte, timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransaction(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout, positive)
}

func TransferBroadcast[T any](suite *f.BaseSuite, testdata testdata.TransferTestData[T], version byte, timeout time.Duration, positive bool) utl.ConsideredTransaction {
	tx := NewSignTransferTransaction(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout, positive)
}
