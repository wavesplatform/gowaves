package alias_utilities

import (
	"time"

	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignAliasTransaction[T any](suite *f.BaseSuite, version byte, testdata testdata.AliasTestData[T]) proto.Transaction {
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
	suite.T().Logf("Alias Transaction JSON: %s", utl.GetTransactionJsonOrErrMsg(tx))
	require.NoError(suite.T(), err)
	return tx
}

func Alias[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration) utl.ConsideredTransaction {
	tx := NewSignAliasTransaction(suite, version, testdata)
	cnsdrTx := utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return cnsdrTx
}

func AliasBroadcast[T any](suite *f.BaseSuite, testdata testdata.AliasTestData[T], version byte, timeout time.Duration) (
	utl.BroadcastedTransaction, error, error) {
	tx := NewSignAliasTransaction(suite, version, testdata)
	brdCstTx, errGo, errScala := utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, timeout)
	return brdCstTx, errGo, errScala
}
