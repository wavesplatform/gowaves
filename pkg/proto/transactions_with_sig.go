package proto

import (
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
)

const (
	issueWithSigFixedBodyLen       = 1 + issueLen
	issueWithSigMinBodyLen         = issueWithSigFixedBodyLen + 4 // 4 because of the shortest allowed Asset name of 4 bytes
	issueWithSigMinLen             = 1 + crypto.SignatureSize + issueWithSigMinBodyLen
	transferWithSigFixedBodyLen    = 1 + transferLen
	transferWithSigMinLen          = 1 + crypto.SignatureSize + transferWithSigFixedBodyLen
	reissueWithSigBodyLen          = 1 + reissueLen
	reissueWithSigMinLen           = 1 + crypto.SignatureSize + reissueWithSigBodyLen
	burnWithSigBodyLen             = 1 + burnLen
	burnWithSigLen                 = burnWithSigBodyLen + crypto.SignatureSize
	exchangeWithSigFixedBodyLen    = 1 + 4 + 4 + 8 + 8 + 8 + 8 + 8 + 8
	exchangeWithSigMinLen          = exchangeWithSigFixedBodyLen + orderV1MinLen + orderV1MinLen + crypto.SignatureSize
	leaseWithSigBodyLen            = 1 + leaseLen
	leaseWithSigMinLen             = leaseWithSigBodyLen + crypto.SignatureSize
	leaseCancelWithSigBodyLen      = 1 + leaseCancelLen
	leaseCancelWithSigMinLen       = leaseCancelWithSigBodyLen + crypto.SignatureSize
	createAliasWithSigFixedBodyLen = 1 + createAliasLen
	createAliasWithSigMinLen       = createAliasWithSigFixedBodyLen + crypto.SignatureSize
)

// IssueWithSig transaction is a transaction to issue new asset.
type IssueWithSig struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Issue
}

func (tx *IssueWithSig) Validate(_ Scheme) (Transaction, error) {
	if tx.Version != 1 {
		return tx, errors.Errorf("unexpected version %d for IssueWithSig", tx.Version)
	}
	ok, err := tx.Issue.Valid()
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx IssueWithSig) BinarySize() int {
	return 2 + crypto.SignatureSize + tx.Issue.BinarySize()
}

func (tx IssueWithSig) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx IssueWithSig) GetVersion() byte {
	return tx.Version
}

func (tx *IssueWithSig) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}
	return nil
}

func (tx *IssueWithSig) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *IssueWithSig) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *IssueWithSig) Clone() *IssueWithSig {
	out := &IssueWithSig{}
	_ = copier.Copy(out, tx)
	return out
}

// NewUnsignedIssueWithSig creates new IssueWithSig transaction without signature and ID.
func NewUnsignedIssueWithSig(senderPK crypto.PublicKey, name, description string, quantity uint64, decimals byte, reissuable bool, timestamp, fee uint64) *IssueWithSig {
	i := Issue{
		SenderPK:    senderPK,
		Name:        name,
		Description: description,
		Quantity:    quantity,
		Decimals:    decimals,
		Reissuable:  reissuable,
		Timestamp:   timestamp,
		Fee:         fee,
	}
	return &IssueWithSig{Type: IssueTransaction, Version: 1, Issue: i}
}

func (tx *IssueWithSig) BodyMarshalBinary(Scheme) ([]byte, error) {
	b, err := tx.Issue.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal IssueWithSig body")
	}
	buf := make([]byte, 1+len(b))
	buf[0] = byte(tx.Type)
	copy(buf[1:], b)
	return buf, nil
}

func (tx *IssueWithSig) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < issueWithSigFixedBodyLen {
		return errors.Errorf("%d bytes is not enough for IssueWithSig transaction, expected not less then %d bytes", l, issueWithSigFixedBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != IssueTransaction {
		return errors.Errorf("unexpected transaction type %d for IssueWithSig transaction", tx.Type)
	}
	tx.Version = 1
	var i Issue
	err := i.UnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IssueWithSig body from bytes")
	}
	tx.Issue = i
	return nil
}

// Sign uses secretKey to sing the transaction.
func (tx *IssueWithSig) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueWithSig transaction")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithSig transaction")
	}
	tx.Signature = &s
	id := crypto.MustFastHash(b)
	tx.ID = &id
	return nil
}

// Verify checks that the signature of transaction is a valid signature for given public key.
func (tx *IssueWithSig) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of IssueWithSig transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

// MarshalBinary saves transaction's binary representation to slice of bytes.
func (tx *IssueWithSig) MarshalBinary(scheme Scheme) ([]byte, error) {
	sl := crypto.SignatureSize
	b, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal IssueWithSig transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, 1+sl+bl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.Signature[:])
	copy(buf[1+sl:], b)
	return buf, nil
}

// UnmarshalBinary reads transaction from its binary representation.
func (tx *IssueWithSig) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < issueWithSigMinLen {
		return errors.Errorf("%d is not enough data for IssueWithSig transaction, expected not less then %d", l, issueWithSigMinLen)
	}
	if data[0] != byte(IssueTransaction) {
		return errors.Errorf("incorrect transaction type %d for IssueWithSig transaction", data[0])
	}
	data = data[1:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	data = data[crypto.SignatureSize:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IssueWithSig transaction")
	}
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *IssueWithSig) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *IssueWithSig) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	issueTx, ok := t.(*IssueWithSig)
	if !ok {
		return errors.New("failed to convert result to IssueWithSig")
	}
	*tx = *issueTx
	return nil
}

func (tx *IssueWithSig) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *IssueWithSig) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	issueTx, ok := t.(*IssueWithSig)
	if !ok {
		return errors.New("failed to convert result to IssueWithSig")
	}
	*tx = *issueTx
	return nil
}

