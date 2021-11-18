package ride

import (
	"testing"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

var (
	_digest = crypto.MustDigestFromBase58("WmryL34P6UwwUphNbhjBRwiCWxX15Nf5D8T7AmQY7yx")
	_asset  = *proto.NewOptionalAssetFromDigest(crypto.MustDigestFromBase58("WmryL34P6UwwUphNbhjBRwiCWxX15Nf5D8T7AmQY7yx"))
	_empty  = rideBytes(nil)
)

type TransferWithSigTestSuite struct {
	suite.Suite
	tx *proto.TransferWithSig
	f  func(scheme byte, tx proto.Transaction) (rideObject, error)
}

func (a *TransferWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.TransferWithSig.Transaction.Clone()
	a.f = transactionToObject
}

func (a *TransferWithSigTestSuite) Test_feeAssetId_Presence() {
	a.tx.Transfer.FeeAsset = _asset
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(_digest.Bytes()), rs["feeAssetId"])
}

func (a *TransferWithSigTestSuite) Test_feeAssetId_Absence() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideUnit{}, rs["feeAssetId"])
}

func (a *TransferWithSigTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(10000), rs["amount"])
}

func (a *TransferWithSigTestSuite) Test_assetId_presence() {
	a.tx.Transfer.AmountAsset = _asset
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), rs["assetId"])
}

func (a *TransferWithSigTestSuite) Test_assetId_absence() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideUnit{}, rs["assetId"])
}

func (a *TransferWithSigTestSuite) Test_recipient() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideRecipient(a.tx.Recipient), rs["recipient"])
}

func (a *TransferWithSigTestSuite) Test_attachment() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	attachmentBytes, err := a.tx.Attachment.Bytes()
	a.NoError(err)
	a.Equal(rideBytes(attachmentBytes), rs["attachment"])
}

func (a *TransferWithSigTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *TransferWithSigTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *TransferWithSigTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *TransferWithSigTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *TransferWithSigTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *TransferWithSigTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *TransferWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *TransferWithSigTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *TransferWithSigTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("TransferTransaction"), rs[instanceFieldName])
}

func TestNewVariablesFromTransferWithSig(t *testing.T) {
	suite.Run(t, new(TransferWithSigTestSuite))
}

type TransferWithProofsTestSuite struct {
	suite.Suite
	tx *proto.TransferWithProofs
	f  func(scheme byte, tx *proto.TransferWithProofs) (rideObject, error)
}

func (a *TransferWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.TransferWithProofs.Transaction.Clone()
	a.f = transferWithProofsToObject
}

func (a *TransferWithProofsTestSuite) Test_feeAssetId_Presence() {
	a.tx.Transfer.FeeAsset = _asset
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(_digest.Bytes()), rs["feeAssetId"])
}

func (a *TransferWithProofsTestSuite) Test_feeAssetId_Absence() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideUnit{}, rs["feeAssetId"])
}

func (a *TransferWithProofsTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["amount"])
}

func (a *TransferWithProofsTestSuite) Test_assetId_presence() {
	a.tx.Transfer.AmountAsset = _asset
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), rs["assetId"])
}

func (a *TransferWithProofsTestSuite) Test_assetId_absence() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideUnit{}, rs["assetId"])
}

func (a *TransferWithProofsTestSuite) Test_recipient() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideRecipient(a.tx.Recipient), rs["recipient"])
}

func (a *TransferWithProofsTestSuite) Test_attachment() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	attachmentBytes, err := a.tx.Attachment.Bytes()
	a.NoError(err)
	a.Equal(rideBytes(attachmentBytes), rs["attachment"])
}

func (a *TransferWithProofsTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *TransferWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *TransferWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *TransferWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *TransferWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *TransferWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *TransferWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0].Bytes())
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *TransferWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *TransferWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("TransferTransaction"), rs[instanceFieldName])
}

func TestNewVariablesFromTransferWithProofs(t *testing.T) {
	suite.Run(t, new(TransferWithProofsTestSuite))
}

