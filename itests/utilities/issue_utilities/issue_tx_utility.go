package issue_utilities

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignIssueTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, name, description string, quantity, timestamp, fee uint64, decimals byte,
	reissuable bool, script proto.Script) proto.Transaction {
	var tx proto.Transaction
	if version == 1 {
		tx = proto.NewUnsignedIssueWithSig(senderPK, name, description, quantity, decimals, reissuable, timestamp, fee)
	} else {
		tx = proto.NewUnsignedIssueWithProofs(version, senderPK, name, description, quantity, decimals,
			reissuable, script, timestamp, fee)
	}
	err := tx.Sign(scheme, senderSK)
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Issue Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func IssueSend(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, name, description string, quantity, timestamp, fee uint64, decimals byte,
	reissuable, waitForTx bool, script proto.Script) utl.ConsideredTransaction {
	tx := NewSignIssueTransaction(suite, version, scheme, senderPK, senderSK, name, description, quantity,
		timestamp, fee, decimals, reissuable, script)
	return utl.SendAndWaitTransaction(suite, tx, scheme, waitForTx)
}

type MakeTx[T any] func(suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte, waitForTx bool) utl.ConsideredTransaction

// MakeTxAndGetDiffBalances This function returns txID with init balance before tx and difference balance after tx for both nodes
func MakeTxAndGetDiffBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool, makeTx MakeTx[T]) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	initBalanceGo, initBalanceScala := utl.GetAvailableBalanceInWaves(suite, testdata.Account.Address)
	tx := makeTx(suite, testdata, version, waitForTx)
	actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
		suite, testdata.Account.Address, initBalanceGo, initBalanceScala)
	actualDiffBalanceInAssetGo, actualDiffBalanceInAssetScala := utl.GetActualDiffBalanceInAssets(suite,
		testdata.Account.Address, tx.TxID, 0, 0)
	return utl.NewConsideredTransaction(tx.TxID, tx.Resp.ResponseGo, tx.Resp.ResponseScala, tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
			tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala),
		utl.NewBalanceInWaves(actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala),
		utl.NewBalanceInAsset(actualDiffBalanceInAssetGo, actualDiffBalanceInAssetScala)
}

func NewSignIssueTransactionWithTestData[T any](suite *f.BaseSuite, version byte, testdata testdata.IssueTestData[T]) proto.Transaction {
	return NewSignIssueTransaction(suite, version, testdata.ChainID, testdata.Account.PublicKey, testdata.Account.SecretKey,
		testdata.AssetName, testdata.AssetDesc, testdata.Quantity, testdata.Timestamp, testdata.Fee, testdata.Decimals,
		testdata.Reissuable, testdata.Script)
}

func IssueSendWithTestData[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignIssueTransactionWithTestData(suite, version, testdata)
	return utl.SendAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func IssueBroadcastWithTestData[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignIssueTransactionWithTestData(suite, version, testdata)
	return utl.BroadcastAndWaitTransaction(suite, tx, testdata.ChainID, waitForTx)
}

func SendIssueTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, IssueSendWithTestData[T])
}

func BroadcastIssueTxAndGetBalances[T any](suite *f.BaseSuite, testdata testdata.IssueTestData[T], version byte,
	waitForTx bool) (utl.ConsideredTransaction, utl.BalanceInWaves, utl.BalanceInAsset) {
	return MakeTxAndGetDiffBalances(suite, testdata, version, waitForTx, IssueBroadcastWithTestData[T])
}

func IssueAssetAmount(suite *f.BaseSuite, version byte, scheme proto.Scheme, accountNumber int,
	assetAmount ...uint64) crypto.Digest {
	var amount uint64
	if len(assetAmount) == 1 {
		amount = assetAmount[0]
	} else {
		amount = 1
	}
	tx := IssueSend(suite, version, scheme, utl.GetAccount(suite, accountNumber).PublicKey,
		utl.GetAccount(suite, accountNumber).SecretKey, "Asset", "Common Asset for testing", amount,
		utl.GetCurrentTimestampInMs(), utl.MinIssueFeeWaves, 8, true, true, nil)
	return tx.TxID
}

func GetVersions(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.IssueTransaction, testdata.IssueMinVersion, testdata.IssueMaxVersion).Sum
}

func GetVersionsSmartAsset(suite *f.BaseSuite) []byte {
	return utl.GetAvailableVersions(suite.T(), proto.IssueTransaction, testdata.IssueSmartAssetMinVersion, testdata.IssueMaxVersion).Sum
}