func (tx *IssueWithSig) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.Issue.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *IssueWithSig) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Signature == nil {
		return nil, errors.New("no signature provided")
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      proofs.Bytes(),
	}, nil
}

// TransferWithSig transaction to transfer any token from one account to another. Version 1.
type TransferWithSig struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Transfer
}

func (tx TransferWithSig) GetProofs() *ProofsV1 {
	return NewProofsFromSignature(tx.Signature)
}

func (tx *TransferWithSig) Validate(scheme Scheme) (Transaction, error) {
	if tx.Version != 1 {
		return tx, errors.Errorf("unexpected version %d for TransferWithSig", tx.Version)
	}
	ok, err := tx.Transfer.Valid(scheme)
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx TransferWithSig) BinarySize() int {
	return 2 + crypto.SignatureSize + tx.Transfer.BinarySize()
}

func (tx TransferWithSig) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx TransferWithSig) GetVersion() byte {
	return tx.Version
}

func (tx *TransferWithSig) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}
	return nil
}

func (tx *TransferWithSig) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *TransferWithSig) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *TransferWithSig) Clone() *TransferWithSig {
	out := &TransferWithSig{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedTransferWithSig creates new TransferWithSig transaction without signature and ID.
func NewUnsignedTransferWithSig(senderPK crypto.PublicKey, amountAsset, feeAsset OptionalAsset, timestamp, amount, fee uint64, recipient Recipient, attachment Attachment) *TransferWithSig {
	t := Transfer{
		SenderPK:    senderPK,
		Recipient:   recipient,
		AmountAsset: amountAsset,
		Amount:      amount,
		FeeAsset:    feeAsset,
		Fee:         fee,
		Timestamp:   timestamp,
		Attachment:  attachment,
	}
	return &TransferWithSig{Type: TransferTransaction, Version: 1, Transfer: t}
}

func (tx *TransferWithSig) BodyMarshalBinary(Scheme) ([]byte, error) {
	b, err := tx.Transfer.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferWithSig body")
	}
	buf := make([]byte, 1+len(b))
	buf[0] = byte(tx.Type)
	copy(buf[1:], b)
	return buf, nil
}

func (tx *TransferWithSig) BodySerialize(s *serializer.Serializer) error {
	err := s.Byte(byte(tx.Type))
	if err != nil {
		return err
	}
	err = tx.Transfer.Serialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to serialize TransferWithSig body")
	}
	return nil
}

func (tx *TransferWithSig) WriteTo(w io.Writer) (int64, error) {
	s := serializer.New(w)
	err := tx.Serialize(s)
	if err != nil {
		return 0, err
	}
	return s.N(), nil
}

func (tx *TransferWithSig) Serialize(s *serializer.Serializer) error {
	err := s.Byte(byte(tx.Type))
	if err != nil {
		return err
	}
	err = s.Bytes(tx.Signature[:])
	if err != nil {
		return err
	}
	err = tx.BodySerialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal TransferWithSig transaction to bytes")
	}
	return nil
}

func (tx *TransferWithSig) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < transferWithSigFixedBodyLen {
		return errors.Errorf("%d bytes is not enough for TransferWithSig transaction, expected not less then %d bytes", l, transferWithSigFixedBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != TransferTransaction {
		return errors.Errorf("unexpected transaction type %d for TransferWithSig transaction", tx.Type)
	}
	tx.Version = 1
	var t Transfer
	err := t.UnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferWithSig body from bytes")
	}
	tx.Transfer = t
	return nil
}

// Sign calculates a signature and a digest as an ID of the transaction.
func (tx *TransferWithSig) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferWithSig transaction")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferWithSig transaction")
	}
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferWithSig transaction")
	}
	tx.ID = &d
	return nil
}

// Verify use given public key to verify that the signature is valid.
func (tx *TransferWithSig) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of TransferWithSig transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

// MarshalBinary saves transaction to its binary representation.
func (tx *TransferWithSig) MarshalBinary(scheme Scheme) ([]byte, error) {
	sl := crypto.SignatureSize
	b, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferWithSig transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, 1+sl+bl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.Signature[:])
	copy(buf[1+sl:], b)
	return buf, nil
}

// UnmarshalBinary reads transaction from its binary representation.
func (tx *TransferWithSig) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < transferWithSigMinLen {
		return errors.Errorf("not enough data for TransferWithSig transaction, expected not less then %d, received %d", transferWithSigMinLen, l)
	}
	if data[0] != byte(TransferTransaction) {
		return errors.Errorf("incorrect transaction type %d for TransferWithSig transaction", data[0])
	}
	data = data[1:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	data = data[crypto.SignatureSize:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferWithSig transaction")
	}
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *TransferWithSig) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *TransferWithSig) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	transferTx, ok := t.(*TransferWithSig)
	if !ok {
		return errors.New("failed to convert result to TransferWithSig")
	}
	*tx = *transferTx
	return nil
}

func (tx *TransferWithSig) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *TransferWithSig) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	transferTx, ok := t.(*TransferWithSig)
	if !ok {
		return errors.New("failed to convert result to TransferWithSig")
	}
	*tx = *transferTx
	return nil
}

func (tx *TransferWithSig) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData, err := tx.Transfer.ToProtobuf()
	if err != nil {
		return nil, err
	}
	fee := &g.Amount{AssetId: tx.FeeAsset.ToID(), Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *TransferWithSig) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Signature == nil {
		return nil, errors.New("no signature provided")
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      proofs.Bytes(),
	}, nil
}