type GenesisTestSuite struct {
	suite.Suite
	tx *proto.Genesis
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *GenesisTestSuite) SetupTest() {
	tx := &proto.Genesis{}
	if err := copier.Copy(tx, byte_helpers.Genesis.Transaction); err != nil {
		panic(err.Error())
	}
	a.tx = tx
	a.f = transactionToObject
}

func (a *GenesisTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["amount"])
}

func (a *GenesisTestSuite) Test_recipient() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideRecipient(proto.NewRecipientFromAddress(a.tx.Recipient)), rs["recipient"])
}

func (a *GenesisTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID(proto.MainNetScheme)
	a.NoError(err)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *GenesisTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(0), rs["fee"])
}

func (a *GenesisTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *GenesisTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func TestNewVariablesFromGenesis(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

type PaymentTestSuite struct {
	suite.Suite
	tx *proto.Payment
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *PaymentTestSuite) SetupTest() {
	tx := &proto.Payment{}
	if err := copier.Copy(tx, byte_helpers.Payment.Transaction); err != nil {
		panic(err.Error())
	}
	a.tx = tx
	a.f = transactionToObject
}

func (a *PaymentTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["amount"])
}

func (a *PaymentTestSuite) Test_recipient() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideRecipient(proto.NewRecipientFromAddress(a.tx.Recipient)), rs["recipient"])
}

func (a *PaymentTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID(proto.MainNetScheme)
	a.NoError(err)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *PaymentTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *PaymentTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *PaymentTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *PaymentTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *PaymentTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *PaymentTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *PaymentTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *PaymentTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("PaymentTransaction"), rs[instanceFieldName])
}

func TestNewVariablesFromPayment(t *testing.T) {
	suite.Run(t, new(PaymentTestSuite))
}

type ReissueWithSigTestSuite struct {
	suite.Suite
	tx *proto.ReissueWithSig
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *ReissueWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.ReissueWithSig.Transaction.Clone()
	a.f = transactionToObject
}

func (a *ReissueWithSigTestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["quantity"])
}

func (a *ReissueWithSigTestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *ReissueWithSigTestSuite) Test_reissuable() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBoolean(a.tx.Reissuable), rs["reissuable"])
}

func (a *ReissueWithSigTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *ReissueWithSigTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *ReissueWithSigTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *ReissueWithSigTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *ReissueWithSigTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *ReissueWithSigTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *ReissueWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *ReissueWithSigTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *ReissueWithSigTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("ReissueTransaction"), rs[instanceFieldName])
}

//ReissueTransaction
func TestNewVariablesFromReissueWithSig(t *testing.T) {
	suite.Run(t, new(ReissueWithSigTestSuite))
}

type ReissueWithProofsTestSuite struct {
	suite.Suite
	tx *proto.ReissueWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *ReissueWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.ReissueWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *ReissueWithProofsTestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["quantity"])
}

func (a *ReissueWithProofsTestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *ReissueWithProofsTestSuite) Test_reissuable() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBoolean(a.tx.Reissuable), rs["reissuable"])
}

func (a *ReissueWithProofsTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *ReissueWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *ReissueWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *ReissueWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *ReissueWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *ReissueWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *ReissueWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *ReissueWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *ReissueWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("ReissueTransaction"), rs[instanceFieldName])
}

//ReissueTransaction
func TestNewVariablesFromReissueWithProofs(t *testing.T) {
	suite.Run(t, new(ReissueWithProofsTestSuite))
}

type BurnWithSigTestSuite struct {
	suite.Suite
	tx *proto.BurnWithSig
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *BurnWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.BurnWithSig.Transaction.Clone()
	a.f = transactionToObject
}

func (a *BurnWithSigTestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["quantity"])
}

func (a *BurnWithSigTestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *BurnWithSigTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *BurnWithSigTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *BurnWithSigTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *BurnWithSigTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1), rs["version"])
}

func (a *BurnWithSigTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *BurnWithSigTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *BurnWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *BurnWithSigTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *BurnWithSigTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("BurnTransaction"), rs[instanceFieldName])
}

//BurnWithSig
func TestNewVariablesFromBurnWithSig(t *testing.T) {
	suite.Run(t, new(BurnWithSigTestSuite))
}

