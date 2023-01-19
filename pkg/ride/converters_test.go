package ride

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/jinzhu/copier"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

var (
	_digest                             = crypto.MustDigestFromBase58("WmryL34P6UwwUphNbhjBRwiCWxX15Nf5D8T7AmQY7yx")
	_asset                              = *proto.NewOptionalAssetFromDigest(crypto.MustDigestFromBase58("WmryL34P6UwwUphNbhjBRwiCWxX15Nf5D8T7AmQY7yx"))
	_empty                              = rideBytes(nil)
	transactionToObjectWithSchemeTestFn = func(ver ast.LibraryVersion, scheme proto.Scheme, tx proto.Transaction) (rideType, error) {
		return transactionToObject(ver, scheme, false, tx)
	}
	transactionToObjectTestFn = func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error) {
		return transactionToObjectWithSchemeTestFn(ver, proto.TestNetScheme, tx)
	}
)

type TransferWithSigTestSuite struct {
	suite.Suite
	tx *proto.TransferWithSig
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *TransferWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.TransferWithSig.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *TransferWithSigTestSuite) Test_feeAssetId_Presence() {
	a.tx.Transfer.FeeAsset = _asset
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	feeAssetID, err := rs.get(feeAssetIDField)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), feeAssetID)
}

func (a *TransferWithSigTestSuite) Test_feeAssetId_Absence() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	feeAssetID, err := rs.get(feeAssetIDField)
	a.NoError(err)
	a.Equal(rideUnit{}, feeAssetID)
}

func (a *TransferWithSigTestSuite) Test_amount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(10000), amount)
}

func (a *TransferWithSigTestSuite) Test_assetId_presence() {
	a.tx.Transfer.AmountAsset = _asset
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetID, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), assetID)
}

func (a *TransferWithSigTestSuite) Test_assetId_absence() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetID, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideUnit{}, assetID)
}

func (a *TransferWithSigTestSuite) Test_recipient() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	recipient, err := rs.get(recipientField)
	a.NoError(err)
	a.Equal(recipientToObject(a.tx.Recipient), recipient)
}

func (a *TransferWithSigTestSuite) Test_attachment() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	attachmentBytes, err := a.tx.Attachment.Bytes()
	a.NoError(err)
	attachment, err := rs.get(attachmentField)
	a.NoError(err)
	a.Equal(rideBytes(attachmentBytes), attachment)
}

func (a *TransferWithSigTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), id)
}

func (a *TransferWithSigTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *TransferWithSigTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *TransferWithSigTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *TransferWithSigTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *TransferWithSigTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *TransferWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *TransferWithSigTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *TransferWithSigTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(transferTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromTransferWithSig(t *testing.T) {
	suite.Run(t, new(TransferWithSigTestSuite))
}

type TransferWithProofsTestSuite struct {
	suite.Suite
	tx *proto.TransferWithProofs
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *TransferWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.TransferWithProofs.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *TransferWithProofsTestSuite) Test_feeAssetId_Presence() {
	a.tx.Transfer.FeeAsset = _asset
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	feeAssetID, err := rs.get(feeAssetIDField)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), feeAssetID)
}

func (a *TransferWithProofsTestSuite) Test_feeAssetId_Absence() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	feeAssetID, err := rs.get(feeAssetIDField)
	a.NoError(err)
	a.Equal(rideUnit{}, feeAssetID)
}

func (a *TransferWithProofsTestSuite) Test_amount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(100000), amount)
}

func (a *TransferWithProofsTestSuite) Test_assetId_presence() {
	a.tx.Transfer.AmountAsset = _asset
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetID, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), assetID)
}

func (a *TransferWithProofsTestSuite) Test_assetId_absence() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetID, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideUnit{}, assetID)
}

func (a *TransferWithProofsTestSuite) Test_recipient() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	recipient, err := rs.get(recipientField)
	a.NoError(err)
	a.Equal(recipientToObject(a.tx.Recipient), recipient)
}

func (a *TransferWithProofsTestSuite) Test_attachment() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	attachmentBytes, err := a.tx.Attachment.Bytes()
	a.NoError(err)
	attachment, err := rs.get(attachmentField)
	a.NoError(err)
	a.Equal(rideBytes(attachmentBytes), attachment)
}

func (a *TransferWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), id)
}

func (a *TransferWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *TransferWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *TransferWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *TransferWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *TransferWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPK, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPK)
}

func (a *TransferWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0].Bytes())
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *TransferWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *TransferWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(transferTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromTransferWithProofs(t *testing.T) {
	suite.Run(t, new(TransferWithProofsTestSuite))
}

type GenesisTestSuite struct {
	suite.Suite
	tx *proto.Genesis
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *GenesisTestSuite) SetupTest() {
	tx := &proto.Genesis{}
	if err := copier.Copy(tx, byte_helpers.Genesis.Transaction); err != nil {
		panic(err.Error())
	}
	a.tx = tx
	a.f = transactionToObjectTestFn
}

func (a *GenesisTestSuite) Test_amount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(100000), amount)
}

func (a *GenesisTestSuite) Test_recipient() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	recipient, err := rs.get(recipientField)
	a.NoError(err)
	a.Equal(recipientToObject(proto.NewRecipientFromAddress(a.tx.Recipient)), recipient)
}

func (a *GenesisTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID(proto.TestNetScheme)
	a.NoError(err)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *GenesisTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(0), fee)
}

func (a *GenesisTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *GenesisTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func TestNewVariablesFromGenesis(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

type PaymentTestSuite struct {
	suite.Suite
	tx *proto.Payment
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *PaymentTestSuite) SetupTest() {
	tx := &proto.Payment{}
	if err := copier.Copy(tx, byte_helpers.Payment.Transaction); err != nil {
		panic(err.Error())
	}
	a.tx = tx
	a.f = transactionToObjectTestFn
}

func (a *PaymentTestSuite) Test_amount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(100000), amount)
}

func (a *PaymentTestSuite) Test_recipient() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	recipient, err := rs.get(recipientField)
	a.NoError(err)
	a.Equal(recipientToObject(proto.NewRecipientFromAddress(a.tx.Recipient)), recipient)
}

func (a *PaymentTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID(proto.TestNetScheme)
	a.NoError(err)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *PaymentTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *PaymentTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *PaymentTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *PaymentTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *PaymentTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPK, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPK)
}

func (a *PaymentTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *PaymentTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *PaymentTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(paymentTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromPayment(t *testing.T) {
	suite.Run(t, new(PaymentTestSuite))
}

type ReissueWithSigTestSuite struct {
	suite.Suite
	tx *proto.ReissueWithSig
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *ReissueWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.ReissueWithSig.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *ReissueWithSigTestSuite) Test_quantity() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	quantity, err := rs.get(quantityField)
	a.NoError(err)
	a.Equal(rideInt(100000), quantity)
}

func (a *ReissueWithSigTestSuite) Test_assetId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetId, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), assetId)
}