func (tx *TransferWithSig) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Type      TransactionType   `json:"type"`
		Version   byte              `json:"version,omitempty"`
		ID        *crypto.Digest    `json:"id,omitempty"`
		Signature *crypto.Signature `json:"signature,omitempty"`
		Transfer
	}{}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	tx.Type = tmp.Type
	tx.Version = tmp.Version
	tx.ID = tmp.ID
	tx.Signature = tmp.Signature
	tx.Transfer = tmp.Transfer
	return nil
}

// ReissueWithSig is a transaction that allows to issue new amount of existing token, if it was issued as reissuable.
type ReissueWithSig struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Reissue
}

func (tx *ReissueWithSig) Validate(_ Scheme) (Transaction, error) {
	if tx.Version != 1 {
		return tx, errors.Errorf("unexpected version %d for ReissueWithSig", tx.Version)
	}
	ok, err := tx.Reissue.Valid()
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx ReissueWithSig) BinarySize() int {
	return 2 + crypto.SignatureSize + tx.Reissue.BinarySize()
}

func (tx ReissueWithSig) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx ReissueWithSig) GetVersion() byte {
	return tx.Version
}

func (tx *ReissueWithSig) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}
	return nil
}

func (tx *ReissueWithSig) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *ReissueWithSig) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *ReissueWithSig) Clone() *ReissueWithSig {
	out := &ReissueWithSig{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedReissueWithSig creates new ReissueWithSig transaction without signature and ID.
func NewUnsignedReissueWithSig(senderPK crypto.PublicKey, assetID crypto.Digest, quantity uint64, reissuable bool, timestamp, fee uint64) *ReissueWithSig {
	r := Reissue{
		SenderPK:   senderPK,
		AssetID:    assetID,
		Quantity:   quantity,
		Reissuable: reissuable,
		Fee:        fee,
		Timestamp:  timestamp,
	}
	return &ReissueWithSig{Type: ReissueTransaction, Version: 1, Reissue: r}
}

func (tx *ReissueWithSig) BodyMarshalBinary(Scheme) ([]byte, error) {
	buf := make([]byte, reissueWithSigBodyLen)
	buf[0] = byte(tx.Type)
	b, err := tx.Reissue.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueWithSig transaction to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *ReissueWithSig) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < reissueWithSigBodyLen {
		return errors.Errorf("not enough data for ReissueWithSig transaction %d, expected not less then %d", l, reissueWithSigBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != ReissueTransaction {
		return errors.Errorf("unexpected transaction type %d for ReissueWithSig transaction", tx.Type)
	}
	tx.Version = 1
	var r Reissue
	err := r.UnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueWithSig transaction body")
	}
	tx.Reissue = r
	return nil
}

// Sign use given private key to calculate signature of the transaction.
// This function also calculates digest of transaction data and assigns it to ID field.
func (tx *ReissueWithSig) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueWithSig transaction")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithSig transaction")
	}
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueWithSig transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that the signature of the transaction is valid for given public key.
func (tx *ReissueWithSig) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of ReissueWithSig transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

// MarshalBinary saves the transaction to its binary representation.
func (tx *ReissueWithSig) MarshalBinary(scheme Scheme) ([]byte, error) {
	sl := crypto.SignatureSize
	b, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueWithSig transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, 1+sl+bl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.Signature[:])
	copy(buf[1+sl:], b)
	return buf, nil
}

// UnmarshalBinary reads transaction from its binary representation.
func (tx *ReissueWithSig) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < reissueWithSigMinLen {
		return errors.Errorf("not enough data for ReissueWithSig transaction, expected not less then %d, received %d", reissueWithSigMinLen, l)
	}
	if data[0] != byte(ReissueTransaction) {
		return errors.Errorf("incorrect transaction type %d for ReissueWithSig transaction", data[0])
	}
	data = data[1:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	data = data[crypto.SignatureSize:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueWithSig transaction")
	}
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *ReissueWithSig) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *ReissueWithSig) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	reissueTx, ok := t.(*ReissueWithSig)
	if !ok {
		return errors.New("failed to convert result to ReissueWithSig")
	}
	*tx = *reissueTx
	return nil
}

func (tx *ReissueWithSig) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *ReissueWithSig) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	reissueTx, ok := t.(*ReissueWithSig)
	if !ok {
		return errors.New("failed to convert result to ReissueWithSig")
	}
	*tx = *reissueTx
	return nil
}

func (tx *ReissueWithSig) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.Reissue.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *ReissueWithSig) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Signature == nil {
		return nil, errors.New("no signature provided")
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{
			WavesTransaction: unsigned,
		},
		Proofs: proofs.Bytes(),
	}, nil
}

// BurnWithSig transaction allows to decrease the total supply of the existing asset. Asset must be reissuable.
type BurnWithSig struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Burn
}

func (tx *BurnWithSig) Validate(_ Scheme) (Transaction, error) {
	if tx.Version != 1 {
		return tx, errors.Errorf("unexpected version %d for BurnWithSig", tx.Version)
	}
	ok, err := tx.Burn.Valid()
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx BurnWithSig) BinarySize() int {
	return 1 + crypto.SignatureSize + tx.Burn.BinarySize()
}

func (tx BurnWithSig) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx BurnWithSig) GetVersion() byte {
	return tx.Version
}

func (tx *BurnWithSig) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}
	return nil
}