type BurnWithProofsTestSuite struct {
	suite.Suite
	tx *proto.BurnWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *BurnWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.BurnWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *BurnWithProofsTestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["quantity"])
}

func (a *BurnWithProofsTestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *BurnWithProofsTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *BurnWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *BurnWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *BurnWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(2), rs["version"])
}

func (a *BurnWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *BurnWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *BurnWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *BurnWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *BurnWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("BurnTransaction"), rs[instanceFieldName])
}

//BurnWithProofs
func TestNewVariablesFromBurnWithProofs(t *testing.T) {
	suite.Run(t, new(BurnWithProofsTestSuite))
}

type MassTransferWithProofsTestSuite struct {
	suite.Suite
	tx *proto.MassTransferWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *MassTransferWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.MassTransferWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *MassTransferWithProofsTestSuite) Test_assetId_presence() {
	a.tx.Asset = _asset
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), rs["assetId"])
}

func (a *MassTransferWithProofsTestSuite) Test_assetId_absence() {
	a.tx.Asset = proto.OptionalAsset{}
	rs, err := a.f(proto.MainNetScheme, a.tx)

	a.NoError(err)
	a.Equal(rideUnit{}, rs["assetId"])
}

func (a *MassTransferWithProofsTestSuite) Test_totalAmount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["totalAmount"])
}

func (a *MassTransferWithProofsTestSuite) Test_transfers() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)

	m := make(rideObject)
	m["recipient"] = rideRecipient(a.tx.Transfers[0].Recipient)
	m["amount"] = rideInt(int64(a.tx.Transfers[0].Amount))
	a.Equal(rideList{m}, rs["transfers"])
}

func (a *MassTransferWithProofsTestSuite) Test_transferCount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1), rs["transferCount"])
}

func (a *MassTransferWithProofsTestSuite) Test_attachment() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	attachmentBytes, err := a.tx.Attachment.Bytes()
	a.NoError(err)
	a.Equal(rideBytes(attachmentBytes), rs["attachment"])
}

func (a *MassTransferWithProofsTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *MassTransferWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *MassTransferWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *MassTransferWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1), rs["version"])
}

func (a *MassTransferWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *MassTransferWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *MassTransferWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *MassTransferWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *MassTransferWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("MassTransferTransaction"), rs[instanceFieldName])
}

//MassTransferTransaction
func TestNewVariablesFromMassTransferWithProofs(t *testing.T) {
	suite.Run(t, new(MassTransferWithProofsTestSuite))
}

type ExchangeWithSigTestSuite struct {
	suite.Suite
	tx *proto.ExchangeWithSig
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *ExchangeWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.ExchangeWithSig.Transaction.Clone()
	a.f = transactionToObject
}

func (a *ExchangeWithSigTestSuite) Test_buyOrder() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs["buyOrder"].instanceOf())
}

func (a *ExchangeWithSigTestSuite) Test_sellOrder() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs["sellOrder"].instanceOf())
}

func (a *ExchangeWithSigTestSuite) Test_price() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["price"])
}

func (a *ExchangeWithSigTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["amount"])
}

func (a *ExchangeWithSigTestSuite) Test_buyMatcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(10000), rs["buyMatcherFee"])
}

func (a *ExchangeWithSigTestSuite) Test_sellMatcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(10000), rs["sellMatcherFee"])
}

func (a *ExchangeWithSigTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *ExchangeWithSigTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *ExchangeWithSigTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *ExchangeWithSigTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1), rs["version"])
}

func (a *ExchangeWithSigTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}
func (a *ExchangeWithSigTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *ExchangeWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *ExchangeWithSigTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *ExchangeWithSigTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("ExchangeTransaction"), rs[instanceFieldName])
}

//ExchangeWithSig
func TestNewVariablesFromExchangeWithSig(t *testing.T) {
	suite.Run(t, new(ExchangeWithSigTestSuite))
}

type ExchangeWithProofsTestSuite struct {
	suite.Suite
	tx *proto.ExchangeWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *ExchangeWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.ExchangeWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *ExchangeWithProofsTestSuite) Test_price() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["price"])
}

func (a *ExchangeWithProofsTestSuite) Test_buyOrder() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs["buyOrder"].instanceOf())
}

