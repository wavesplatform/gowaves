package issue_utilities

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	"github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignIssueTransaction[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T]) *proto.IssueWithSig {
	tx := proto.NewUnsignedIssueWithSig(testdata.Account.PublicKey, testdata.AssetName,
		testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func Issue[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], timeout time.Duration) (*proto.IssueWithSig, error, error) {
	tx := NewSignIssueTransaction(suite, testdata)
	errGo, errScala := utilities.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return tx, errGo, errScala
}

func IssueBroadcast[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], timeout time.Duration) (
	utilities.BroadcastedTransaction, error, error) {
	tx := NewSignIssueTransaction(suite, testdata)
	brdCstTx, errGo, errScala := utilities.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return brdCstTx, errGo, errScala
}