func (tx *BurnWithSig) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *BurnWithSig) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *BurnWithSig) Clone() *BurnWithSig {
	out := &BurnWithSig{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedBurnWithSig creates new BurnWithSig transaction with no signature and ID.
func NewUnsignedBurnWithSig(senderPK crypto.PublicKey, assetID crypto.Digest, amount, timestamp, fee uint64) *BurnWithSig {
	b := Burn{
		SenderPK:  senderPK,
		AssetID:   assetID,
		Amount:    amount,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &BurnWithSig{Type: BurnTransaction, Version: 1, Burn: b}
}

func (tx *BurnWithSig) BodyMarshalBinary(Scheme) ([]byte, error) {
	buf := make([]byte, burnWithSigBodyLen)
	buf[0] = byte(tx.Type)
	b, err := tx.Burn.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnWithSig transaction to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *BurnWithSig) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < burnWithSigBodyLen {
		return errors.Errorf("%d bytes is not enough for BurnWithSig transaction, expected not less then %d", l, burnWithSigBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != BurnTransaction {
		return errors.Errorf("unexpected transaction type %d for BurnWithSig transaction", tx.Type)
	}
	tx.Version = 1
	var b Burn
	err := b.UnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnWithSig transaction body")
	}
	tx.Burn = b
	return nil
}

// Sign calculates and sets signature and ID of the transaction.
func (tx *BurnWithSig) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnWithSig transaction")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithSig transaction")
	}
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnWithSig transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that the signature of the transaction is valid for the given public key.
func (tx *BurnWithSig) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of BurnWithSig transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

// MarshalBinary saves transaction to
func (tx *BurnWithSig) MarshalBinary(scheme Scheme) ([]byte, error) {
	b, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnWithSig transaction to bytes")
	}
	buf := make([]byte, burnWithSigLen)
	copy(buf, b)
	copy(buf[burnWithSigBodyLen:], tx.Signature[:])
	return buf, nil
}

// UnmarshalBinary reads transaction form its binary representation.
func (tx *BurnWithSig) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < burnWithSigLen {
		return errors.Errorf("not enough data for BurnWithSig transaction, expected not less then %d, received %d", burnWithSigLen, l)
	}
	err := tx.bodyUnmarshalBinary(data[:burnWithSigBodyLen])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnWithSig transaction")
	}
	var s crypto.Signature
	copy(s[:], data[burnWithSigBodyLen:burnWithSigBodyLen+crypto.SignatureSize])
	tx.Signature = &s
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *BurnWithSig) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *BurnWithSig) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	burnTx, ok := t.(*BurnWithSig)
	if !ok {
		return errors.New("failed to convert result to BurnWithSig")
	}
	*tx = *burnTx
	return nil
}

func (tx *BurnWithSig) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *BurnWithSig) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	burnTx, ok := t.(*BurnWithSig)
	if !ok {
		return errors.New("failed to convert result to BurnWithSig")
	}
	*tx = *burnTx
	return nil
}

func (tx *BurnWithSig) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.Burn.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *BurnWithSig) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Signature == nil {
		return nil, errors.New("no signature provided")
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      proofs.Bytes(),
	}, nil
}

// ExchangeWithSig is a transaction to store settlement on blockchain.
type ExchangeWithSig struct {
	Type           TransactionType   `json:"type"`
	Version        byte              `json:"version,omitempty"`
	ID             *crypto.Digest    `json:"id,omitempty"`
	Signature      *crypto.Signature `json:"signature,omitempty"`
	SenderPK       crypto.PublicKey  `json:"senderPublicKey"`
	Order1         *OrderV1          `json:"order1"`
	Order2         *OrderV1          `json:"order2"`
	Price          uint64            `json:"price"`
	Amount         uint64            `json:"amount"`
	BuyMatcherFee  uint64            `json:"buyMatcherFee"`
	SellMatcherFee uint64            `json:"sellMatcherFee"`
	Fee            uint64            `json:"fee"`
	Timestamp      uint64            `json:"timestamp,omitempty"`
}

func (tx ExchangeWithSig) BinarySize() int {
	return 1 + crypto.SignatureSize + 48 + 4 + tx.Order1.BinarySize() + 4 + tx.Order2.BinarySize()
}

func (tx ExchangeWithSig) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx ExchangeWithSig) GetVersion() byte {
	return tx.Version
}

func (tx *ExchangeWithSig) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}
	return nil
}

func (tx *ExchangeWithSig) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *ExchangeWithSig) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *ExchangeWithSig) Clone() *ExchangeWithSig {
	out := &ExchangeWithSig{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

func (tx ExchangeWithSig) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx ExchangeWithSig) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx ExchangeWithSig) GetBuyOrder() (Order, error) {
	return tx.Order1, nil
}

func (tx ExchangeWithSig) GetSellOrder() (Order, error) {
	return tx.Order2, nil
}

func (tx ExchangeWithSig) GetOrder1() Order {
	return tx.Order1
}

func (tx ExchangeWithSig) GetOrder2() Order {
	return tx.Order2
}

func (tx ExchangeWithSig) GetPrice() uint64 {
	return tx.Price
}

func (tx ExchangeWithSig) GetAmount() uint64 {
	return tx.Amount
}

func (tx ExchangeWithSig) GetBuyMatcherFee() uint64 {
	return tx.BuyMatcherFee
}

func (tx ExchangeWithSig) GetSellMatcherFee() uint64 {
	return tx.SellMatcherFee
}
func (tx ExchangeWithSig) GetFee() uint64 {
	return tx.Fee
}

func (tx ExchangeWithSig) GetTimestamp() uint64 {
	return tx.Timestamp
}

func NewUnsignedExchangeWithSig(buy, sell *OrderV1, price, amount, buyMatcherFee, sellMatcherFee, fee, timestamp uint64) *ExchangeWithSig {
	return &ExchangeWithSig{
		Type:           ExchangeTransaction,
		Version:        1,
		SenderPK:       buy.MatcherPK,
		Order1:         buy,
		Order2:         sell,
		Price:          price,
		Amount:         amount,
		BuyMatcherFee:  buyMatcherFee,
		SellMatcherFee: sellMatcherFee,
		Fee:            fee,
		Timestamp:      timestamp,
	}
}

