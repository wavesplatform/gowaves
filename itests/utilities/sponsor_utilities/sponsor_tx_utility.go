package sponsor_utilities

import (
	"github.com/stretchr/testify/require"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func NewSignSponsorshipTransaction(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, minAssetFee, fee, timestamp uint64) proto.Transaction {
	var tx proto.Transaction
	tx = proto.NewUnsignedSponsorshipWithProofs(version, senderPK, assetID, minAssetFee, fee, timestamp)
	err := tx.Sign(scheme, senderSK)
	txJson := utl.GetTransactionJsonOrErrMsg(tx)
	suite.T().Logf("Sponsorship Transaction JSON after sign: %s", txJson)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func SponsorshipSend(suite *f.BaseSuite, version byte, scheme proto.Scheme, senderPK crypto.PublicKey,
	senderSK crypto.SecretKey, assetID crypto.Digest, minAssetFee, fee, timestamp uint64,
	waitForTx bool) utl.ConsideredTransaction {
	tx := NewSignSponsorshipTransaction(suite, version, scheme, senderPK, senderSK, assetID, minAssetFee, fee, timestamp)
	return utl.SendAndWaitTransaction(suite, tx, scheme, waitForTx)
}

//func NewSignSponsorshipTransactionWithTestData