func (a *ReissueWithSigTestSuite) Test_reissuable() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	reissuable, err := rs.get(reissuableField)
	a.NoError(err)
	a.Equal(rideBoolean(a.tx.Reissuable), reissuable)
}

func (a *ReissueWithSigTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *ReissueWithSigTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *ReissueWithSigTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *ReissueWithSigTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *ReissueWithSigTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *ReissueWithSigTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPK, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPK)
}

func (a *ReissueWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *ReissueWithSigTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *ReissueWithSigTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(reissueTransactionTypeName, rs.instanceOf())
}

// ReissueTransaction
func TestNewVariablesFromReissueWithSig(t *testing.T) {
	suite.Run(t, new(ReissueWithSigTestSuite))
}

type ReissueWithProofsTestSuite struct {
	suite.Suite
	tx     *proto.ReissueWithProofs
	scheme proto.Scheme
	f      func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *ReissueWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.ReissueWithProofs.Transaction.Clone()
	a.scheme = proto.MainNetScheme
	a.f = func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error) {
		return transactionToObjectWithSchemeTestFn(ver, a.scheme, tx)
	}
}

func (a *ReissueWithProofsTestSuite) Test_quantity() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	quantity, err := rs.get(quantityField)
	a.NoError(err)
	a.Equal(rideInt(100000), quantity)
}

func (a *ReissueWithProofsTestSuite) Test_assetId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetId, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), assetId)
}

func (a *ReissueWithProofsTestSuite) Test_reissuable() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	reissuable, err := rs.get(reissuableField)
	a.NoError(err)
	a.Equal(rideBoolean(a.tx.Reissuable), reissuable)
}

func (a *ReissueWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *ReissueWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *ReissueWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *ReissueWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *ReissueWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(a.scheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *ReissueWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPK, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPK)
}

func (a *ReissueWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *ReissueWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *ReissueWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(reissueTransactionTypeName, rs.instanceOf())
}

// ReissueTransaction
func TestNewVariablesFromReissueWithProofs(t *testing.T) {
	suite.Run(t, new(ReissueWithProofsTestSuite))
}

type BurnWithSigTestSuite struct {
	suite.Suite
	tx *proto.BurnWithSig
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *BurnWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.BurnWithSig.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *BurnWithSigTestSuite) Test_quantity() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	quantity, err := rs.get(quantityField)
	a.NoError(err)
	a.Equal(rideInt(100000), quantity)
}

func (a *BurnWithSigTestSuite) Test_assetId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetId, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), assetId)
}

func (a *BurnWithSigTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), id)
}

func (a *BurnWithSigTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *BurnWithSigTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *BurnWithSigTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(1), version)
}

func (a *BurnWithSigTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *BurnWithSigTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPK, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPK)
}

func (a *BurnWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *BurnWithSigTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *BurnWithSigTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(burnTransactionTypeName, rs.instanceOf())
}

// BurnWithSig
func TestNewVariablesFromBurnWithSig(t *testing.T) {
	suite.Run(t, new(BurnWithSigTestSuite))
}

type BurnWithProofsTestSuite struct {
	suite.Suite
	tx     *proto.BurnWithProofs
	scheme proto.Scheme
	f      func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *BurnWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.BurnWithProofs.Transaction.Clone()
	a.scheme = proto.MainNetScheme
	a.f = func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error) {
		return transactionToObjectWithSchemeTestFn(ver, a.scheme, tx)
	}
}

func (a *BurnWithProofsTestSuite) Test_quantity() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	quantity, err := rs.get(quantityField)
	a.NoError(err)
	a.Equal(rideInt(100000), quantity)
}

func (a *BurnWithProofsTestSuite) Test_assetId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetId, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), assetId)
}

func (a *BurnWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), id)
}

func (a *BurnWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *BurnWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *BurnWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(2), version)
}

func (a *BurnWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(a.scheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *BurnWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPK, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPK)
}

func (a *BurnWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *BurnWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *BurnWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(burnTransactionTypeName, rs.instanceOf())
}

// BurnWithProofs
func TestNewVariablesFromBurnWithProofs(t *testing.T) {
	suite.Run(t, new(BurnWithProofsTestSuite))
}

type MassTransferWithProofsTestSuite struct {
	suite.Suite
	tx *proto.MassTransferWithProofs
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *MassTransferWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.MassTransferWithProofs.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *MassTransferWithProofsTestSuite) Test_assetId_presence() {
	a.tx.Asset = _asset
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetId, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), assetId)
}

func (a *MassTransferWithProofsTestSuite) Test_assetId_absence() {
	a.tx.Asset = proto.OptionalAsset{}
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetId, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideUnit{}, assetId)
}

func (a *MassTransferWithProofsTestSuite) Test_totalAmount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	totalAmount, err := rs.get(totalAmountField)
	a.NoError(err)
	a.Equal(rideInt(100000), totalAmount)
}

func (a *MassTransferWithProofsTestSuite) Test_transfers() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)

	m := newRideTransfer(
		recipientToObject(a.tx.Transfers[0].Recipient),
		rideInt(int64(a.tx.Transfers[0].Amount)),
	)
	transfers, err := rs.get(transfersField)
	a.NoError(err)
	a.Equal(rideList{m}, transfers)
}

func (a *MassTransferWithProofsTestSuite) Test_transferCount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	transfersCount, err := rs.get(transfersCountField)
	a.NoError(err)
	a.Equal(rideInt(1), transfersCount)
}

func (a *MassTransferWithProofsTestSuite) Test_attachment() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	attachmentBytes, err := a.tx.Attachment.Bytes()
	a.NoError(err)
	attachment, err := rs.get(attachmentField)
	a.NoError(err)
	a.Equal(rideBytes(attachmentBytes), attachment)
}

func (a *MassTransferWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), id)
}

func (a *MassTransferWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *MassTransferWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *MassTransferWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(1), version)
}

func (a *MassTransferWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *MassTransferWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *MassTransferWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *MassTransferWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *MassTransferWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(massTransferTransactionTypeName, rs.instanceOf())
}

// MassTransferTransaction
func TestNewVariablesFromMassTransferWithProofs(t *testing.T) {
	suite.Run(t, new(MassTransferWithProofsTestSuite))
}

type ExchangeWithSigTestSuite struct {
	suite.Suite
	tx *proto.ExchangeWithSig
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *ExchangeWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.ExchangeWithSig.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *ExchangeWithSigTestSuite) Test_buyOrder() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	buyOrder, err := rs.get(buyOrderField)
	a.NoError(err)
	a.Equal(orderTypeName, buyOrder.instanceOf())
}