func (tx *ExchangeWithSig) Validate(_ Scheme) (Transaction, error) {
	if tx.Version != 1 {
		return tx, errors.Errorf("unexpected version %d for ExchangeWithSig", tx.Version)
	}
	ok, err := tx.Order1.Valid()
	if !ok {
		return tx, errors.Wrap(err, "invalid buy order")
	}
	ok, err = tx.Order2.Valid()
	if !ok {
		return tx, errors.Wrap(err, "invalid sell order")
	}
	if tx.Order1.OrderType != Buy {
		return tx, errors.New("incorrect order type of buy order")
	}
	if tx.Order2.OrderType != Sell {
		return tx, errors.New("incorrect order type of sell order")
	}
	if tx.Order2.MatcherPK != tx.Order1.MatcherPK {
		return tx, errors.New("unmatched matcher's public keys")
	}
	if tx.Order2.AssetPair != tx.Order1.AssetPair {
		return tx, errors.New("different asset pairs")
	}
	if tx.Amount == 0 {
		return tx, errors.New("amount should be positive")
	}
	if !validJVMLong(tx.Amount) {
		return tx, errors.New("amount is too big")
	}
	if tx.Price == 0 {
		return tx, errors.New("price should be positive")
	}
	if !validJVMLong(tx.Price) {
		return tx, errors.New("price is too big")
	}
	if tx.Price > tx.Order1.Price || tx.Price < tx.Order2.Price {
		if tx.Price > tx.Order1.Price {
			return tx, errors.Errorf("invalid price: tx.Price %d > tx.Order1.Price %d", tx.Price, tx.Order1.Price)
		}
		if tx.Price < tx.Order2.Price {
			return tx, errors.Errorf("invalid price: tx.Price %d < tx.Order2.Price %d", tx.Price, tx.Order2.Price)
		}
		panic("unreachable")
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	if !validJVMLong(tx.BuyMatcherFee) {
		return tx, errors.New("buy matcher's fee is too big")
	}
	if !validJVMLong(tx.SellMatcherFee) {
		return tx, errors.New("sell matcher's fee is too big")
	}
	if tx.Order1.Expiration < tx.Timestamp {
		return tx, errors.New("invalid buy order expiration")
	}
	if tx.Order1.Expiration-tx.Timestamp > MaxOrderTTL {
		return tx, errors.New("buy order expiration should be earlier than 30 days")
	}
	if tx.Order2.Expiration < tx.Timestamp {
		return tx, errors.New("invalid sell order expiration")
	}
	if tx.Order2.Expiration-tx.Timestamp > MaxOrderTTL {
		return tx, errors.New("sell order expiration should be earlier than 30 days")
	}
	return tx, nil
}

func (tx *ExchangeWithSig) BodyMarshalBinary(Scheme) ([]byte, error) {
	bob, err := tx.Order1.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal ExchangeWithSig body to bytes")
	}
	bol := uint32(len(bob))
	sob, err := tx.Order2.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal ExchangeWithSig body to bytes")
	}
	sol := uint32(len(sob))
	var p uint32
	buf := make([]byte, exchangeWithSigFixedBodyLen+bol+sol)
	buf[0] = byte(tx.Type)
	p++
	binary.BigEndian.PutUint32(buf[p:], bol)
	p += 4
	binary.BigEndian.PutUint32(buf[p:], sol)
	p += 4
	copy(buf[p:], bob)
	p += bol
	copy(buf[p:], sob)
	p += sol
	binary.BigEndian.PutUint64(buf[p:], tx.Price)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.BuyMatcherFee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.SellMatcherFee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func (tx *ExchangeWithSig) BodySerialize(s *serializer.Serializer) error {
	err := s.Byte(byte(tx.Type))
	if err != nil {
		return err
	}
	bob := bytebufferpool.Get()
	defer bytebufferpool.Put(bob)
	s1 := serializer.New(bob)
	err = tx.Order1.Serialize(s1)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal ExchangeWithSig body to bytes")
	}
	bol := uint32(len(bob.B))

	sob := bytebufferpool.Get()
	defer bytebufferpool.Put(sob)
	s2 := serializer.New(bob)
	err = tx.Order2.Serialize(s2)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal ExchangeWithSig body to bytes")
	}
	sol := uint32(len(sob.B))
	err = s.Uint32(bol)
	if err != nil {
		return err
	}
	err = s.Uint32(sol)
	if err != nil {
		return err
	}
	err = s.Bytes(bob.B)
	if err != nil {
		return err
	}
	err = s.Bytes(sob.B)
	if err != nil {
		return err
	}
	err = s.Uint64(tx.Price)
	if err != nil {
		return err
	}
	err = s.Uint64(tx.Amount)
	if err != nil {
		return err
	}
	err = s.Uint64(tx.BuyMatcherFee)
	if err != nil {
		return err
	}
	err = s.Uint64(tx.SellMatcherFee)
	if err != nil {
		return err
	}
	err = s.Uint64(tx.Fee)
	if err != nil {
		return err
	}
	return s.Uint64(tx.Timestamp)
}