func (a *ExchangeWithProofsTestSuite) Test_sellOrder() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs["sellOrder"].instanceOf())
}

func (a *ExchangeWithProofsTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["amount"])
}

func (a *ExchangeWithProofsTestSuite) Test_buyMatcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(10000), rs["buyMatcherFee"])
}

func (a *ExchangeWithProofsTestSuite) Test_sellMatcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(10000), rs["sellMatcherFee"])
}

func (a *ExchangeWithProofsTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *ExchangeWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *ExchangeWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *ExchangeWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(2), rs["version"])
}

func (a *ExchangeWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *ExchangeWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *ExchangeWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *ExchangeWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *ExchangeWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("ExchangeTransaction"), rs[instanceFieldName])
}

//ExchangeWithProofs
func TestNewVariablesFromExchangeWithProofs(t *testing.T) {
	suite.Run(t, new(ExchangeWithProofsTestSuite))
}

type OrderTestSuite struct {
	suite.Suite
	tx proto.Order
	f  func(scheme proto.Scheme, tx proto.Order) (rideObject, error)
	d  crypto.Digest
	aa proto.OptionalAsset
	pa proto.OptionalAsset
}

func (a *OrderTestSuite) SetupTest() {
	sk, pk, _ := crypto.GenerateKeyPair([]byte("test"))
	a.d, _ = crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	a.aa = *proto.NewOptionalAssetFromDigest(a.d)
	a.pa = *proto.NewOptionalAssetFromDigest(a.d)
	_, matcherPk, _ := crypto.GenerateKeyPair([]byte("test1"))

	sellOrder := proto.NewUnsignedOrderV1(
		pk,
		matcherPk,
		a.aa,
		a.pa,
		proto.Sell,
		100000,
		10000,
		proto.Timestamp(1544715621),
		proto.Timestamp(1544715621),
		10000)

	a.NoError(sellOrder.Sign(proto.MainNetScheme, sk))

	a.tx = sellOrder
	a.f = orderToObject
}

func (a *OrderTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID()
	a.NoError(err)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *OrderTestSuite) Test_matcherPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	tmp := a.tx.GetMatcherPK()
	a.Equal(rideBytes(tmp.Bytes()), rs["matcherPublicKey"])
}

func (a *OrderTestSuite) Test_assetPair() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(assetPairToObject(a.aa, a.pa), rs["assetPair"])
}

func (a *OrderTestSuite) Test_orderType() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Sell", rs["orderType"].instanceOf())
}

func (a *OrderTestSuite) Test_price() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["price"])
}

func (a *OrderTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(10000), rs["amount"])
}

func (a *OrderTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(byte_helpers.TIMESTAMP)), rs["timestamp"])
}

func (a *OrderTestSuite) Test_expiration() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(byte_helpers.TIMESTAMP)), rs["expiration"])
}

func (a *OrderTestSuite) Test_matcherFee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(10000), rs["matcherFee"])
}

func (a *OrderTestSuite) Test_matcherFeeAssetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideUnit{}, rs["matcherFeeAssetId"])
}

func (a *OrderTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.GetSenderPK())
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *OrderTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	pk := a.tx.GetSenderPK()
	a.Equal(rideBytes(pk.Bytes()), rs["senderPublicKey"])
}

func (a *OrderTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	proofs, _ := a.tx.GetProofs()
	sig, _ := crypto.NewSignatureFromBytes(proofs.Proofs[0])
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *OrderTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	p, _ := a.tx.GetProofs()
	a.Equal(rideList{rideBytes(p.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *OrderTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("Order", rs.instanceOf())
}

//Order
func TestNewVariablesFromOrderV1(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}

type SetAssetScriptWithProofsTestSuite struct {
	suite.Suite
	tx *proto.SetAssetScriptWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *SetAssetScriptWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.SetAssetScriptWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *SetAssetScriptWithProofsTestSuite) Test_script() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	av, ok := rs["script"]
	a.Assert().True(ok)
	a.Equal(rideBytes("hello"), av)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), rs["assetId"])
}

func (a *SetAssetScriptWithProofsTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID(proto.MainNetScheme)
	a.NoError(err)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *SetAssetScriptWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *SetAssetScriptWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *SetAssetScriptWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1), rs["version"])
}

func (a *SetAssetScriptWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *SetAssetScriptWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *SetAssetScriptWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *SetAssetScriptWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *SetAssetScriptWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("SetAssetScriptTransaction", rs.instanceOf())
}

//SetAssetScriptWithProofs
func TestNewVariablesFromSetAssetScriptWithProofs(t *testing.T) {
	suite.Run(t, new(SetAssetScriptWithProofsTestSuite))
}

type InvokeScriptWithProofsTestSuite struct {
	suite.Suite
	tx *proto.InvokeScriptWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *InvokeScriptWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.InvokeScriptWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *InvokeScriptWithProofsTestSuite) Test_dappAddress() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideRecipient(a.tx.ScriptRecipient), rs["dApp"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_payment_presence() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	payment, ok := rs["payment"].(rideObject)
	a.Assert().True(ok)
	asset, ok := payment["assetId"]
	a.Assert().True(ok)
	a.Equal(rideBytes(byte_helpers.Digest.Bytes()), asset)

	amount, ok := payment["amount"]
	a.Assert().True(ok)
	a.Equal(rideInt(100000), amount)
}

func (a *InvokeScriptWithProofsTestSuite) Test_payment_absence() {
	a.tx.Payments = nil
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideUnit{}, rs["payment"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_feeAssetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(byte_helpers.Digest.Bytes()), rs["feeAssetId"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_function() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("funcname"), rs["function"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_args() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideString("StringArgument")}, rs["args"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_id() {
	rs, err := a.f(proto.MainNetScheme, a.tx)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), rs["id"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1), rs["version"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *InvokeScriptWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *InvokeScriptWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("InvokeScriptTransaction", rs.instanceOf())
}

//InvokeScriptTransaction
func TestNewVariablesFromInvokeScriptWithProofs(t *testing.T) {
	suite.Run(t, new(InvokeScriptWithProofsTestSuite))
}

type IssueWithSigTestSuite struct {
	suite.Suite
	tx *proto.IssueWithSig
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *IssueWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.IssueWithSig.Transaction.Clone()
	a.f = transactionToObject
}

func (a *IssueWithSigTestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1000), rs["quantity"])
}

func (a *IssueWithSigTestSuite) Test_name() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("name"), rs["name"])
}

func (a *IssueWithSigTestSuite) Test_description() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("description"), rs["description"])
}

func (a *IssueWithSigTestSuite) Test_reissuable() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBoolean(a.tx.Reissuable), rs["reissuable"])
}

func (a *IssueWithSigTestSuite) Test_decimals() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(4), rs["decimals"])
}

func (a *IssueWithSigTestSuite) Test_script() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideUnit{}, rs["script"])
}

func (a *IssueWithSigTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *IssueWithSigTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *IssueWithSigTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *IssueWithSigTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *IssueWithSigTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *IssueWithSigTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *IssueWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *IssueWithSigTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *IssueWithSigTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("IssueTransaction", rs.instanceOf())
}

func TestNewVariablesFromIssueWithSig(t *testing.T) {
	suite.Run(t, new(IssueWithSigTestSuite))
}

type IssueWithProofsTestSuite struct {
	suite.Suite
	tx *proto.IssueWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *IssueWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.IssueWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *IssueWithProofsTestSuite) Test_quantity() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1000), rs["quantity"])
}

func (a *IssueWithProofsTestSuite) Test_name() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("name"), rs["name"])
}

func (a *IssueWithProofsTestSuite) Test_description() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString("description"), rs["description"])
}

func (a *IssueWithProofsTestSuite) Test_reissuable() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBoolean(a.tx.Reissuable), rs["reissuable"])
}

func (a *IssueWithProofsTestSuite) Test_decimals() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(4), rs["decimals"])
}

func (a *IssueWithProofsTestSuite) Test_script() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	av, ok := rs["script"]
	a.Assert().True(ok)
	a.Equal(rideBytes("script"), av)
}

