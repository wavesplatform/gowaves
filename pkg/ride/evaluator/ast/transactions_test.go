package ast

import (
	"testing"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

var _digest = crypto.MustDigestFromBase58("WmryL34P6UwwUphNbhjBRwiCWxX15Nf5D8T7AmQY7yx")
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
	a.Equal(NewBytes(_digest.Bytes()), rs["feeAssetId"])
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
	a.Equal(NewBytes(_digest.Bytes()), rs["assetId"])
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
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
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
	a.Equal(NewBytes(_digest.Bytes()), rs["feeAssetId"])
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
	a.Equal(NewBytes(_digest.Bytes()), rs["assetId"])
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
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
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
	tx := &proto.Genesis{}
	_ = copier.Copy(tx, byte_helpers.Genesis.Transaction)
	a.tx = tx
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

type PaymentTestSuite struct {
	suite.Suite
	tx *proto.Payment
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *PaymentTestSuite) SetupTest() {
	tx := &proto.Payment{}
	_ = copier.Copy(tx, byte_helpers.Payment.Transaction)
	a.tx = tx
	a.f = NewVariablesFromTransaction
}

func (a *PaymentTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["amount"])
}

func (a *PaymentTestSuite) Test_recipient() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewRecipientFromProtoRecipient(proto.NewRecipientFromAddress(a.tx.Recipient)), rs["recipient"])
}

func (a *PaymentTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID()
	a.NoError(err)
	a.Equal(NewBytes(id), rs["id"])
}

func (a *PaymentTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *PaymentTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *PaymentTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *PaymentTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *PaymentTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *PaymentTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *PaymentTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *PaymentTestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("PaymentTransaction"), rs[InstanceFieldName])
}

func TestNewVariablesFromPayment(t *testing.T) {
	suite.Run(t, new(PaymentTestSuite))
}

type ReissueV1TestSuite struct {
	suite.Suite
	tx *proto.ReissueV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *ReissueV1TestSuite) SetupTest() {
	a.tx = byte_helpers.ReissueV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *ReissueV1TestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["quantity"])
}

func (a *ReissueV1TestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *ReissueV1TestSuite) Test_reissuable() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBoolean(a.tx.Reissuable), rs["reissuable"])
}

func (a *ReissueV1TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *ReissueV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *ReissueV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *ReissueV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *ReissueV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *ReissueV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *ReissueV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *ReissueV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *ReissueV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("ReissueTransaction"), rs[InstanceFieldName])
}

//ReissueTransaction
func TestNewVariablesFromReissueV1(t *testing.T) {
	suite.Run(t, new(ReissueV1TestSuite))
}

type ReissueV2TestSuite struct {
	suite.Suite
	tx *proto.ReissueV2
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *ReissueV2TestSuite) SetupTest() {
	a.tx = byte_helpers.ReissueV2.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *ReissueV2TestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["quantity"])
}

func (a *ReissueV2TestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *ReissueV2TestSuite) Test_reissuable() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBoolean(a.tx.Reissuable), rs["reissuable"])
}

func (a *ReissueV2TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *ReissueV2TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *ReissueV2TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *ReissueV2TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *ReissueV2TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *ReissueV2TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *ReissueV2TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *ReissueV2TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *ReissueV2TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("ReissueTransaction"), rs[InstanceFieldName])
}

//ReissueTransaction
func TestNewVariablesFromReissueV2(t *testing.T) {
	suite.Run(t, new(ReissueV2TestSuite))
}

type BurnV1TestSuite struct {
	suite.Suite
	tx *proto.BurnV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *BurnV1TestSuite) SetupTest() {
	a.tx = byte_helpers.BurnV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *BurnV1TestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["quantity"])
}

func (a *BurnV1TestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *BurnV1TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *BurnV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *BurnV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *BurnV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1), rs["version"])
}

func (a *BurnV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *BurnV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *BurnV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *BurnV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *BurnV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("BurnTransaction"), rs[InstanceFieldName])
}