func (a *ExchangeWithSigTestSuite) Test_sellOrder() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	sellOrder, err := rs.get(sellOrderField)
	a.NoError(err)
	a.Equal(orderTypeName, sellOrder.instanceOf())
}

func (a *ExchangeWithSigTestSuite) Test_price() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	price, err := rs.get(priceField)
	a.NoError(err)
	a.Equal(rideInt(100000), price)
}

func (a *ExchangeWithSigTestSuite) Test_amount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(100000), amount)
}

func (a *ExchangeWithSigTestSuite) Test_buyMatcherFee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	buyMatcherFee, err := rs.get(buyMatcherFeeField)
	a.NoError(err)
	a.Equal(rideInt(10000), buyMatcherFee)
}

func (a *ExchangeWithSigTestSuite) Test_sellMatcherFee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	sellMatcherFee, err := rs.get(sellMatcherFeeField)
	a.NoError(err)
	a.Equal(rideInt(10000), sellMatcherFee)
}

func (a *ExchangeWithSigTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), id)
}

func (a *ExchangeWithSigTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *ExchangeWithSigTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *ExchangeWithSigTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(1), version)
}

func (a *ExchangeWithSigTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}
func (a *ExchangeWithSigTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *ExchangeWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *ExchangeWithSigTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *ExchangeWithSigTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(exchangeTransactionTypeName, rs.instanceOf())
}

// ExchangeWithSig
func TestNewVariablesFromExchangeWithSig(t *testing.T) {
	suite.Run(t, new(ExchangeWithSigTestSuite))
}

type ExchangeWithProofsTestSuite struct {
	suite.Suite
	tx *proto.ExchangeWithProofs
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *ExchangeWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.ExchangeWithProofs.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *ExchangeWithProofsTestSuite) Test_price() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	price, err := rs.get(priceField)
	a.NoError(err)
	a.Equal(rideInt(100000), price)
}

func (a *ExchangeWithProofsTestSuite) Test_buyOrder() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	buyOrder, err := rs.get(buyOrderField)
	a.NoError(err)
	a.Equal(orderTypeName, buyOrder.instanceOf())
}

func (a *ExchangeWithProofsTestSuite) Test_sellOrder() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	sellOrder, err := rs.get(sellOrderField)
	a.NoError(err)
	a.Equal(orderTypeName, sellOrder.instanceOf())
}

func (a *ExchangeWithProofsTestSuite) Test_amount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(100000), amount)
}

func (a *ExchangeWithProofsTestSuite) Test_buyMatcherFee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	buyMatcherFee, err := rs.get(buyMatcherFeeField)
	a.NoError(err)
	a.Equal(rideInt(10000), buyMatcherFee)
}

func (a *ExchangeWithProofsTestSuite) Test_sellMatcherFee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	sellMatcherFee, err := rs.get(sellMatcherFeeField)
	a.NoError(err)
	a.Equal(rideInt(10000), sellMatcherFee)
}

func (a *ExchangeWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), id)
}

func (a *ExchangeWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *ExchangeWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *ExchangeWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(2), version)
}

func (a *ExchangeWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *ExchangeWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *ExchangeWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *ExchangeWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *ExchangeWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(exchangeTransactionTypeName, rs.instanceOf())
}

// ExchangeWithProofs
func TestNewVariablesFromExchangeWithProofs(t *testing.T) {
	suite.Run(t, new(ExchangeWithProofsTestSuite))
}

type OrderTestSuite struct {
	suite.Suite
	tx proto.Order
	f  func(scheme proto.Scheme, tx proto.Order) (rideOrder, error)
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

	a.NoError(sellOrder.Sign(proto.TestNetScheme, sk))

	a.tx = sellOrder
	a.f = orderToObject
}

func (a *OrderTestSuite) Test_id() {
	rs, err := a.f(proto.TestNetScheme, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID()
	a.NoError(err)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *OrderTestSuite) Test_matcherPublicKey() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	tmp := a.tx.GetMatcherPK()
	matcherPublicKey, err := rs.get(matcherPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(tmp.Bytes()), matcherPublicKey)
}

func (a *OrderTestSuite) Test_assetPair() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	assetPair, err := rs.get(assetPairField)
	a.NoError(err)
	a.Equal(assetPairToObject(a.aa, a.pa), assetPair)
}

func (a *OrderTestSuite) Test_orderType() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	orderType, err := rs.get(orderTypeField)
	a.NoError(err)
	a.Equal("Sell", orderType.instanceOf())
}

func (a *OrderTestSuite) Test_price() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	price, err := rs.get(priceField)
	a.NoError(err)
	a.Equal(rideInt(100000), price)
}

func (a *OrderTestSuite) Test_amount() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(10000), amount)
}

func (a *OrderTestSuite) Test_timestamp() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(byte_helpers.TIMESTAMP)), timestamp)
}

func (a *OrderTestSuite) Test_expiration() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	expiration, err := rs.get(expirationField)
	a.NoError(err)
	a.Equal(rideInt(int64(byte_helpers.TIMESTAMP)), expiration)
}

func (a *OrderTestSuite) Test_matcherFee() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	matcherFee, err := rs.get(matcherFeeField)
	a.NoError(err)
	a.Equal(rideInt(10000), matcherFee)
}

func (a *OrderTestSuite) Test_matcherFeeAssetId() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	matcherFeeAssetId, err := rs.get(matcherFeeAssetIDField)
	a.NoError(err)
	a.Equal(rideUnit{}, matcherFeeAssetId)
}

func (a *OrderTestSuite) Test_sender() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	addr, err := a.tx.GetSender(proto.TestNetScheme)
	a.NoError(err)
	wavesAddr, err := addr.ToWavesAddress(proto.TestNetScheme)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(wavesAddr), sender)
}

func (a *OrderTestSuite) Test_senderPublicKey() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	pkBytes := a.tx.GetSenderPKBytes()
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(pkBytes), senderPublicKey)
}

func (a *OrderTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	proofs, _ := a.tx.GetProofs()
	sig, _ := crypto.NewSignatureFromBytes(proofs.Proofs[0])
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *OrderTestSuite) Test_proofs() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	p, _ := a.tx.GetProofs()
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(p.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *OrderTestSuite) Test_instanceFieldName() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	a.Equal(orderTypeName, rs.instanceOf())
}

// OrderV1
func TestNewVariablesFromOrderV1(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}

type EthereumOrderV4TestSuite struct {
	OrderTestSuite
	matcherFeeAssetID proto.OptionalAsset
}