func (tx *ExchangeWithSig) bodyUnmarshalBinary(data []byte) (int, error) {
	const expectedLen = exchangeWithSigFixedBodyLen + orderV1MinLen + orderV1MinLen
	if l := len(data); l < expectedLen {
		return 0, errors.Errorf("not enough data for ExchangeWithSig transaction, expected not less then %d, received %d", expectedLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != ExchangeTransaction {
		return 0, errors.Errorf("unexpected transaction type %d for ExchangeWithSig transaction", tx.Type)
	}
	tx.Version = 1
	n := 1
	bol := binary.BigEndian.Uint32(data[n:])
	n += 4
	sol := binary.BigEndian.Uint32(data[n:])
	n += 4
	var bo OrderV1
	err := bo.UnmarshalBinary(data[n:])
	if err != nil {
		return 0, errors.Wrapf(err, "failed to unmarshal ExchangeWithSig body from bytes")
	}
	tx.Order1 = &bo
	n += int(bol)
	var so OrderV1
	err = so.UnmarshalBinary(data[n:])
	if err != nil {
		return 0, errors.Wrapf(err, "failed to unmarshal ExchangeWithSig body from bytes")
	}
	tx.Order2 = &so
	n += int(sol)
	tx.Price = binary.BigEndian.Uint64(data[n:])
	n += 8
	tx.Amount = binary.BigEndian.Uint64(data[n:])
	n += 8
	tx.BuyMatcherFee = binary.BigEndian.Uint64(data[n:])
	n += 8
	tx.SellMatcherFee = binary.BigEndian.Uint64(data[n:])
	n += 8
	tx.Fee = binary.BigEndian.Uint64(data[n:])
	n += 8
	tx.Timestamp = binary.BigEndian.Uint64(data[n:])
	n += 8
	tx.SenderPK = tx.Order1.MatcherPK
	return n, nil
}

// Sign calculates ID and Signature of the transaction
func (tx *ExchangeWithSig) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeWithSig transaction")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "crypto.Sign() failed")
	}
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeWithSig transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that signature of the transaction is valid.
func (tx *ExchangeWithSig) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of ExchangeWithSig transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

// MarshalBinary saves the transaction to its binary representation.
func (tx *ExchangeWithSig) MarshalBinary(scheme Scheme) ([]byte, error) {
	b, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ExchangeWithSig transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], tx.Signature[:])
	return buf, nil
}

// UnmarshalBinary loads the transaction from its binary representation.
func (tx *ExchangeWithSig) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < exchangeWithSigMinLen {
		return errors.Errorf("not enough data for ExchangeWithSig transaction, expected not less then %d, received %d", exchangeWithSigMinLen, l)
	}
	if data[0] != byte(ExchangeTransaction) {
		return errors.Errorf("incorrect transaction type %d for ExchangeWithSig transaction", data[0])
	}
	bl, err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeWithSig transaction from bytes")
	}
	data = data[bl:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *ExchangeWithSig) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *ExchangeWithSig) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	exchangeTx, ok := t.(*ExchangeWithSig)
	if !ok {
		return errors.New("failed to convert result to ExchangeWithSig")
	}
	*tx = *exchangeTx
	return nil
}

func (tx *ExchangeWithSig) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *ExchangeWithSig) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	exchangeTx, ok := t.(*ExchangeWithSig)
	if !ok {
		return errors.New("failed to convert result to ExchangeWithSig")
	}
	*tx = *exchangeTx
	return nil
}

func (tx *ExchangeWithSig) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	orders := make([]*g.Order, 2)
	orders[0] = tx.Order1.ToProtobufSigned(scheme)
	orders[1] = tx.Order2.ToProtobufSigned(scheme)
	txData := &g.Transaction_Exchange{Exchange: &g.ExchangeTransactionData{
		Amount:         int64(tx.Amount),
		Price:          int64(tx.Price),
		BuyMatcherFee:  int64(tx.BuyMatcherFee),
		SellMatcherFee: int64(tx.SellMatcherFee),
		Orders:         orders,
	}}
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *ExchangeWithSig) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Signature == nil {
		return nil, errors.New("no signature provided")
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      proofs.Bytes(),
	}, nil
}

// LeaseWithSig is a transaction that allows to lease Waves to other account.
type LeaseWithSig struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Lease
}

func (tx *LeaseWithSig) Validate(scheme Scheme) (Transaction, error) {
	if tx.Version != 1 {
		return tx, errors.Errorf("unexpected version %d for LeaseWithSig", tx.Version)
	}
	ok, err := tx.Lease.Valid(scheme)
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx LeaseWithSig) BinarySize() int {
	return 1 + crypto.SignatureSize + tx.Lease.BinarySize()
}

func (tx LeaseWithSig) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx LeaseWithSig) GetVersion() byte {
	return tx.Version
}

func (tx *LeaseWithSig) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}
	return nil
}

func (tx *LeaseWithSig) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *LeaseWithSig) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *LeaseWithSig) Clone() *LeaseWithSig {
	out := &LeaseWithSig{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedLeaseWithSig creates new LeaseWithSig transaction without signature and ID set.
func NewUnsignedLeaseWithSig(senderPK crypto.PublicKey, recipient Recipient, amount, fee, timestamp uint64) *LeaseWithSig {
	l := Lease{
		SenderPK:  senderPK,
		Recipient: recipient,
		Amount:    amount,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &LeaseWithSig{Type: LeaseTransaction, Version: 1, Lease: l}
}

func (tx *LeaseWithSig) BodyMarshalBinary(Scheme) ([]byte, error) {
	rl := tx.Recipient.BinarySize()
	buf := make([]byte, leaseWithSigBodyLen+rl)
	buf[0] = byte(tx.Type)
	b, err := tx.Lease.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseWithSig transaction to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *LeaseWithSig) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseWithSigBodyLen {
		return errors.Errorf("not enough data for LeaseWithSig transaction, expected not less then %d, received %d", leaseWithSigBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != LeaseTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseWithSig transaction", tx.Type)
	}
	tx.Version = 1
	var l Lease
	err := l.UnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseWithSig transaction from bytes")
	}
	tx.Lease = l
	return nil
}

// Sign calculates ID and Signature of the transaction.
func (tx *LeaseWithSig) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseWithSig transaction")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithSig transaction")
	}
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseWithSig transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that the signature of the transaction is valid for the given public key.
func (tx *LeaseWithSig) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of LeaseWithSig transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