func (a *IssueWithProofsTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *IssueWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *IssueWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *IssueWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *IssueWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *IssueWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *IssueWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *IssueWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *IssueWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("IssueTransaction", rs.instanceOf())
}

func TestNewVariablesFromIssueWithProofs(t *testing.T) {
	suite.Run(t, new(IssueWithProofsTestSuite))
}

//
type LeaseWithSigTestSuite struct {
	suite.Suite
	tx *proto.LeaseWithSig
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *LeaseWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseWithSig.Transaction.Clone()
	a.f = transactionToObject
}

func (a *LeaseWithSigTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["amount"])
}

func (a *LeaseWithSigTestSuite) Test_recipient() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideRecipient(a.tx.Recipient), rs["recipient"])
}

func (a *LeaseWithSigTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *LeaseWithSigTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *LeaseWithSigTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *LeaseWithSigTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *LeaseWithSigTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *LeaseWithSigTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *LeaseWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *LeaseWithSigTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *LeaseWithSigTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("LeaseTransaction", rs.instanceOf())
}

func TestNewVariablesFromLeaseWithSig(t *testing.T) {
	suite.Run(t, new(LeaseWithSigTestSuite))
}

//
type LeaseWithProofsTestSuite struct {
	suite.Suite
	tx *proto.LeaseWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *LeaseWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *LeaseWithProofsTestSuite) Test_amount() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(100000), rs["amount"])
}

func (a *LeaseWithProofsTestSuite) Test_recipient() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideRecipient(a.tx.Recipient), rs["recipient"])
}

func (a *LeaseWithProofsTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *LeaseWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *LeaseWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *LeaseWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *LeaseWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *LeaseWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *LeaseWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *LeaseWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *LeaseWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("LeaseTransaction", rs.instanceOf())
}

func TestNewVariablesFromLeaseWithProofs(t *testing.T) {
	suite.Run(t, new(LeaseWithProofsTestSuite))
}

//
type LeaseCancelWithSigTestSuite struct {
	suite.Suite
	tx *proto.LeaseCancelWithSig
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *LeaseCancelWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseCancelWithSig.Transaction.Clone()
	a.f = transactionToObject
}

func (a *LeaseCancelWithSigTestSuite) Test_leaseId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(_digest.Bytes()), rs["leaseId"])
}

func (a *LeaseCancelWithSigTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *LeaseCancelWithSigTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *LeaseCancelWithSigTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *LeaseCancelWithSigTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *LeaseCancelWithSigTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *LeaseCancelWithSigTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *LeaseCancelWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *LeaseCancelWithSigTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *LeaseCancelWithSigTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("LeaseCancelTransaction", rs.instanceOf())
}

func TestNewVariablesFromLeaseCancelWithSig(t *testing.T) {
	suite.Run(t, new(LeaseCancelWithSigTestSuite))
}

//
type LeaseCancelWithProofsTestSuite struct {
	suite.Suite
	tx *proto.LeaseCancelWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *LeaseCancelWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseCancelWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *LeaseCancelWithProofsTestSuite) Test_leaseId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.LeaseID.Bytes()), rs["leaseId"])
}

func (a *LeaseCancelWithProofsTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *LeaseCancelWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *LeaseCancelWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *LeaseCancelWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *LeaseCancelWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *LeaseCancelWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *LeaseCancelWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *LeaseCancelWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *LeaseCancelWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("LeaseCancelTransaction", rs.instanceOf())
}

func TestNewVariablesFromLeaseCancelWithProofs(t *testing.T) {
	suite.Run(t, new(LeaseCancelWithProofsTestSuite))
}

//
type DataWithProofsTestSuite struct {
	suite.Suite
	tx *proto.DataWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *DataWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.DataWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *DataWithProofsTestSuite) Test_data() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	list, ok := rs["data"].(rideList)
	a.Assert().True(ok)
	o, ok := list[0].(rideObject)
	a.Assert().True(ok)
	v, ok := o["value"].(rideBytes)
	a.Assert().True(ok)
	a.Equal(rideBytes("hello"), v)
}

func (a *DataWithProofsTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *DataWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *DataWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *DataWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *DataWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *DataWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *DataWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *DataWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *DataWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("DataTransaction", rs.instanceOf())
}