//BurnV1
func TestNewVariablesFromBurnV1(t *testing.T) {
	suite.Run(t, new(BurnV1TestSuite))
}

type BurnV2TestSuite struct {
	suite.Suite
	tx *proto.BurnV2
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *BurnV2TestSuite) SetupTest() {
	a.tx = byte_helpers.BurnV2.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *BurnV2TestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["quantity"])
}

func (a *BurnV2TestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *BurnV2TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *BurnV2TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *BurnV2TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *BurnV2TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(2), rs["version"])
}

func (a *BurnV2TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *BurnV2TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *BurnV2TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *BurnV2TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *BurnV2TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("BurnTransaction"), rs[InstanceFieldName])
}

//BurnV2
func TestNewVariablesFromBurnV2(t *testing.T) {
	suite.Run(t, new(BurnV2TestSuite))
}

type MassTransferV1TestSuite struct {
	suite.Suite
	tx *proto.MassTransferV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *MassTransferV1TestSuite) SetupTest() {
	a.tx = byte_helpers.MassTransferV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *MassTransferV1TestSuite) Test_assetId_presence() {
	a.tx.Asset = optionalAsset
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(_digest.Bytes()), rs["assetId"])
}

func (a *MassTransferV1TestSuite) Test_assetId_absence() {
	a.tx.Asset = proto.OptionalAsset{}
	rs, err := a.f(proto.MainNetScheme, a.tx)

	a.NoError(err)
	a.Equal(NewUnit(), rs["assetId"])
}

func (a *MassTransferV1TestSuite) Test_totalAmount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["totalAmount"])
}

func (a *MassTransferV1TestSuite) Test_transfers() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)

	m := make(map[string]Expr)
	m["recipient"] = NewRecipientFromProtoRecipient(a.tx.Transfers[0].Recipient)
	m["amount"] = NewLong(int64(a.tx.Transfers[0].Amount))
	obj := NewObject(m)
	a.Equal(Exprs{obj}, rs["transfers"])
}

func (a *MassTransferV1TestSuite) Test_transferCount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1), rs["transferCount"])
}

func (a *MassTransferV1TestSuite) Test_attachment() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.Attachment.Bytes()), rs["attachment"])
}

func (a *MassTransferV1TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *MassTransferV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *MassTransferV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *MassTransferV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1), rs["version"])
}

func (a *MassTransferV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *MassTransferV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *MassTransferV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *MassTransferV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *MassTransferV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("MassTransferTransaction"), rs[InstanceFieldName])
}

//MassTransferTransaction
func TestNewVariablesFromMassTransferV1(t *testing.T) {
	suite.Run(t, new(MassTransferV1TestSuite))
}

type ExchangeV1TestSuite struct {
	suite.Suite
	tx *proto.ExchangeV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *ExchangeV1TestSuite) SetupTest() {
	a.tx = byte_helpers.ExchangeV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *ExchangeV1TestSuite) Test_buyOrder() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs["buyOrder"].InstanceOf())
}

func (a *ExchangeV1TestSuite) Test_sellOrder() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs["sellOrder"].InstanceOf())
}

func (a *ExchangeV1TestSuite) Test_price() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["price"])
}

func (a *ExchangeV1TestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["amount"])
}

func (a *ExchangeV1TestSuite) Test_buyMatcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(10000), rs["buyMatcherFee"])
}

func (a *ExchangeV1TestSuite) Test_sellMatcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(10000), rs["sellMatcherFee"])
}

func (a *ExchangeV1TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *ExchangeV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *ExchangeV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *ExchangeV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1), rs["version"])
}

func (a *ExchangeV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}
func (a *ExchangeV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *ExchangeV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *ExchangeV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *ExchangeV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("ExchangeTransaction"), rs[InstanceFieldName])
}

//ExchangeV1
func TestNewVariablesFromExchangeV1(t *testing.T) {
	suite.Run(t, new(ExchangeV1TestSuite))
}

