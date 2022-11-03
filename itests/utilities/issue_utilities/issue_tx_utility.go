package issue_utilities

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignIssueTransaction[T any](suite *f.BaseSuite, version byte, testdata testdata.IssueTestData[T]) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedIssueWithSig(testdata.Account.PublicKey, testdata.AssetName,
			testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	} else {
		tx = proto.NewUnsignedIssueWithProofs(version, testdata.ChainID, testdata.Account.PublicKey, testdata.AssetName,
			testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, nil, testdata.Timestamp, testdata.Fee)
	}
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func Issue[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, timeout time.Duration) (crypto.Digest, error, error) {
	tx := NewSignIssueTransaction(suite, version, testdata)
	errGo, errScala := utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	txID := utl.ExtractTxID(suite.T(), tx, testdata.ChainID)
	return txID, errGo, errScala
}

func IssueBroadcast[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, timeout time.Duration) (
	utl.BroadcastedTransaction, error, error) {
	tx := NewSignIssueTransaction(suite, version, testdata)
	brdCstTx, errGo, errScala := utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return brdCstTx, errGo, errScala
}