func TestNewVariablesFromDataWithProofsTestSuite(t *testing.T) {
	suite.Run(t, new(DataWithProofsTestSuite))
}

//
type SponsorshipWithProofsTestSuite struct {
	suite.Suite
	tx *proto.SponsorshipWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *SponsorshipWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.SponsorshipWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *SponsorshipWithProofsTestSuite) Test_assetId() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(_digest.Bytes()), rs["assetId"])
}

func (a *SponsorshipWithProofsTestSuite) Test_minSponsoredAssetFee_presence() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(1000), rs["minSponsoredAssetFee"])
}

func (a *SponsorshipWithProofsTestSuite) Test_minSponsoredAssetFee_absence() {
	a.tx.MinAssetFee = 0
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideUnit{}, rs["minSponsoredAssetFee"])
}

func (a *SponsorshipWithProofsTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *SponsorshipWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *SponsorshipWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *SponsorshipWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *SponsorshipWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *SponsorshipWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *SponsorshipWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *SponsorshipWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *SponsorshipWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("SponsorFeeTransaction", rs.instanceOf())
}

func TestNewVariablesFromSponsorshipWithProofs(t *testing.T) {
	suite.Run(t, new(SponsorshipWithProofsTestSuite))
}

//
type CreateAliasWithSigTestSuite struct {
	suite.Suite
	tx *proto.CreateAliasWithSig
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *CreateAliasWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.CreateAliasWithSig.Transaction.Clone()
	a.f = transactionToObject
}

func (a *CreateAliasWithSigTestSuite) Test_alias() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString(a.tx.Alias.String()), rs["alias"])
}

func (a *CreateAliasWithSigTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *CreateAliasWithSigTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *CreateAliasWithSigTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *CreateAliasWithSigTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *CreateAliasWithSigTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *CreateAliasWithSigTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *CreateAliasWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	a.True(crypto.Verify(pub, *a.tx.Signature, rs["bodyBytes"].(rideBytes)))
}

func (a *CreateAliasWithSigTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *CreateAliasWithSigTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("CreateAliasTransaction", rs.instanceOf())
}

func TestNewVariablesFromCreateAliasWithSigTestSuite(t *testing.T) {
	suite.Run(t, new(CreateAliasWithSigTestSuite))
}

//
type CreateAliasWithProofsTestSuite struct {
	suite.Suite
	tx *proto.CreateAliasWithProofs
	f  func(scheme proto.Scheme, tx proto.Transaction) (rideObject, error)
}

func (a *CreateAliasWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.CreateAliasWithProofs.Transaction.Clone()
	a.f = transactionToObject
}

func (a *CreateAliasWithProofsTestSuite) Test_alias() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideString(a.tx.Alias.String()), rs["alias"])
}

func (a *CreateAliasWithProofsTestSuite) Test_id() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	id, _ := a.tx.GetID(proto.MainNetScheme)
	a.Equal(rideBytes(id), rs["id"])
}

func (a *CreateAliasWithProofsTestSuite) Test_fee() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Fee)), rs["fee"])
}

func (a *CreateAliasWithProofsTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Timestamp)), rs["timestamp"])
}

func (a *CreateAliasWithProofsTestSuite) Test_version() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideInt(int64(a.tx.Version)), rs["version"])
}

func (a *CreateAliasWithProofsTestSuite) Test_sender() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, a.tx.SenderPK)
	a.NoError(err)
	a.Equal(rideAddress(addr), rs["sender"])
}

func (a *CreateAliasWithProofsTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), rs["senderPublicKey"])
}

func (a *CreateAliasWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.IsType(rideBytes{}, rs["bodyBytes"])
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, rs["bodyBytes"].(rideBytes)))
}

func (a *CreateAliasWithProofsTestSuite) Test_proofs() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, rs["proofs"])
}

func (a *CreateAliasWithProofsTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.MainNetScheme, a.tx)
	a.Equal("CreateAliasTransaction", rs.instanceOf())
}

func TestNewVariablesFromCreateAliasWithProofsTestSuite(t *testing.T) {
	suite.Run(t, new(CreateAliasWithProofsTestSuite))
}