type ExchangeV2TestSuite struct {
	suite.Suite
	tx *proto.ExchangeV2
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *ExchangeV2TestSuite) SetupTest() {
	a.tx = byte_helpers.ExchangeV2.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *ExchangeV2TestSuite) Test_price() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["price"])
}

func (a *ExchangeV2TestSuite) Test_buyOrder() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs["buyOrder"].InstanceOf())
}

func (a *ExchangeV2TestSuite) Test_sellOrder() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs["sellOrder"].InstanceOf())
}

func (a *ExchangeV2TestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["amount"])
}

func (a *ExchangeV2TestSuite) Test_buyMatcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(10000), rs["buyMatcherFee"])
}

func (a *ExchangeV2TestSuite) Test_sellMatcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(10000), rs["sellMatcherFee"])
}

func (a *ExchangeV2TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *ExchangeV2TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *ExchangeV2TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *ExchangeV2TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(2), rs["version"])
}

func (a *ExchangeV2TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *ExchangeV2TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *ExchangeV2TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *ExchangeV2TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *ExchangeV2TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("ExchangeTransaction"), rs[InstanceFieldName])
}

//ExchangeV2
func TestNewVariablesFromExchangeV2(t *testing.T) {
	suite.Run(t, new(ExchangeV2TestSuite))
}

type OrderTestSuite struct {
	suite.Suite
	tx proto.Order
	f  func(scheme proto.Scheme, tx proto.Order) (map[string]Expr, error)
	d  crypto.Digest
}

func (a *OrderTestSuite) SetupTest() {
	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))
	a.d, _ = crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")

	_, matcherPk, _ := crypto.GenerateKeyPair([]byte("test1"))

	sellOrder := proto.NewUnsignedOrderV1(
		pk,
		matcherPk,
		*proto.NewOptionalAssetFromDigest(a.d),
		*proto.NewOptionalAssetFromDigest(a.d),
		proto.Sell,
		100000,
		10000,
		proto.Timestamp(1544715621),
		proto.Timestamp(1544715621),
		10000)

	a.NoError(sellOrder.Sign(sk))

	a.tx = sellOrder
	a.f = newVariablesFromOrder
}

func (a *OrderTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID()
	a.NoError(err)
	a.Equal(NewBytes(id), rs["id"])
}

func (a *OrderTestSuite) Test_matcherPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	tmp := a.tx.GetMatcherPK()
	a.Equal(NewBytes(tmp.Bytes()), rs["matcherPublicKey"])
}

func (a *OrderTestSuite) Test_assetPair() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewAssetPair(NewBytes(a.d.Bytes()), NewBytes(a.d.Bytes())), rs["assetPair"])
}

func (a *OrderTestSuite) Test_orderType() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Sell", rs["orderType"].InstanceOf())
}

func (a *OrderTestSuite) Test_price() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["price"])
}

func (a *OrderTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(10000), rs["amount"])
}

func (a *OrderTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(byte_helpers.TIMESTAMP)), rs["timestamp"])
}

func (a *OrderTestSuite) Test_expiration() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(byte_helpers.TIMESTAMP)), rs["expiration"])
}

func (a *OrderTestSuite) Test_matcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(10000), rs["matcherFee"])
}

func (a *OrderTestSuite) Test_matcherFeeAssetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewUnit(), rs["matcherFeeAssetId"])
}

func (a *OrderTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.GetSenderPK())
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *OrderTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	pk := a.tx.GetSenderPK()
	a.Equal(NewBytes(pk.Bytes()), rs["senderPublicKey"])
}