func (a *EthereumOrderV4TestSuite) SetupTest() {
	sk, err := crypto.ECDSAPrivateKeyFromHexString("0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4")
	a.NoError(err)

	a.d, _ = crypto.NewDigestFromBase58("9shLH9vfJxRgbhJ1c3dw2gj5fUGRr8asfUpQjj4rZQKQ")
	a.aa = *proto.NewOptionalAssetFromDigest(a.d)
	a.pa = *proto.NewOptionalAssetFromDigest(a.d)
	a.matcherFeeAssetID = *proto.NewOptionalAssetFromDigest(a.d)
	_, matcherPk, _ := crypto.GenerateKeyPair([]byte("test1"))

	sellOrder := proto.NewUnsignedEthereumOrderV4(
		(*proto.EthereumPublicKey)(sk.PubKey()),
		matcherPk,
		a.aa,
		a.pa,
		proto.Sell,
		100000,
		10000,
		proto.Timestamp(1544715621),
		proto.Timestamp(1544715621),
		10000,
		a.matcherFeeAssetID,
		proto.OrderPriceModeDefault,
	)
	sellOrder.Proofs = proto.NewProofs()

	a.NoError(sellOrder.EthereumSign(proto.TestNetScheme, (*proto.EthereumPrivateKey)(sk)))

	a.tx = sellOrder
	a.f = orderToObject
}

func (a *EthereumOrderV4TestSuite) Test_proofs() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	p, _ := a.tx.GetProofs()
	a.NotNil(p)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{_empty, _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *EthereumOrderV4TestSuite) Test_bodyBytes() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.Nil(bodyBytes)
}

func (a *EthereumOrderV4TestSuite) Test_matcherFeeAssetId() {
	rs, _ := a.f(proto.TestNetScheme, a.tx)
	matcherFeeAssetId, err := rs.get(matcherFeeAssetIDField)
	a.NoError(err)
	a.Equal(rideBytes(a.matcherFeeAssetID.ID.Bytes()), matcherFeeAssetId)
}

// EthereumOrderV4
func TestNewVariablesFromEthereumOrderV4(t *testing.T) {
	suite.Run(t, new(EthereumOrderV4TestSuite))
}

type SetAssetScriptWithProofsTestSuite struct {
	suite.Suite
	tx     *proto.SetAssetScriptWithProofs
	scheme proto.Scheme
	f      func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *SetAssetScriptWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.SetAssetScriptWithProofs.Transaction.Clone()
	a.scheme = proto.MainNetScheme
	a.f = func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error) {
		return transactionToObjectWithSchemeTestFn(ver, a.scheme, tx)
	}
}

func (a *SetAssetScriptWithProofsTestSuite) Test_script() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	script, err := rs.get(scriptField)
	a.NoError(err)
	a.Equal(rideBytes("hello"), script)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_assetId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetId, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.AssetID.Bytes()), assetId)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := a.tx.GetID(a.scheme)
	a.NoError(err)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(1), version)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(a.scheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *SetAssetScriptWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *SetAssetScriptWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(setAssetScriptTransactionTypeName, rs.instanceOf())
}

// SetAssetScriptWithProofs
func TestNewVariablesFromSetAssetScriptWithProofs(t *testing.T) {
	suite.Run(t, new(SetAssetScriptWithProofsTestSuite))
}

type presenceCase struct {
	v        ast.LibraryVersion
	presence bool
}

type InvocationTestSuite struct {
	suite.Suite
	tx             *proto.InvokeScriptWithProofs
	scheme         proto.Scheme
	f              func(ver ast.LibraryVersion, scheme byte, tx proto.Transaction) (rideType, error)
	presenceChecks func(string, []presenceCase, rideType)
}

func (a *InvocationTestSuite) SetupTest() {
	a.tx = byte_helpers.InvokeScriptWithProofs.Transaction.Clone()
	a.f = invocationToObject
	a.scheme = proto.TestNetScheme
	a.presenceChecks = func(fieldName string, cases []presenceCase, expected rideType) {
		for _, testCase := range cases {
			rs, err := a.f(testCase.v, a.scheme, a.tx)
			a.NoError(err, testCase.v)

			fieldVal, err := rs.get(fieldName)
			if testCase.presence {
				a.NoError(err, testCase.v)
				a.Equal(expected, fieldVal, testCase.v)
			} else {
				a.EqualError(err, fmt.Sprintf("type '%s' has no property '%s'", invocationTypeName, fieldName), testCase.v)
			}
		}
	}
}

func (a *InvocationTestSuite) Test_payment() {
	a.presenceChecks(
		paymentField,
		[]presenceCase{
			{v: ast.LibV3, presence: true},
			{v: ast.LibV4, presence: false},
			{v: ast.LibV5, presence: false},
		},
		attachedPaymentToObject(a.tx.Payments[0]),
	)
}

func (a *InvocationTestSuite) Test_callerPublicKey() {
	a.presenceChecks(
		callerPublicKeyField,
		[]presenceCase{
			{v: ast.LibV3, presence: true},
			{v: ast.LibV4, presence: true},
			{v: ast.LibV5, presence: true},
		},
		rideBytes(a.tx.SenderPK.Bytes()),
	)
}

func (a *InvocationTestSuite) Test_feeAssetID() {
	a.presenceChecks(
		feeAssetIDField,
		[]presenceCase{
			{v: ast.LibV3, presence: true},
			{v: ast.LibV4, presence: true},
			{v: ast.LibV5, presence: true},
		},
		rideBytes(a.tx.FeeAsset.ID.Bytes()),
	)
}

func (a *InvocationTestSuite) Test_transactionID() {
	a.presenceChecks(
		transactionIDField,
		[]presenceCase{
			{v: ast.LibV3, presence: true},
			{v: ast.LibV4, presence: true},
			{v: ast.LibV5, presence: true},
		},
		rideBytes(a.tx.ID.Bytes()),
	)
}

func (a *InvocationTestSuite) Test_caller() {
	addr, err := a.tx.GetSender(a.scheme)
	a.NoError(err)
	expected, ok := addr.(proto.WavesAddress)
	a.True(ok)

	a.presenceChecks(
		callerField,
		[]presenceCase{
			{v: ast.LibV3, presence: true},
			{v: ast.LibV4, presence: true},
			{v: ast.LibV5, presence: true},
		},
		rideAddress(expected),
	)
}

func (a *InvocationTestSuite) Test_fee() {
	a.presenceChecks(
		feeField,
		[]presenceCase{
			{v: ast.LibV3, presence: true},
			{v: ast.LibV4, presence: true},
			{v: ast.LibV5, presence: true},
		},
		rideInt(a.tx.Fee),
	)
}