// MarshalBinary saves the transaction to its binary representation.
func (tx *LeaseWithSig) MarshalBinary(scheme Scheme) ([]byte, error) {
	b, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseWithSig transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], tx.Signature[:])
	return buf, nil
}

// UnmarshalBinary reads the transaction from bytes slice.
func (tx *LeaseWithSig) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < leaseWithSigMinLen {
		return errors.Errorf("not enough data for LeaseWithSig transaction, expected not less then %d, received %d", leaseWithSigMinLen, l)
	}
	if data[0] != byte(LeaseTransaction) {
		return errors.Errorf("incorrect transaction type %d for LeaseWithSig transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseWithSig transaction from bytes")
	}
	bl := leaseWithSigBodyLen + tx.Recipient.BinarySize()
	data = data[bl:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *LeaseWithSig) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *LeaseWithSig) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	leaseTx, ok := t.(*LeaseWithSig)
	if !ok {
		return errors.New("failed to convert result to LeaseWithSig")
	}
	*tx = *leaseTx
	return nil
}

func (tx *LeaseWithSig) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *LeaseWithSig) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	leaseTx, ok := t.(*LeaseWithSig)
	if !ok {
		return errors.New("failed to convert result to LeaseWithSig")
	}
	*tx = *leaseTx
	return nil
}

func (tx *LeaseWithSig) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData, err := tx.Lease.ToProtobuf()
	if err != nil {
		return nil, err
	}
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *LeaseWithSig) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Signature == nil {
		return nil, errors.New("no signature provided")
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      proofs.Bytes(),
	}, nil
}

// LeaseCancelWithSig transaction can be used to cancel previously created leasing.
type LeaseCancelWithSig struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	LeaseCancel
}

func (tx *LeaseCancelWithSig) Validate(_ Scheme) (Transaction, error) {
	if tx.Version != 1 {
		return tx, errors.Errorf("unexpected version %d for LeaseCancelWithSig", tx.Version)
	}
	ok, err := tx.LeaseCancel.Valid()
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx LeaseCancelWithSig) BinarySize() int {
	return 1 + crypto.SignatureSize + tx.LeaseCancel.BinarySize()
}

func (tx LeaseCancelWithSig) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx LeaseCancelWithSig) GetVersion() byte {
	return tx.Version
}

func (tx *LeaseCancelWithSig) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}
	return nil
}

func (tx *LeaseCancelWithSig) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *LeaseCancelWithSig) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *LeaseCancelWithSig) Clone() *LeaseCancelWithSig {
	out := &LeaseCancelWithSig{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedLeaseCancelWithSig creates new LeaseCancelWithSig transaction structure without a signature and an ID.
func NewUnsignedLeaseCancelWithSig(senderPK crypto.PublicKey, leaseID crypto.Digest, fee, timestamp uint64) *LeaseCancelWithSig {
	lc := LeaseCancel{
		SenderPK:  senderPK,
		LeaseID:   leaseID,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &LeaseCancelWithSig{Type: LeaseCancelTransaction, Version: 1, LeaseCancel: lc}
}

func (tx *LeaseCancelWithSig) BodyMarshalBinary(Scheme) ([]byte, error) {
	buf := make([]byte, leaseCancelWithSigBodyLen)
	buf[0] = byte(tx.Type)
	b, err := tx.LeaseCancel.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelWithSig to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *LeaseCancelWithSig) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseCancelWithSigBodyLen {
		return errors.Errorf("not enough data for LeaseCancelWithSig transaction, expected not less then %d, received %d", leaseCancelWithSigBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != LeaseCancelTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseCancelWithSig transaction", tx.Type)

	}
	tx.Version = 1
	var lc LeaseCancel
	err := lc.UnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelWithSig from bytes")
	}
	tx.LeaseCancel = lc
	return nil
}

func (tx *LeaseCancelWithSig) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithSig transaction")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithSig transaction")
	}
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithSig transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that signature of the transaction is valid for the given public key.
func (tx *LeaseCancelWithSig) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of LeaseCancelWithSig transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

// MarshalBinary saves transaction to its binary representation.
func (tx *LeaseCancelWithSig) MarshalBinary(scheme Scheme) ([]byte, error) {
	b, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelWithSig transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], tx.Signature[:])
	return buf, nil
}

func (tx *LeaseCancelWithSig) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < leaseCancelWithSigMinLen {
		return errors.Errorf("not enough data for LeaseCancelWithSig transaction, expected not less then %d, received %d", leaseCancelWithSigMinLen, l)
	}
	if data[0] != byte(LeaseCancelTransaction) {
		return errors.Errorf("incorrect transaction type %d for LeaseCancelWithSig transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelWithSig transaction from bytes")
	}
	data = data[leaseCancelWithSigBodyLen:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *LeaseCancelWithSig) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *LeaseCancelWithSig) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	leaseCancelTx, ok := t.(*LeaseCancelWithSig)
	if !ok {
		return errors.New("failed to convert result to LeaseCancelWithSig")
	}
	*tx = *leaseCancelTx
	return nil
}

func (tx *LeaseCancelWithSig) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *LeaseCancelWithSig) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	leaseCancelTx, ok := t.(*LeaseCancelWithSig)
	if !ok {
		return errors.New("failed to convert result to LeaseCancelWithSig")
	}
	*tx = *leaseCancelTx
	return nil
}

func (tx *LeaseCancelWithSig) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.LeaseCancel.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *LeaseCancelWithSig) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Signature == nil {
		return nil, errors.New("no signature provided")
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      proofs.Bytes(),
	}, nil
}

type CreateAliasWithSig struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	CreateAlias
}