func (a *OrderTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	proofs, _ := a.tx.GetProofs()
	sig, _ := crypto.NewSignatureFromBytes(proofs.Proofs[0])
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *OrderTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	p, _ := a.tx.GetProofs()
	a.Equal(Exprs{NewBytes(p.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *OrderTestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", NewObject(rs).InstanceOf())
}

//Order
func TestNewVariablesFromOrderV1(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}

type SetAssetScriptV1TestSuite struct {
	suite.Suite
	tx *proto.SetAssetScriptV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *SetAssetScriptV1TestSuite) SetupTest() {
	a.tx = byte_helpers.SetAssetScriptV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *SetAssetScriptV1TestSuite) Test_script() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes([]byte("hello")), rs["script"])
}

func (a *SetAssetScriptV1TestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *SetAssetScriptV1TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID()
	a.NoError(err)
	a.Equal(NewBytes(id), rs["id"])
}

func (a *SetAssetScriptV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *SetAssetScriptV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *SetAssetScriptV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1), rs["version"])
}

func (a *SetAssetScriptV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *SetAssetScriptV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *SetAssetScriptV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *SetAssetScriptV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *SetAssetScriptV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("SetAssetScriptTransaction", NewObject(rs).InstanceOf())
}

//SetAssetScriptV1
func TestNewVariablesFromSetAssetScriptV1(t *testing.T) {
	suite.Run(t, new(SetAssetScriptV1TestSuite))
}

type InvokeScriptV1TestSuite struct {
	suite.Suite
	tx *proto.InvokeScriptV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *InvokeScriptV1TestSuite) SetupTest() {
	a.tx = byte_helpers.InvokeScriptV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *InvokeScriptV1TestSuite) Test_dappAddress() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewRecipientFromProtoRecipient(a.tx.ScriptRecipient), rs["dappAddress"])
}

func (a *InvokeScriptV1TestSuite) Test_payment_presence() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	payment := rs["payment"].(Getable)
	asset, _ := payment.Get("assetId")
	a.Equal(NewBytes(byte_helpers.Digest.Bytes()), asset)

	amount, _ := payment.Get("amount")
	a.Equal(NewLong(100000), amount)
}

func (a *InvokeScriptV1TestSuite) Test_payment_absence() {
	a.tx.Payments = nil
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewUnit(), rs["payment"])
}

func (a *InvokeScriptV1TestSuite) Test_feeAssetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(byte_helpers.Digest.Bytes()), rs["feeAssetId"])
}

func (a *InvokeScriptV1TestSuite) Test_function() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("funcname"), rs["function"])
}

func (a *InvokeScriptV1TestSuite) Test_args() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Params(NewString("StringArgument")), rs["args"])
}

func (a *InvokeScriptV1TestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(NewBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *InvokeScriptV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *InvokeScriptV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *InvokeScriptV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1), rs["version"])
}

func (a *InvokeScriptV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *InvokeScriptV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *InvokeScriptV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *InvokeScriptV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *InvokeScriptV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("InvokeScriptTransaction", NewObject(rs).InstanceOf())
}

//InvokeScriptTransaction
func TestNewVariablesFromInvokeScriptV1(t *testing.T) {
	suite.Run(t, new(InvokeScriptV1TestSuite))
}

type IssueV1TestSuite struct {
	suite.Suite
	tx *proto.IssueV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *IssueV1TestSuite) SetupTest() {
	a.tx = byte_helpers.IssueV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *IssueV1TestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1000), rs["quantity"])
}

func (a *IssueV1TestSuite) Test_name() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("name"), rs["name"])
}

func (a *IssueV1TestSuite) Test_description() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("description"), rs["description"])
}

func (a *IssueV1TestSuite) Test_reissuable() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBoolean(a.tx.Reissuable), rs["reissuable"])
}

func (a *IssueV1TestSuite) Test_decimals() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(4), rs["decimals"])
}

func (a *IssueV1TestSuite) Test_script() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewUnit(), rs["script"])
}