func (a *InvocationTestSuite) Test_payments() {
	payments := make(rideList, len(a.tx.Payments))
	for i, payment := range a.tx.Payments {
		payments[i] = attachedPaymentToObject(payment)
	}

	a.presenceChecks(
		paymentsField,
		[]presenceCase{
			{v: ast.LibV3, presence: false},
			{v: ast.LibV4, presence: true},
			{v: ast.LibV5, presence: true},
		},
		payments,
	)
}

func (a *InvocationTestSuite) Test_originCaller() {
	addr, err := a.tx.GetSender(a.scheme)
	a.NoError(err)
	expected, ok := addr.(proto.WavesAddress)
	a.True(ok)

	a.presenceChecks(
		originCallerField,
		[]presenceCase{
			{v: ast.LibV3, presence: false},
			{v: ast.LibV4, presence: false},
			{v: ast.LibV5, presence: true},
		},
		rideAddress(expected),
	)
}

func (a *InvocationTestSuite) Test_originCallerPublicKey() {
	a.presenceChecks(
		originCallerPublicKeyField,
		[]presenceCase{
			{v: ast.LibV3, presence: false},
			{v: ast.LibV4, presence: false},
			{v: ast.LibV5, presence: true},
		},
		rideBytes(a.tx.SenderPK.Bytes()),
	)
}

// Invocation
func TestNewVariablesInvocation(t *testing.T) {
	suite.Run(t, new(InvocationTestSuite))
}

type InvokeScriptWithProofsTestSuite struct {
	suite.Suite
	tx     *proto.InvokeScriptWithProofs
	scheme proto.Scheme
	f      func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *InvokeScriptWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.InvokeScriptWithProofs.Transaction.Clone()
	a.scheme = proto.MainNetScheme
	a.f = func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error) {
		return transactionToObjectWithSchemeTestFn(ver, a.scheme, tx)
	}
}

func (a *InvokeScriptWithProofsTestSuite) Test_dappAddress() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	dApp, err := rs.get(dAppField)
	a.NoError(err)
	a.Equal(recipientToObject(a.tx.ScriptRecipient), dApp)
}

func (a *InvokeScriptWithProofsTestSuite) Test_payment_presence() {
	rs, err := a.f(ast.LibV3, a.tx)
	a.NoError(err)
	payment, err := rs.get(paymentField)
	a.NoError(err)
	asset, err := payment.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(byte_helpers.Digest.Bytes()), asset)
	amount, err := payment.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(100000), amount)
}

func (a *InvokeScriptWithProofsTestSuite) Test_payment_absence() {
	rs, err := a.f(ast.LibV4, a.tx)
	a.NoError(err)
	_, err = rs.get(paymentField)
	a.EqualError(err, fmt.Sprintf("type '%s' has no property '%s'", invokeScriptTransactionTypeName, paymentField))
}

func (a *InvokeScriptWithProofsTestSuite) Test_payments_presence() {
	rs, err := a.f(ast.LibV4, a.tx)
	a.NoError(err)
	payments, err := rs.get(paymentsField)
	a.NoError(err)

	expectedPayments := make(rideList, len(a.tx.Payments))
	for i, payment := range a.tx.Payments {
		expectedPayments[i] = attachedPaymentToObject(payment)
	}
	a.Equal(expectedPayments, payments)
}

func (a *InvokeScriptWithProofsTestSuite) Test_payments_absence() {
	rs, err := a.f(ast.LibV3, a.tx)
	a.NoError(err)
	_, err = rs.get(paymentsField)
	a.EqualError(err, fmt.Sprintf("type '%s' has no property '%s'", invokeScriptTransactionTypeName, paymentsField))
}

func (a *InvokeScriptWithProofsTestSuite) Test_feeAssetId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	feeAssetId, err := rs.get(feeAssetIDField)
	a.NoError(err)
	a.Equal(rideBytes(byte_helpers.Digest.Bytes()), feeAssetId)
}

func (a *InvokeScriptWithProofsTestSuite) Test_function() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	function, err := rs.get(functionField)
	a.NoError(err)
	a.Equal(rideString("funcname"), function)
}

func (a *InvokeScriptWithProofsTestSuite) Test_args() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	args, err := rs.get(argsField)
	a.NoError(err)
	a.Equal(rideList{rideString("StringArgument")}, args)
}

func (a *InvokeScriptWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.ID.Bytes()), id)
}

func (a *InvokeScriptWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *InvokeScriptWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *InvokeScriptWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(1), version)
}

func (a *InvokeScriptWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(a.scheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *InvokeScriptWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPK, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPK)
}

func (a *InvokeScriptWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *InvokeScriptWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *InvokeScriptWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(invokeScriptTransactionTypeName, rs.instanceOf())
}

// InvokeScriptTransaction
func TestNewVariablesFromInvokeScriptWithProofs(t *testing.T) {
	suite.Run(t, new(InvokeScriptWithProofsTestSuite))
}

type IssueWithSigTestSuite struct {
	suite.Suite
	tx *proto.IssueWithSig
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *IssueWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.IssueWithSig.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *IssueWithSigTestSuite) Test_quantity() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	quantity, err := rs.get(quantityField)
	a.NoError(err)
	a.Equal(rideInt(1000), quantity)
}

func (a *IssueWithSigTestSuite) Test_name() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	name, err := rs.get(nameField)
	a.NoError(err)
	a.Equal(rideString("name"), name)
}

func (a *IssueWithSigTestSuite) Test_description() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	description, err := rs.get(descriptionField)
	a.NoError(err)
	a.Equal(rideString("description"), description)
}

func (a *IssueWithSigTestSuite) Test_reissuable() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	reissuable, err := rs.get(reissuableField)
	a.NoError(err)
	a.Equal(rideBoolean(a.tx.Reissuable), reissuable)
}

func (a *IssueWithSigTestSuite) Test_decimals() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	decimals, err := rs.get(decimalsField)
	a.NoError(err)
	a.Equal(rideInt(4), decimals)
}

func (a *IssueWithSigTestSuite) Test_script() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	script, err := rs.get(scriptField)
	a.NoError(err)
	a.Equal(rideUnit{}, script)
}

func (a *IssueWithSigTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *IssueWithSigTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *IssueWithSigTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *IssueWithSigTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *IssueWithSigTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *IssueWithSigTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *IssueWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *IssueWithSigTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *IssueWithSigTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(issueTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromIssueWithSig(t *testing.T) {
	suite.Run(t, new(IssueWithSigTestSuite))
}

type IssueWithProofsTestSuite struct {
	suite.Suite
	tx     *proto.IssueWithProofs
	scheme proto.Scheme
	f      func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *IssueWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.IssueWithProofs.Transaction.Clone()
	a.scheme = proto.MainNetScheme
	a.f = func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error) {
		return transactionToObjectWithSchemeTestFn(ver, a.scheme, tx)
	}
}

func (a *IssueWithProofsTestSuite) Test_quantity() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	quantity, err := rs.get(quantityField)
	a.NoError(err)
	a.Equal(rideInt(1000), quantity)
}

