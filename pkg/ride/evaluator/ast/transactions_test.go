package ast

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

var digest = crypto.MustDigestFromBase58("WmryL34P6UwwUphNbhjBRwiCWxX15Nf5D8T7AmQY7yx")
var optionalAsset = *proto.NewOptionalAssetFromDigest(crypto.MustDigestFromBase58("WmryL34P6UwwUphNbhjBRwiCWxX15Nf5D8T7AmQY7yx"))

type TransferV1TestSuite struct {
	suite.Suite
	tx *proto.TransferV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *TransferV1TestSuite) SetupTest() {
	a.tx = byte_helpers.TransferV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *TransferV1TestSuite) Test_feeAssetId_Presence() {
	a.tx.Transfer.FeeAsset = optionalAsset
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(digest.Bytes()), rs["feeAssetId"])
}

func (a *TransferV1TestSuite) Test_feeAssetId_Absence() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewUnit(), rs["feeAssetId"])
}

func (a *TransferV1TestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(10000), rs["amount"])
}

func (a *TransferV1TestSuite) Test_assetId_presence() {
	a.tx.Transfer.AmountAsset = optionalAsset
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(digest.Bytes()), rs["assetId"])
}

func (a *TransferV1TestSuite) Test_assetId_absence() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewUnit(), rs["assetId"])
}

func (a *TransferV1TestSuite) Test_recipient() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewRecipientFromProtoRecipient(a.tx.Recipient), rs["recipient"])
}

func (a *TransferV1TestSuite) Test_attachment() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.Attachment.Bytes()), rs["attachment"])
}

func (a *TransferV1TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *TransferV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *TransferV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *TransferV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *TransferV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *TransferV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *TransferV1TestSuite) Test_bodyBytes() {
	_, pub := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *TransferV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *TransferV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("TransferTransaction"), rs[InstanceFieldName])
}

func TestNewVariablesFromTransferV1(t *testing.T) {
	suite.Run(t, new(TransferV1TestSuite))
}

type TransferV2TestSuite struct {
	suite.Suite
	tx *proto.TransferV2
	f  func(scheme proto.Scheme, tx *proto.TransferV2) (map[string]Expr, error)
}

func (a *TransferV2TestSuite) SetupTest() {
	a.tx = byte_helpers.TransferV2.Transaction.Clone()
	a.f = newVariablesFromTransferV2
}

func (a *TransferV2TestSuite) Test_feeAssetId_Presence() {
	a.tx.Transfer.FeeAsset = optionalAsset
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(digest.Bytes()), rs["feeAssetId"])
}

func (a *TransferV2TestSuite) Test_feeAssetId_Absence() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewUnit(), rs["feeAssetId"])
}

func (a *TransferV2TestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["amount"])
}

func (a *TransferV2TestSuite) Test_assetId_presence() {
	a.tx.Transfer.AmountAsset = optionalAsset
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(digest.Bytes()), rs["assetId"])
}

func (a *TransferV2TestSuite) Test_assetId_absence() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewUnit(), rs["assetId"])
}

func (a *TransferV2TestSuite) Test_recipient() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewRecipientFromProtoRecipient(a.tx.Recipient), rs["recipient"])
}

func (a *TransferV2TestSuite) Test_attachment() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.Attachment.Bytes()), rs["attachment"])
}

func (a *TransferV2TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *TransferV2TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *TransferV2TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *TransferV2TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *TransferV2TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *TransferV2TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *TransferV2TestSuite) Test_bodyBytes() {
	_, pub := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0].Bytes())
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *TransferV2TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *TransferV2TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("TransferTransaction"), rs[InstanceFieldName])
}

func TestNewVariablesFromTransferV2(t *testing.T) {
	suite.Run(t, new(TransferV2TestSuite))
}

type GenesisTestSuite struct {
	suite.Suite
	tx *proto.Genesis
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *GenesisTestSuite) SetupTest() {
	a.tx = byte_helpers.Genesis.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *GenesisTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["amount"])
}

func (a *GenesisTestSuite) Test_recipient() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewRecipientFromProtoRecipient(proto.NewRecipientFromAddress(a.tx.Recipient)), rs["recipient"])
}

func (a *GenesisTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID()
	a.NoError(err)
	a.Equal(NewBytes(id), rs["id"])
}

func (a *GenesisTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(0), rs["fee"])
}

func (a *GenesisTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *GenesisTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func TestNewVariablesFromGenesis(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}