func (a *IssueV1TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *IssueV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *IssueV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *IssueV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *IssueV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *IssueV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *IssueV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *IssueV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *IssueV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("IssueTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromIssueV1(t *testing.T) {
	suite.Run(t, new(IssueV1TestSuite))
}

type IssueV2TestSuite struct {
	suite.Suite
	tx *proto.IssueV2
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *IssueV2TestSuite) SetupTest() {
	a.tx = byte_helpers.IssueV2.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *IssueV2TestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1000), rs["quantity"])
}

func (a *IssueV2TestSuite) Test_name() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("name"), rs["name"])
}

func (a *IssueV2TestSuite) Test_description() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString("description"), rs["description"])
}

func (a *IssueV2TestSuite) Test_reissuable() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBoolean(a.tx.Reissuable), rs["reissuable"])
}

func (a *IssueV2TestSuite) Test_decimals() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(4), rs["decimals"])
}

func (a *IssueV2TestSuite) Test_script() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes([]byte("script")), rs["script"])
}

func (a *IssueV2TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *IssueV2TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *IssueV2TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *IssueV2TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *IssueV2TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *IssueV2TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *IssueV2TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *IssueV2TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *IssueV2TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("IssueTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromIssueV2(t *testing.T) {
	suite.Run(t, new(IssueV2TestSuite))
}

//
type LeaseV1TestSuite struct {
	suite.Suite
	tx *proto.LeaseV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *LeaseV1TestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *LeaseV1TestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["amount"])
}

func (a *LeaseV1TestSuite) Test_recipient() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewRecipientFromProtoRecipient(a.tx.Recipient), rs["recipient"])
}

func (a *LeaseV1TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *LeaseV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *LeaseV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *LeaseV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *LeaseV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *LeaseV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *LeaseV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *LeaseV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *LeaseV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("LeaseTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromLeaseV1(t *testing.T) {
	suite.Run(t, new(LeaseV1TestSuite))
}

//
type LeaseV2TestSuite struct {
	suite.Suite
	tx *proto.LeaseV2
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *LeaseV2TestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseV2.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *LeaseV2TestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(100000), rs["amount"])
}

func (a *LeaseV2TestSuite) Test_recipient() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewRecipientFromProtoRecipient(a.tx.Recipient), rs["recipient"])
}

func (a *LeaseV2TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *LeaseV2TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *LeaseV2TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *LeaseV2TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *LeaseV2TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *LeaseV2TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *LeaseV2TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *LeaseV2TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *LeaseV2TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("LeaseTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromLeaseV2(t *testing.T) {
	suite.Run(t, new(LeaseV2TestSuite))
}

//
type LeaseCancelV1TestSuite struct {
	suite.Suite
	tx *proto.LeaseCancelV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *LeaseCancelV1TestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseCancelV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *LeaseCancelV1TestSuite) Test_leaseId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(_digest.Bytes()), rs["leaseId"])
}

func (a *LeaseCancelV1TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *LeaseCancelV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *LeaseCancelV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *LeaseCancelV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *LeaseCancelV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *LeaseCancelV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *LeaseCancelV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *LeaseCancelV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *LeaseCancelV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("LeaseCancelTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromLeaseCancelV1(t *testing.T) {
	suite.Run(t, new(LeaseCancelV1TestSuite))
}

//
type LeaseCancelV2TestSuite struct {
	suite.Suite
	tx *proto.LeaseCancelV2
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *LeaseCancelV2TestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseCancelV2.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *LeaseCancelV2TestSuite) Test_leaseId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.LeaseID.Bytes()), rs["leaseId"])
}

func (a *LeaseCancelV2TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *LeaseCancelV2TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *LeaseCancelV2TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *LeaseCancelV2TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *LeaseCancelV2TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *LeaseCancelV2TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *LeaseCancelV2TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *LeaseCancelV2TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *LeaseCancelV2TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("LeaseCancelTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromLeaseCancelV2(t *testing.T) {
	suite.Run(t, new(LeaseCancelV2TestSuite))
}

//
type DataV1TestSuite struct {
	suite.Suite
	tx *proto.DataV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *DataV1TestSuite) SetupTest() {
	a.tx = byte_helpers.DataV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *DataV1TestSuite) Test_data() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes([]byte("hello")), rs["data"].(*DataEntryListExpr).GetByIndex(0, proto.DataBinary))
}