func (a *IssueWithProofsTestSuite) Test_name() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	name, err := rs.get(nameField)
	a.NoError(err)
	a.Equal(rideString("name"), name)
}

func (a *IssueWithProofsTestSuite) Test_description() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	description, err := rs.get(descriptionField)
	a.NoError(err)
	a.Equal(rideString("description"), description)
}

func (a *IssueWithProofsTestSuite) Test_reissuable() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	reissuable, err := rs.get(reissuableField)
	a.NoError(err)
	a.Equal(rideBoolean(a.tx.Reissuable), reissuable)
}

func (a *IssueWithProofsTestSuite) Test_decimals() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	decimals, err := rs.get(decimalsField)
	a.NoError(err)
	a.Equal(rideInt(4), decimals)
}

func (a *IssueWithProofsTestSuite) Test_script() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	script, err := rs.get(scriptField)
	a.NoError(err)
	a.Equal(rideBytes("script"), script)
}

func (a *IssueWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(a.scheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *IssueWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *IssueWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *IssueWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *IssueWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(a.scheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *IssueWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *IssueWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *IssueWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *IssueWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(issueTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromIssueWithProofs(t *testing.T) {
	suite.Run(t, new(IssueWithProofsTestSuite))
}

type LeaseWithSigTestSuite struct {
	suite.Suite
	tx *proto.LeaseWithSig
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *LeaseWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseWithSig.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *LeaseWithSigTestSuite) Test_amount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(100000), amount)
}

func (a *LeaseWithSigTestSuite) Test_recipient() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	recipient, err := rs.get(recipientField)
	a.NoError(err)
	a.Equal(recipientToObject(a.tx.Recipient), recipient)
}

func (a *LeaseWithSigTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *LeaseWithSigTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *LeaseWithSigTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *LeaseWithSigTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *LeaseWithSigTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *LeaseWithSigTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *LeaseWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *LeaseWithSigTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *LeaseWithSigTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(leaseTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromLeaseWithSig(t *testing.T) {
	suite.Run(t, new(LeaseWithSigTestSuite))
}

type LeaseWithProofsTestSuite struct {
	suite.Suite
	tx *proto.LeaseWithProofs
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *LeaseWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseWithProofs.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *LeaseWithProofsTestSuite) Test_amount() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	amount, err := rs.get(amountField)
	a.NoError(err)
	a.Equal(rideInt(100000), amount)
}

func (a *LeaseWithProofsTestSuite) Test_recipient() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	recipient, err := rs.get(recipientField)
	a.NoError(err)
	a.Equal(recipientToObject(a.tx.Recipient), recipient)
}

func (a *LeaseWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *LeaseWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *LeaseWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *LeaseWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *LeaseWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *LeaseWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *LeaseWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *LeaseWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *LeaseWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(leaseTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromLeaseWithProofs(t *testing.T) {
	suite.Run(t, new(LeaseWithProofsTestSuite))
}

type LeaseCancelWithSigTestSuite struct {
	suite.Suite
	tx *proto.LeaseCancelWithSig
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *LeaseCancelWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseCancelWithSig.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *LeaseCancelWithSigTestSuite) Test_leaseId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	leaseId, err := rs.get(leaseIDField)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), leaseId)
}

func (a *LeaseCancelWithSigTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *LeaseCancelWithSigTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *LeaseCancelWithSigTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *LeaseCancelWithSigTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *LeaseCancelWithSigTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *LeaseCancelWithSigTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *LeaseCancelWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *LeaseCancelWithSigTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *LeaseCancelWithSigTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(leaseCancelTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromLeaseCancelWithSig(t *testing.T) {
	suite.Run(t, new(LeaseCancelWithSigTestSuite))
}

type LeaseCancelWithProofsTestSuite struct {
	suite.Suite
	tx     *proto.LeaseCancelWithProofs
	scheme proto.Scheme
	f      func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *LeaseCancelWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.LeaseCancelWithProofs.Transaction.Clone()
	a.scheme = proto.MainNetScheme
	a.f = func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error) {
		return transactionToObjectWithSchemeTestFn(ver, a.scheme, tx)
	}
}

func (a *LeaseCancelWithProofsTestSuite) Test_leaseId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	leaseId, err := rs.get(leaseIDField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.LeaseID.Bytes()), leaseId)
}

