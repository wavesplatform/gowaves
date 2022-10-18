package reissue_utilities

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignReissueTransaction[T any](suite *f.BaseSuite, testdata testdata.ReissueTestData[T]) *proto.ReissueWithSig {
	tx := proto.NewUnsignedReissueWithSig(testdata.Account.PublicKey, testdata.AssetID, testdata.Quantity, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func Reissue[T any](suite *f.BaseSuite, testdata testdata.ReissueTestData[T], timeout time.Duration) (*proto.ReissueWithSig, error, error) {
	tx := NewSignReissueTransaction(suite, testdata)
	errGo, errScala := utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return tx, errGo, errScala
}