func (a *DataV1TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *DataV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *DataV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *DataV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *DataV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *DataV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *DataV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *DataV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *DataV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("DataTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromDataV1TestSuite(t *testing.T) {
	suite.Run(t, new(DataV1TestSuite))
}

//
type SponsorshipV1TestSuite struct {
	suite.Suite
	tx *proto.SponsorshipV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *SponsorshipV1TestSuite) SetupTest() {
	a.tx = byte_helpers.SponsorshipV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *SponsorshipV1TestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(_digest.Bytes()), rs["assetId"])
}

func (a *SponsorshipV1TestSuite) Test_minSponsoredAssetFee_presence() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(1000), rs["minSponsoredAssetFee"])
}

func (a *SponsorshipV1TestSuite) Test_minSponsoredAssetFee_absence() {
	a.tx.MinAssetFee = 0
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewUnit(), rs["minSponsoredAssetFee"])
}

func (a *SponsorshipV1TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *SponsorshipV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *SponsorshipV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *SponsorshipV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *SponsorshipV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *SponsorshipV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *SponsorshipV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *SponsorshipV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *SponsorshipV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("SponsorFeeTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromSponsorshipV1(t *testing.T) {
	suite.Run(t, new(SponsorshipV1TestSuite))
}

//
type CreateAliasV1TestSuite struct {
	suite.Suite
	tx *proto.CreateAliasV1
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *CreateAliasV1TestSuite) SetupTest() {
	a.tx = byte_helpers.CreateAliasV1.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *CreateAliasV1TestSuite) Test_alias() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString(a.tx.Alias.String()), rs["alias"])
}

func (a *CreateAliasV1TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *CreateAliasV1TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *CreateAliasV1TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *CreateAliasV1TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *CreateAliasV1TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *CreateAliasV1TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *CreateAliasV1TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *CreateAliasV1TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Signature.Bytes())}, rs["proofs"])
}

func (a *CreateAliasV1TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("CreateAliasTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromCreateAliasV1TestSuite(t *testing.T) {
	suite.Run(t, new(CreateAliasV1TestSuite))
}

//
type CreateAliasV2TestSuite struct {
	suite.Suite
	tx *proto.CreateAliasV2
	f  func(scheme proto.Scheme, tx proto.Transaction) (map[string]Expr, error)
}

func (a *CreateAliasV2TestSuite) SetupTest() {
	a.tx = byte_helpers.CreateAliasV2.Transaction.Clone()
	a.f = NewVariablesFromTransaction
}

func (a *CreateAliasV2TestSuite) Test_alias() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewString(a.tx.Alias.String()), rs["alias"])
}

func (a *CreateAliasV2TestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID()
	a.Equal(NewBytes(id), rs["id"])
}

func (a *CreateAliasV2TestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Fee)), rs["fee"])
}

func (a *CreateAliasV2TestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *CreateAliasV2TestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewLong(int64(a.tx.Version)), rs["version"])
}

func (a *CreateAliasV2TestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(NewAddressFromProtoAddress(addr), rs["sender"])
}

func (a *CreateAliasV2TestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(NewBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *CreateAliasV2TestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(&BytesExpr{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(*BytesExpr).Value))
}

func (a *CreateAliasV2TestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(Exprs{NewBytes(a.tx.Proofs.Proofs[0].Bytes())}, rs["proofs"])
}

func (a *CreateAliasV2TestSuite) Test_InstanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("CreateAliasTransaction", NewObject(rs).InstanceOf())
}

func TestNewVariablesFromCreateAliasV2TestSuite(t *testing.T) {
	suite.Run(t, new(CreateAliasV2TestSuite))
}