func (a *LeaseCancelWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(a.scheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *LeaseCancelWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *LeaseCancelWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *LeaseCancelWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *LeaseCancelWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(a.scheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *LeaseCancelWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *LeaseCancelWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *LeaseCancelWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *LeaseCancelWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(leaseCancelTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromLeaseCancelWithProofs(t *testing.T) {
	suite.Run(t, new(LeaseCancelWithProofsTestSuite))
}

type DataWithProofsTestSuite struct {
	suite.Suite
	tx *proto.DataWithProofs
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *DataWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.DataWithProofs.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *DataWithProofsTestSuite) Test_data() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	listRaw, err := rs.get(dataField)
	a.NoError(err)
	list, ok := listRaw.(rideList)
	a.Assert().True(ok)
	v, err := list[0].get(valueField)
	a.NoError(err)
	a.Equal(rideBytes("hello"), v)
}

func (a *DataWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *DataWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *DataWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *DataWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *DataWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *DataWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *DataWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *DataWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *DataWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(dataTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromDataWithProofsTestSuite(t *testing.T) {
	suite.Run(t, new(DataWithProofsTestSuite))
}

type SponsorshipWithProofsTestSuite struct {
	suite.Suite
	tx *proto.SponsorshipWithProofs
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *SponsorshipWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.SponsorshipWithProofs.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *SponsorshipWithProofsTestSuite) Test_assetId() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	assetId, err := rs.get(assetIDField)
	a.NoError(err)
	a.Equal(rideBytes(_digest.Bytes()), assetId)
}

func (a *SponsorshipWithProofsTestSuite) Test_minSponsoredAssetFee_presence() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	minSponsoredAssetFee, err := rs.get(minSponsoredAssetFeeField)
	a.NoError(err)
	a.Equal(rideInt(1000), minSponsoredAssetFee)
}

func (a *SponsorshipWithProofsTestSuite) Test_minSponsoredAssetFee_absence() {
	a.tx.MinAssetFee = 0
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	field, err := rs.get(minSponsoredAssetFeeField)
	a.NoError(err)
	a.Equal(rideUnit{}, field)
}

func (a *SponsorshipWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *SponsorshipWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *SponsorshipWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *SponsorshipWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *SponsorshipWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *SponsorshipWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *SponsorshipWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *SponsorshipWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *SponsorshipWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(sponsorFeeTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromSponsorshipWithProofs(t *testing.T) {
	suite.Run(t, new(SponsorshipWithProofsTestSuite))
}

type CreateAliasWithSigTestSuite struct {
	suite.Suite
	tx *proto.CreateAliasWithSig
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *CreateAliasWithSigTestSuite) SetupTest() {
	a.tx = byte_helpers.CreateAliasWithSig.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *CreateAliasWithSigTestSuite) Test_alias() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	alias, err := rs.get(aliasField)
	a.NoError(err)
	a.Equal(rideString(a.tx.Alias.Alias), alias)
}

func (a *CreateAliasWithSigTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *CreateAliasWithSigTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *CreateAliasWithSigTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *CreateAliasWithSigTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *CreateAliasWithSigTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *CreateAliasWithSigTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *CreateAliasWithSigTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	a.True(crypto.Verify(pub, *a.tx.Signature, bodyBytes.(rideBytes)))
}

func (a *CreateAliasWithSigTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Signature.Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *CreateAliasWithSigTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(createAliasTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromCreateAliasWithSigTestSuite(t *testing.T) {
	suite.Run(t, new(CreateAliasWithSigTestSuite))
}

type CreateAliasWithProofsTestSuite struct {
	suite.Suite
	tx *proto.CreateAliasWithProofs
	f  func(ver ast.LibraryVersion, tx proto.Transaction) (rideType, error)
}

func (a *CreateAliasWithProofsTestSuite) SetupTest() {
	a.tx = byte_helpers.CreateAliasWithProofs.Transaction.Clone()
	a.f = transactionToObjectTestFn
}

func (a *CreateAliasWithProofsTestSuite) Test_alias() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	alias, err := rs.get(aliasField)
	a.NoError(err)
	a.Equal(rideString(a.tx.Alias.Alias), alias)
}

func (a *CreateAliasWithProofsTestSuite) Test_id() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	id, _ := a.tx.GetID(proto.TestNetScheme)
	ID, err := rs.get(idField)
	a.NoError(err)
	a.Equal(rideBytes(id), ID)
}

func (a *CreateAliasWithProofsTestSuite) Test_fee() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	fee, err := rs.get(feeField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Fee)), fee)
}

func (a *CreateAliasWithProofsTestSuite) Test_timestamp() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	timestamp, err := rs.get(timestampField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Timestamp)), timestamp)
}

func (a *CreateAliasWithProofsTestSuite) Test_version() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	version, err := rs.get(versionField)
	a.NoError(err)
	a.Equal(rideInt(int64(a.tx.Version)), version)
}

func (a *CreateAliasWithProofsTestSuite) Test_sender() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, a.tx.SenderPK)
	a.NoError(err)
	sender, err := rs.get(senderField)
	a.NoError(err)
	a.Equal(rideAddress(addr), sender)
}

func (a *CreateAliasWithProofsTestSuite) Test_senderPublicKey() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	senderPublicKey, err := rs.get(senderPublicKeyField)
	a.NoError(err)
	a.Equal(rideBytes(a.tx.SenderPK.Bytes()), senderPublicKey)
}

func (a *CreateAliasWithProofsTestSuite) Test_bodyBytes() {
	_, pub, _ := crypto.GenerateKeyPair([]byte("test"))
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	bodyBytes, err := rs.get(bodyBytesField)
	a.NoError(err)
	a.IsType(rideBytes{}, bodyBytes)
	sig, _ := crypto.NewSignatureFromBytes(a.tx.Proofs.Proofs[0])
	a.True(crypto.Verify(pub, sig, bodyBytes.(rideBytes)))
}

func (a *CreateAliasWithProofsTestSuite) Test_proofs() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	proofs, err := rs.get(proofsField)
	a.NoError(err)
	a.Equal(rideList{rideBytes(a.tx.Proofs.Proofs[0].Bytes()), _empty, _empty, _empty, _empty, _empty, _empty, _empty}, proofs)
}

func (a *CreateAliasWithProofsTestSuite) Test_instanceFieldName() {
	rs, err := a.f(ast.LibV1, a.tx)
	a.NoError(err)
	a.Equal(createAliasTransactionTypeName, rs.instanceOf())
}

func TestNewVariablesFromCreateAliasWithProofsTestSuite(t *testing.T) {
	suite.Run(t, new(CreateAliasWithProofsTestSuite))
}

func defaultEthLegacyTxData(value int64, to *proto.EthereumAddress, data []byte, gas uint64) *proto.EthereumLegacyTx {
	v := big.NewInt(87) // MainNet byte
	v.Mul(v, big.NewInt(2))
	v.Add(v, big.NewInt(35))

	return &proto.EthereumLegacyTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     data,
		GasPrice: big.NewInt(1),
		Nonce:    1479168000000,
		Gas:      gas,
		V:        v,
	}
}

func TestEthereumTransferWavesTransformTxToRideObj(t *testing.T) {
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	txData := defaultEthLegacyTxData(1000000000000000, &recipientEth, nil, 100000)
	tx := proto.NewEthereumTransaction(txData, proto.NewEthereumTransferWavesTxKind(), &crypto.Digest{}, &senderPK, 0)

	rideObj, err := transactionToObject(ast.LibV6, proto.TestNetScheme, false, &tx)
	assert.NoError(t, err)

	sender, err := tx.WavesAddressFrom(proto.TestNetScheme)
	assert.NoError(t, err)
	recipient, err := tx.WavesAddressTo(proto.TestNetScheme)
	assert.NoError(t, err)
	senderPublicKey, err := rideObj.get(senderPublicKeyField)
	assert.NoError(t, err)
	assert.Equal(t, rideBytes(senderPK.SerializeXYCoordinates()), senderPublicKey)
	senderF, err := rideObj.get(senderField)
	assert.NoError(t, err)
	assert.Equal(t, rideAddress(sender), senderF)
	recipientF, err := rideObj.get(recipientField)
	assert.NoError(t, err)
	assert.Equal(t, recipientToObject(proto.NewRecipientFromAddress(*recipient)), recipientF)
	amount, err := rideObj.get(amountField)
	assert.NoError(t, err)
	assert.Equal(t, rideInt(100000), amount)
	fee, err := rideObj.get(feeField)
	assert.NoError(t, err)
	assert.Equal(t, rideInt(100000), fee)
}

func makeLessDataAmount(t *testing.T, decodedData *ethabi.DecodedCallData) {
	v, ok := decodedData.Inputs[1].Value.(ethabi.BigInt)
	assert.True(t, ok)
	res := new(big.Int).Div(v.V, big.NewInt(int64(proto.DiffEthWaves)))
	decodedData.Inputs[1].Value = ethabi.BigInt{V: res}
}

