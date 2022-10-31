package alias_utilities

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type expectedData interface {
	Positive() bool
}

func NewSignAliasTransaction[T expectedData](suite *f.BaseSuite, version byte, testdata testdata.AliasTestData[T]) proto.Transaction {
	var tx proto.Transaction
	alias := proto.NewAlias(testdata.ChainID, testdata.Alias)
	if version == 1 {
		tx = proto.NewUnsignedCreateAliasWithSig(
			testdata.Account.PublicKey, *alias,
			testdata.Fee, testdata.Timestamp)
	} else {
		tx = proto.NewUnsignedCreateAliasWithProofs(version, testdata.Account.PublicKey, *alias,
			testdata.Fee, testdata.Timestamp)
	}
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	if testdata.Expected.Positive() {
		suite.T().Logf("Alias Transaction JSON: %s", utl.GetTransactionJson(suite, tx))
	}
	require.NoError(suite.T(), err)
	return tx
}

func Alias[T expectedData](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration) (
	crypto.Digest, error, error) {
	tx := NewSignAliasTransaction(suite, version, testdata)
	errGo, errScala := utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return utl.ExtractTxID(suite.T(), tx, testdata.ChainID), errGo, errScala
}

func AliasBroadcast[T expectedData](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration) (
	utl.BroadcastedTransaction, error, error) {
	tx := NewSignAliasTransaction(suite, version, testdata)
	brdCstTx, errGo, errScala := utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return brdCstTx, errGo, errScala
}