func (tx *CreateAliasWithSig) Validate(scheme Scheme) (Transaction, error) {
	if tx.Version != 1 {
		return tx, errors.Errorf("unexpected version %d for CreateAliasWithSig", tx.Version)
	}
	ok, err := tx.CreateAlias.Valid(scheme)
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx CreateAliasWithSig) BinarySize() int {
	return 1 + crypto.SignatureSize + tx.CreateAlias.BinarySize()
}

func (tx CreateAliasWithSig) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx CreateAliasWithSig) GetVersion() byte {
	return tx.Version
}

func (tx *CreateAliasWithSig) GenerateID(scheme Scheme) error {
	if tx.ID != nil {
		return nil
	}
	// user can send tx through HTTP endpoint and Scheme there will be ignored (set to 0)
	// but correct scheme is necessary in Verify method, so that's a crunch for that
	tx.Alias.Scheme = scheme // TODO: create special method for providing scheme value for Tx
	if IsProtobufTx(tx) {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
		return nil
	}
	id, err := tx.CreateAlias.id()
	if err != nil {
		return err
	}
	tx.ID = id
	return nil
}

func (tx *CreateAliasWithSig) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *CreateAliasWithSig) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *CreateAliasWithSig) Clone() *CreateAliasWithSig {
	out := &CreateAliasWithSig{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

func NewUnsignedCreateAliasWithSig(senderPK crypto.PublicKey, alias Alias, fee, timestamp uint64) *CreateAliasWithSig {
	ca := CreateAlias{
		SenderPK:  senderPK,
		Alias:     alias,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &CreateAliasWithSig{Type: CreateAliasTransaction, Version: 1, CreateAlias: ca}
}

func (tx *CreateAliasWithSig) BodyMarshalBinary(Scheme) ([]byte, error) {
	buf := make([]byte, createAliasWithSigFixedBodyLen+len(tx.Alias.Alias))
	buf[0] = byte(tx.Type)
	b, err := tx.CreateAlias.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasWithSig transaction body to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *CreateAliasWithSig) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < createAliasWithSigFixedBodyLen {
		return errors.Errorf("not enough data for CreateAliasWithSig transaction, expected not less then %d, received %d", createAliasWithSigFixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != CreateAliasTransaction {
		return errors.Errorf("unexpected transaction type %d for CreateAliasWithSig transaction", tx.Type)
	}
	tx.Version = 1
	var ca CreateAlias
	err := ca.UnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasWithSig transaction from bytes")
	}
	tx.CreateAlias = ca
	return nil
}

func (tx *CreateAliasWithSig) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasWithSig transaction")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasWithSig transaction")
	}
	tx.Signature = &s
	return nil
}

func (tx *CreateAliasWithSig) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of CreateAliasWithSig transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *CreateAliasWithSig) MarshalBinary(scheme Scheme) ([]byte, error) {
	b, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasWithSig transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], tx.Signature[:])
	return buf, nil
}

func (tx *CreateAliasWithSig) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < createAliasWithSigMinLen {
		return errors.Errorf("not enough data for CreateAliasWithSig transaction, expected not less then %d, received %d", createAliasWithSigMinLen, l)
	}
	if data[0] != byte(CreateAliasTransaction) {
		return errors.Errorf("incorrect transaction type %d for CreateAliasWithSig transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasWithSig transaction from bytes")
	}
	bl := createAliasWithSigFixedBodyLen + len(tx.Alias.Alias)
	data = data[bl:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

// Deprecated: use UnmarshalJSONWithScheme.
func (tx *CreateAliasWithSig) UnmarshalJSON(data []byte) error {
	const ignoreChainID Scheme = 0
	return tx.UnmarshalJSONWithScheme(data, ignoreChainID)
}

func (tx *CreateAliasWithSig) UnmarshalJSONWithScheme(data []byte, scheme Scheme) error {
	tmp := struct {
		Type      TransactionType   `json:"type"`
		Version   byte              `json:"version,omitempty"`
		Signature *crypto.Signature `json:"signature,omitempty"`
		SenderPK  crypto.PublicKey  `json:"senderPublicKey"`
		Alias     string            `json:"alias"`
		Fee       uint64            `json:"fee"`
		Timestamp uint64            `json:"timestamp,omitempty"`
	}{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasWithSig from JSON")
	}
	tx.Type = tmp.Type
	tx.Version = tmp.Version
	tx.Signature = tmp.Signature
	tx.SenderPK = tmp.SenderPK
	tx.Alias = *NewAlias(scheme, tmp.Alias)
	tx.Fee = tmp.Fee
	tx.Timestamp = tmp.Timestamp
	return nil
}

func (tx *CreateAliasWithSig) MarshalJSON() ([]byte, error) {
	type shadowed CreateAliasWithSig
	tmp := struct {
		Alias string `json:"alias"`
		*shadowed
	}{tx.Alias.Alias, (*shadowed)(tx)}
	return json.Marshal(tmp)
}

func (tx *CreateAliasWithSig) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *CreateAliasWithSig) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	createAliasTx, ok := t.(*CreateAliasWithSig)
	if !ok {
		return errors.New("failed to convert result to CreateAliasWithSig")
	}
	*tx = *createAliasTx
	return nil
}

func (tx *CreateAliasWithSig) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *CreateAliasWithSig) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	createAliasTx, ok := t.(*CreateAliasWithSig)
	if !ok {
		return errors.New("failed to convert result to CreateAliasWithSig")
	}
	*tx = *createAliasTx
	return nil
}

func (tx *CreateAliasWithSig) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.CreateAlias.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *CreateAliasWithSig) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Signature == nil {
		return nil, errors.New("no signature provided")
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      proofs.Bytes(),
	}, nil
}