func TestEthereumTransferAssetsTransformTxToRideObj(t *testing.T) {
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)
	//var TxSeveralData []proto.EthereumTxData
	//TxSeveralData = append(TxSeveralData, defaultEthereumLegacyTxData(1000000000000000, &recipientEth), defaultEthereumDynamicFeeTx(1000000000000000, &recipientEth), defaultEthereumAccessListTx(1000000000000000, &recipientEth))
	/*
		from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5
		transfer(address,uint256)
		1 = 0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c
		2 = 209470300000000000000000
	*/
	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)
	var txData proto.EthereumTxData = defaultEthLegacyTxData(1000000000000000, &recipientEth, data, 100000)
	tx := proto.NewEthereumTransaction(txData, nil, &crypto.Digest{}, &senderPK, 0)
	db := ethabi.NewErc20MethodsMap()
	assert.NotNil(t, tx.Data())
	decodedData, err := db.ParseCallDataRide(tx.Data())
	assert.NoError(t, err)
	makeLessDataAmount(t, decodedData)

	assetID := (*proto.AssetID)(tx.To())
	var r crypto.Digest
	copy(r[:20], assetID[:])
	asset := &proto.AssetInfo{ID: r}

	erc20arguments, err := ethabi.GetERC20TransferArguments(decodedData)
	assert.NoError(t, err)

	tx.TxKind = proto.NewEthereumTransferAssetsErc20TxKind(*decodedData, *proto.NewOptionalAssetFromDigest(asset.ID), erc20arguments)

	rideObj, err := transactionToObject(ast.LibV6, proto.TestNetScheme, false, &tx)
	assert.NoError(t, err)

	sender, err := tx.WavesAddressFrom(proto.TestNetScheme)
	assert.NoError(t, err)

	senderPublicKey, err := rideObj.get(senderPublicKeyField)
	assert.NoError(t, err)
	assert.Equal(t, rideBytes(senderPK.SerializeXYCoordinates()), senderPublicKey)
	senderF, err := rideObj.get(senderField)
	assert.NoError(t, err)
	assert.Equal(t, rideAddress(sender), senderF)

	erc20TransferRecipient, err := proto.EthereumAddress(erc20arguments.Recipient).ToWavesAddress(proto.TestNetScheme)
	assert.NoError(t, err)

	recipientF, err := rideObj.get(recipientField)
	assert.NoError(t, err)
	assert.Equal(t, recipientToObject(proto.NewRecipientFromAddress(erc20TransferRecipient)), recipientF)
	amount, err := rideObj.get(amountField)
	assert.NoError(t, err)
	assert.Equal(t, rideInt(20947030000000), amount)
	fee, err := rideObj.get(feeField)
	assert.NoError(t, err)
	assert.Equal(t, rideInt(100000), fee)
}

func TestArgumentsConversion(t *testing.T) {
	ri := rideInt(12345)
	rs := rideString("xxx")
	rt := rideBoolean(true)
	rb := rideBytes([]byte{0xca, 0xfe, 0xbe, 0xbe, 0xde, 0xad, 0xbe, 0xef})
	rl := rideList([]rideType{ri, rs, rt, rb})
	ru := rideUnit{}
	ra := rideAddress(proto.MustAddressFromString("3N9b3KejqpXFkbvZBKobythymXM4d3m2oRD"))
	for _, test := range []struct {
		args  rideList
		check bool
		ok    bool
		res   []rideType
	}{
		{rideList([]rideType{ri, rs, rt, rb}), true, true, []rideType{ri, rs, rt, rb}},
		{rideList([]rideType{ri, rs, rt, rb, rl}), true, true, []rideType{ri, rs, rt, rb, rl}},
		{rideList([]rideType{rl, rl, rl, rl, rl}), true, true, []rideType{rl, rl, rl, rl, rl}},
		{rideList([]rideType{ru, ri, rs, rt, rb, rl}), true, false, nil},
		{rideList([]rideType{ri, rs, rt, rb, rideList([]rideType{ri, rs, rt, rb, ru})}), true, false, nil},
		{rideList([]rideType{ru, ri, rs, rt, rb, rl, ra}), true, false, nil},
		{rideList([]rideType{ru, ri, rs, rt, rb, rideList([]rideType{ri, rs, ra})}), true, false, nil},
		{rideList([]rideType{ri, rs, rt, rb}), false, true, []rideType{ri, rs, rt, rb}},
		{rideList([]rideType{ri, rs, rt, rb, rl}), false, true, []rideType{ri, rs, rt, rb, rl}},
		{rideList([]rideType{rl, rl, rl, rl, rl}), false, true, []rideType{rl, rl, rl, rl, rl}},
		{rideList([]rideType{ru, ri, rs, rt, rb, rl}), false, true, []rideType{ru, ri, rs, rt, rb, rl}},
		{rideList([]rideType{ru, ri, rs, rt, rb, rl, ra}), false, true, []rideType{ru, ri, rs, rt, rb, rl, ra}},
		{rideList([]rideType{ru, ri, rs, rt, rb, rideList([]rideType{ri, rs, ra})}), false, true, []rideType{ru, ri, rs, rt, rb, rideList([]rideType{ri, rs, ra})}},
	} {
		r, err := convertListArguments(test.args, test.check)
		if test.ok {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
		assert.ElementsMatch(t, test.res, r)
	}
}

func TestSetScriptRideObjectScriptField(t *testing.T) {
	const (
		dig = "7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN"
		sig = "3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2"
	)
	bigString := strings.Repeat("1", proto.MaxContractScriptSizeV1V5)
	bigPlusString := bigString + "1"
	mustDecodeBase58 := func(s string) []byte {
		res, err := base58.Decode(s)
		require.NoError(t, err)
		return res
	}
	tests := []struct {
		expectedScriptField            rideType
		scriptBytes                    string
		consensusImprovementsActivated bool
	}{
		{rideUnit{}, "", false},
		{rideBytes(mustDecodeBase58(dig)), dig, false},
		{rideBytes(mustDecodeBase58(bigString)), bigString, false},
		{rideUnit{}, bigPlusString, false},
		{rideBytes(mustDecodeBase58(bigPlusString)), bigPlusString, true},
	}
	for _, tc := range tests {
		testSetScriptTransaction := makeSetScriptTransactionObject(t, sig, dig, tc.scriptBytes, 1, 2, tc.consensusImprovementsActivated)
		actualScriptField, err := testSetScriptTransaction.get(scriptField)
		require.NoError(t, err)
		assert.Equal(t, tc.expectedScriptField, actualScriptField)
	}
}
