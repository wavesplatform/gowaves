package proto

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

const (
	issueWithProofsFixedBodyLen                          = 1 + 1 + 1 + crypto.PublicKeySize + 2 + 2 + 8 + 1 + 1 + 8 + 8 + 1
	issueWithProofsMinBodyLen                            = issueWithProofsFixedBodyLen + 4 // 4 because of the shortest allowed Asset name of 4 bytes
	issueWithProofsMinLen                                = 1 + issueWithProofsMinBodyLen + proofsMinLen
	transferWithProofsFixedBodyLen                       = 1 + 1 + transferLen
	transferWithProofsMinLen                             = 1 + transferWithProofsFixedBodyLen + proofsMinLen
	reissueWithProofsBodyLen                             = 3 + reissueLen
	reissueWithProofsMinLen                              = 1 + reissueWithProofsBodyLen + proofsMinLen
	burnWithProofsBodyLen                                = 1 + 1 + 1 + burnLen
	burnWithProofsLen                                    = 1 + burnWithProofsBodyLen + proofsMinLen
	exchangeWithProofsFixedBodyLen                       = 1 + 1 + 1 + 4 + 4 + 8 + 8 + 8 + 8 + 8 + 8
	exchangeWithProofsMinLen                             = exchangeWithProofsFixedBodyLen + orderV2MinLen + orderV2MinLen + proofsMinLen
	leaseWithProofsBodyLen                               = 1 + 1 + 1 + leaseLen
	leaseWithProofsMinLen                                = leaseWithProofsBodyLen + proofsMinLen
	leaseCancelWithProofsBodyLen                         = 1 + 1 + 1 + leaseCancelLen
	leaseCancelWithProofsMinLen                          = 1 + leaseCancelWithProofsBodyLen + proofsMinLen
	createAliasWithProofsFixedBodyLen                    = 1 + 1 + createAliasLen
	createAliasWithProofsMinLen                          = 1 + createAliasWithProofsFixedBodyLen + proofsMinLen
	massTransferEntryLen                                 = 8
	massTransferWithProofsFixedLen                       = 1 + 1 + crypto.PublicKeySize + 1 + 2 + 8 + 8 + 2
	massTransferWithProofsMinLen                         = massTransferWithProofsFixedLen + proofsMinLen
	dataWithProofsFixedBodyLen                           = 1 + 1 + crypto.PublicKeySize + 2 + 8 + 8
	dataWithProofsMinLen                                 = dataWithProofsFixedBodyLen + proofsMinLen
	setScriptWithProofsFixedBodyLen                      = 1 + 1 + 1 + crypto.PublicKeySize + 1 + 8 + 8
	setScriptWithProofsMinLen                            = 1 + setScriptWithProofsFixedBodyLen + proofsMinLen
	sponsorshipWithProofsBodyLen                         = 1 + 1 + crypto.PublicKeySize + crypto.DigestSize + 8 + 8 + 8
	sponsorshipWithProofsMinLen                          = 1 + 1 + 1 + sponsorshipWithProofsBodyLen + proofsMinLen
	setAssetScriptWithProofsFixedBodyLen                 = 1 + 1 + 1 + crypto.PublicKeySize + crypto.DigestSize + 8 + 8 + 1
	setAssetScriptWithProofsMinLen                       = 1 + setScriptWithProofsFixedBodyLen + proofsMinLen
	invokeScriptWithProofsFixedBodyLen                   = 1 + 1 + 1 + crypto.PublicKeySize + 8 + 8
	invokeScriptWithProofsMinLen                         = 1 + invokeScriptWithProofsFixedBodyLen + proofsMinLen
	maxTransfers                                         = 100
	maxEntries                                           = 100
	maxDataWithProofsTxBytes                         int = 1.2 * MaxDataWithProofsBytes // according to the scala's node realization
	maxArguments                                         = 22
	maxFunctionNameBytes                                 = 255
	maxInvokeScriptWithProofsBinaryTransactionsBytes     = 5 * 1024
	maxInvokeScriptWithProofsProtobufPayloadBytes        = 5 * 1024
)

// IssueWithProofs is a transaction to issue new asset, second version.
type IssueWithProofs struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Script  Script          `json:"script"`
	Issue
}

func (tx IssueWithProofs) BinarySize() int {
	scriptSize := 1
	if len(tx.Script) > 0 {
		scriptSize += 2 + len(tx.Script)
	}
	return 4 + tx.Proofs.BinarySize() + scriptSize + tx.Issue.BinarySize()
}

func (tx *IssueWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *IssueWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	issueTx, ok := t.(*IssueWithProofs)
	if !ok {
		return errors.New("failed to convert result to IssueWithProofs")
	}
	*tx = *issueTx
	return nil
}

func (tx *IssueWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *IssueWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	issueTx, ok := t.(*IssueWithProofs)
	if !ok {
		return errors.New("failed to convert result to IssueWithProofs")
	}
	*tx = *issueTx
	return nil
}

func (tx *IssueWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.Issue.ToProtobuf()
	txData.Issue.Script = tx.Script
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *IssueWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx IssueWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx IssueWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *IssueWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *IssueWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *IssueWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *IssueWithProofs) Clone() *IssueWithProofs {
	out := &IssueWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedIssueWithProofs creates a new IssueWithProofs transaction with empty Proofs.
func NewUnsignedIssueWithProofs(v byte, senderPK crypto.PublicKey, name, description string, quantity uint64, decimals byte, reissuable bool, script []byte, timestamp, fee uint64) *IssueWithProofs {
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
	return &IssueWithProofs{Type: IssueTransaction, Version: v, Script: script, Issue: i}
}

func (tx *IssueWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 2 || tx.Version > MaxIssueTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for IssueWithProofs", tx.Version)
	}
	ok, err := tx.Issue.Valid()
	if !ok {
		return tx, err
	}
	if tx.NonEmptyScript() {
		if err := serialization.CheckHeader(tx.Script); err != nil {
			return tx, err
		}
	}
	// we don't need to validate scheme here because scheme is included in binary representation of tx
	// so if scheme is invalid == signature is invalid
	return tx, nil
}

// NonEmptyScript returns true if the script of the transaction is not empty, otherwise false.
func (tx *IssueWithProofs) NonEmptyScript() bool {
	return len(tx.Script) != 0
}

func (tx *IssueWithProofs) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	var p int
	nl := len(tx.Name)
	dl := len(tx.Description)
	sl := len(tx.Script)
	if sl > 0 {
		sl += 2
	}
	buf := make([]byte, issueWithProofsFixedBodyLen+nl+dl+sl)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = scheme
	p = 3
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	PutStringWithUInt16Len(buf[p:], tx.Name)
	p += 2 + nl
	PutStringWithUInt16Len(buf[p:], tx.Description)
	p += 2 + dl
	binary.BigEndian.PutUint64(buf[p:], tx.Quantity)
	p += 8
	buf[p] = tx.Decimals
	p++
	PutBool(buf[p:], tx.Reissuable)
	p++
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	PutBool(buf[p:], tx.NonEmptyScript())
	p++
	if tx.NonEmptyScript() {
		if err := PutBytesWithUInt16Len(buf[p:], tx.Script); err != nil {
			return nil, errors.Wrap(err, "failed to marshal body of IssueWithProofs transaction")
		}
	}
	return buf, nil
}

func (tx *IssueWithProofs) bodyUnmarshalBinary(data []byte, scheme Scheme) error {
	const message = "failed to unmarshal field %q of IssueWithProofs transaction"
	if l := len(data); l < issueWithProofsMinBodyLen {
		return errors.Errorf("not enough data for IssueWithProofs transaction %d, expected not less then %d", l, issueWithProofsMinBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != IssueTransaction {
		return errors.Errorf("unexpected transaction type %d for IssueWithProofs transaction", tx.Type)
	}
	tx.Version = data[1]
	if unmarshalledScheme := data[2]; unmarshalledScheme != scheme {
		return errors.Errorf("scheme mismatch: got %d, want %d", unmarshalledScheme, scheme)
	}
	data = data[3:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	tx.Name, err = StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrapf(err, message, "AppName")
	}
	data = data[2+len(tx.Name):]
	tx.Description, err = StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrapf(err, message, "Description")
	}
	data = data[2+len(tx.Description):]
	tx.Quantity = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Decimals = data[0]
	data = data[1:]
	tx.Reissuable, err = Bool(data)
	if err != nil {
		return errors.Wrapf(err, message, "Reissuable")
	}
	data = data[1:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	p, err := Bool(data)
	if err != nil {
		return errors.Wrapf(err, message, "Script")
	}
	data = data[1:]
	if p {
		s, err := BytesWithUInt16Len(data)
		if err != nil {
			return errors.Wrapf(err, message, "Script")
		}
		tx.Script = s
	}
	return nil
}

// Sign calculates transaction signature using given secret key.
func (tx *IssueWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueWithProofs transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that the transaction signature is valid for given public key.
func (tx *IssueWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of IssueWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary converts transaction to its binary representation.
func (tx *IssueWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal IssueWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal IssueWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal IssueWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	var p int
	buf[p] = 0
	p++
	copy(buf[p:], bb)
	p += bl
	copy(buf[p:], pb)
	return buf, nil
}

// UnmarshalBinary reads transaction from its binary representation.
func (tx *IssueWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < issueWithProofsMinLen {
		return errors.Errorf("not enough data for IssueWithProofs transaction, expected not less then %d, received %d", issueWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d for IssueWithProofs transaction, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data, scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IssueWithProofs transaction")
	}
	sl := len(tx.Script)
	if sl > 0 {
		sl += 2
	}
	bl := issueWithProofsFixedBodyLen + len(tx.Name) + len(tx.Description) + sl
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IssueWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

// TransferWithProofs transaction to transfer any token from one account to another. Version 2.
type TransferWithProofs struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Transfer
}

func (tx TransferWithProofs) BinarySize() int {
	return 3 + tx.Proofs.BinarySize() + tx.Transfer.BinarySize()
}

func (tx TransferWithProofs) GetProofs() *ProofsV1 {
	return tx.Proofs
}

func (tx *TransferWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *TransferWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	transferTx, ok := t.(*TransferWithProofs)
	if !ok {
		return errors.New("failed to convert result to TransferWithProofs")
	}
	*tx = *transferTx
	return nil
}

func (tx *TransferWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *TransferWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	transferTx, ok := t.(*TransferWithProofs)
	if !ok {
		return errors.New("failed to convert result to TransferWithProofs")
	}
	*tx = *transferTx
	return nil
}

func (tx *TransferWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
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

func (tx *TransferWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx TransferWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx TransferWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *TransferWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *TransferWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *TransferWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *TransferWithProofs) Clone() *TransferWithProofs {
	out := &TransferWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedTransferWithProofs creates new TransferWithProofs transaction without proofs and ID.
func NewUnsignedTransferWithProofs(v byte, senderPK crypto.PublicKey, amountAsset, feeAsset OptionalAsset, timestamp, amount, fee uint64, recipient Recipient, attachment Attachment) *TransferWithProofs {
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
	return &TransferWithProofs{Type: TransferTransaction, Version: v, Transfer: t}
}

func (tx *TransferWithProofs) Validate(scheme Scheme) (Transaction, error) {
	if tx.Version < 2 || tx.Version > MaxTransferTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for TransferWithProofs", tx.Version)
	}
	ok, err := tx.Transfer.Valid(scheme)
	if !ok {
		return tx, err
	}
	//TODO: validate script
	return tx, nil
}

func (tx *TransferWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	b, err := tx.Transfer.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferWithProofs body")
	}
	buf := make([]byte, 2+len(b))
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	copy(buf[2:], b)
	return buf, nil
}

func (tx *TransferWithProofs) BodySerialize(s *serializer.Serializer) error {
	buf := [2]byte{byte(tx.Type), tx.Version}
	err := s.Bytes(buf[:])
	if err != nil {
		return err
	}
	err = tx.Transfer.Serialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal TransferWithProofs body")
	}
	return nil
}

func (tx *TransferWithProofs) BodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < transferWithProofsFixedBodyLen {
		return errors.Errorf("%d bytes is not enough for TransferWithProofs transaction, expected not less then %d bytes", l, transferWithProofsFixedBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != TransferTransaction {
		return errors.Errorf("unexpected transaction type %d for TransferWithProofs transaction", tx.Type)
	}
	tx.Version = data[1]
	if v := tx.Version; v < 2 {
		return errors.Errorf("unexpected version %d for TransferWithProofs transaction", v)
	}
	var t Transfer
	err := t.UnmarshalBinary(data[2:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferWithProofs body from bytes")
	}
	tx.Transfer = t
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *TransferWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferWithProofs transaction")
	}
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *TransferWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of TransferWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary writes TransferWithProofs transaction to its byte representation.
func (tx *TransferWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal TransferWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

func (tx *TransferWithProofs) Serialize(s *serializer.Serializer) error {
	err := s.Byte(0)
	if err != nil {
		return err
	}
	err = tx.BodySerialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal TransferWithProofs transaction to bytes")
	}
	if tx.Proofs == nil {
		return errors.New("failed to marshal TransferWithProofs transaction to bytes: no proofs")
	}
	err = tx.Proofs.Serialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal TransferWithProofs transaction to bytes")
	}
	return nil
}

// UnmarshalBinary reads TransferWithProofs from its byte representation.
func (tx *TransferWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < transferWithProofsMinLen {
		return errors.Errorf("not enough data for TransferWithProofs transaction, expected not less then %d, received %d", transferWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.BodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferWithProofs transaction from bytes")
	}
	aal := 0
	if tx.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	fal := 0
	if tx.FeeAsset.Present {
		fal += crypto.DigestSize
	}
	atl := tx.attachmentSize()
	rl := tx.Recipient.BinarySize()
	bl := transferWithProofsFixedBodyLen + aal + fal + atl + rl
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *TransferWithProofs) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Type    TransactionType `json:"type"`
		Version byte            `json:"version,omitempty"`
		ID      *crypto.Digest  `json:"id,omitempty"`
		Proofs  *ProofsV1       `json:"proofs,omitempty"`
		Transfer
	}{}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	tx.Type = tmp.Type
	tx.Version = tmp.Version
	tx.ID = tmp.ID
	tx.Proofs = tmp.Proofs
	tx.Transfer = tmp.Transfer
	return nil
}

// ReissueWithProofs same as ReissueWithSig but version 2 with Proofs.
type ReissueWithProofs struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Reissue
}

func (tx ReissueWithProofs) BinarySize() int {
	return 4 + tx.Proofs.BinarySize() + tx.Reissue.BinarySize()
}

func (tx *ReissueWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *ReissueWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	reissueTx, ok := t.(*ReissueWithProofs)
	if !ok {
		return errors.New("failed to convert result to ReissueWithProofs")
	}
	*tx = *reissueTx
	return nil
}

func (tx *ReissueWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *ReissueWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	reissueTx, ok := t.(*ReissueWithProofs)
	if !ok {
		return errors.New("failed to convert result to ReissueWithProofs")
	}
	*tx = *reissueTx
	return nil
}

func (tx *ReissueWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.Reissue.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *ReissueWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx ReissueWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx ReissueWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *ReissueWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *ReissueWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *ReissueWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *ReissueWithProofs) Clone() *ReissueWithProofs {
	out := &ReissueWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedReissueWithProofs creates new ReissueWithProofs transaction without signature and ID.
func NewUnsignedReissueWithProofs(v byte, senderPK crypto.PublicKey, assetID crypto.Digest, quantity uint64, reissuable bool, timestamp, fee uint64) *ReissueWithProofs {
	r := Reissue{
		SenderPK:   senderPK,
		AssetID:    assetID,
		Quantity:   quantity,
		Reissuable: reissuable,
		Fee:        fee,
		Timestamp:  timestamp,
	}
	return &ReissueWithProofs{Type: ReissueTransaction, Version: v, Reissue: r}
}

func (tx *ReissueWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 2 || tx.Version > MaxReissueTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for ReissueWithProofs", tx.Version)
	}
	ok, err := tx.Reissue.Valid()
	if !ok {
		return tx, err
	}
	//TODO: add current blockchain scheme validation
	return tx, nil
}

func (tx *ReissueWithProofs) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	buf := make([]byte, reissueWithProofsBodyLen)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = scheme
	b, err := tx.Reissue.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueWithProofs body")
	}
	copy(buf[3:], b)
	return buf, nil
}

func (tx *ReissueWithProofs) bodyUnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < reissueWithProofsBodyLen {
		return errors.Errorf("%d bytes is not enough for ReissueWithProofs transaction, expected not less then %d bytes", l, reissueWithProofsBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != ReissueTransaction {
		return errors.Errorf("unexpected transaction type %d for ReissueWithProofs transaction", tx.Type)
	}
	tx.Version = data[1]
	if v := tx.Version; v < 2 {
		return errors.Errorf("unexpected version %d for ReissueWithProofs transaction", v)
	}
	if unmarshalledScheme := data[2]; unmarshalledScheme != scheme {
		return errors.Errorf("scheme mismatch: got %d, want %d", unmarshalledScheme, scheme)
	}
	var r Reissue
	err := r.UnmarshalBinary(data[3:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueWithProofs body from bytes")
	}
	tx.Reissue = r
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *ReissueWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueWithProofs transaction")
	}
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *ReissueWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of ReissueWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary writes ReissueWithProofs transaction to its byte representation.
func (tx *ReissueWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal ReissueWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads ReissueWithProofs from its byte representation.
func (tx *ReissueWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < reissueWithProofsMinLen {
		return errors.Errorf("not enough data for ReissueWithProofs transaction, expected not less then %d, received %d", reissueWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data, scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueWithProofs transaction from bytes")
	}
	data = data[reissueWithProofsBodyLen:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

// BurnWithProofs same as BurnWithSig but version 2 with Proofs.
type BurnWithProofs struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Burn
}

func (tx BurnWithProofs) BinarySize() int {
	return 4 + tx.Proofs.BinarySize() + tx.Burn.BinarySize()
}

func (tx *BurnWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *BurnWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	burnTx, ok := t.(*BurnWithProofs)
	if !ok {
		return errors.New("failed to convert result to BurnWithProofs")
	}
	*tx = *burnTx
	return nil
}

func (tx *BurnWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *BurnWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	burnTx, ok := t.(*BurnWithProofs)
	if !ok {
		return errors.New("failed to convert result to BurnWithProofs")
	}
	*tx = *burnTx
	return nil
}

func (tx *BurnWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.Burn.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *BurnWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx BurnWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx BurnWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *BurnWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *BurnWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *BurnWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *BurnWithProofs) Clone() *BurnWithProofs {
	out := &BurnWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedBurnWithProofs creates new BurnWithProofs transaction without proofs and ID.
func NewUnsignedBurnWithProofs(v byte, senderPK crypto.PublicKey, assetID crypto.Digest, amount, timestamp, fee uint64) *BurnWithProofs {
	b := Burn{
		SenderPK:  senderPK,
		AssetID:   assetID,
		Amount:    amount,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &BurnWithProofs{Type: BurnTransaction, Version: v, Burn: b}
}

func (tx *BurnWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 2 || tx.Version > MaxBurnTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for BurnWithProofs", tx.Version)
	}
	ok, err := tx.Burn.Valid()
	if !ok {
		return tx, err
	}
	// we don't need to validate scheme here because scheme is included in binary representation of tx
	// so if scheme is invalid == signature is invalid
	return tx, nil
}

func (tx *BurnWithProofs) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	buf := make([]byte, burnWithProofsBodyLen)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = scheme
	b, err := tx.Burn.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnWithProofs body")
	}
	copy(buf[3:], b)
	return buf, nil
}

func (tx *BurnWithProofs) bodyUnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < burnWithProofsBodyLen {
		return errors.Errorf("%d bytes is not enough for BurnWithProofs transaction, expected not less then %d bytes", l, burnWithProofsBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != BurnTransaction {
		return errors.Errorf("unexpected transaction type %d for BurnWithProofs transaction", tx.Type)
	}
	tx.Version = data[1]
	if v := tx.Version; v < 2 {
		return errors.Errorf("unexpected version %d for BurnWithProofs transaction", v)
	}
	if unmarshalledScheme := data[2]; unmarshalledScheme != scheme {
		return errors.Errorf("scheme mismatch: got %d, want %d", unmarshalledScheme, scheme)
	}
	var b Burn
	err := b.UnmarshalBinary(data[3:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnWithProofs body from bytes")
	}
	tx.Burn = b
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *BurnWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnWithProofs transaction")
	}
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *BurnWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of BurnWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary writes BurnWithProofs transaction to its byte representation.
func (tx *BurnWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal BurnWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads BurnWithProofs from its byte representation.
func (tx *BurnWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < burnWithProofsLen {
		return errors.Errorf("not enough data for BurnWithProofs transaction, expected not less then %d, received %d", burnWithProofsBodyLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data, scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnWithProofs transaction from bytes")
	}
	data = data[burnWithProofsBodyLen:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

// ExchangeWithProofs is a transaction to store settlement on blockchain.
type ExchangeWithProofs struct {
	Type           TransactionType  `json:"type"`
	Version        byte             `json:"version,omitempty"`
	ID             *crypto.Digest   `json:"id,omitempty"`
	Proofs         *ProofsV1        `json:"proofs,omitempty"`
	SenderPK       crypto.PublicKey `json:"senderPublicKey"`
	Order1         Order            `json:"order1"`
	Order2         Order            `json:"order2"`
	Price          uint64           `json:"price"`
	Amount         uint64           `json:"amount"`
	BuyMatcherFee  uint64           `json:"buyMatcherFee"`
	SellMatcherFee uint64           `json:"sellMatcherFee"`
	Fee            uint64           `json:"fee"`
	Timestamp      uint64           `json:"timestamp,omitempty"`
}

func (tx ExchangeWithProofs) BinarySize() int {
	boSize := 4 + tx.Order1.BinarySize()
	soSize := 4 + tx.Order2.BinarySize()
	if tx.Order1.GetVersion() == 1 {
		boSize += 1
	}
	if tx.Order2.GetVersion() == 1 {
		soSize += 1
	}
	return 3 + tx.Proofs.BinarySize() + 48 + boSize + soSize
}

func (tx *ExchangeWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *ExchangeWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	exchangeTx, ok := t.(*ExchangeWithProofs)
	if !ok {
		return errors.New("failed to convert result to ExchangeWithProofs")
	}
	*tx = *exchangeTx
	return nil
}

func (tx *ExchangeWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *ExchangeWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	exchangeTx, ok := t.(*ExchangeWithProofs)
	if !ok {
		return errors.New("failed to convert result to ExchangeWithProofs")
	}
	*tx = *exchangeTx
	return nil
}

func (tx *ExchangeWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	orders := []*g.Order{
		tx.Order1.ToProtobufSigned(scheme),
		tx.Order2.ToProtobufSigned(scheme),
	}
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

func (tx *ExchangeWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx ExchangeWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx ExchangeWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *ExchangeWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *ExchangeWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *ExchangeWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *ExchangeWithProofs) Clone() *ExchangeWithProofs {
	out := &ExchangeWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

func (tx ExchangeWithProofs) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx ExchangeWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx ExchangeWithProofs) GetBuyOrder() (Order, error) {
	if tx.Order1.GetOrderType() == Buy {
		return tx.Order1, nil
	}
	if tx.Order2.GetOrderType() == Buy {
		return tx.Order2, nil
	}
	return nil, errors.New("no buy order")
}

func (tx ExchangeWithProofs) GetSellOrder() (Order, error) {
	if tx.Order2.GetOrderType() == Sell {
		return tx.Order2, nil
	}
	if tx.Order1.GetOrderType() == Sell {
		return tx.Order1, nil
	}
	return nil, errors.New("no sell order")
}

func (tx ExchangeWithProofs) GetOrder1() Order {
	return tx.Order1
}

func (tx ExchangeWithProofs) GetOrder2() Order {
	return tx.Order2
}

func (tx ExchangeWithProofs) GetPrice() uint64 {
	return tx.Price
}

func (tx ExchangeWithProofs) GetAmount() uint64 {
	return tx.Amount
}

func (tx ExchangeWithProofs) GetBuyMatcherFee() uint64 {
	return tx.BuyMatcherFee
}

func (tx ExchangeWithProofs) GetSellMatcherFee() uint64 {
	return tx.SellMatcherFee
}
func (tx ExchangeWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx ExchangeWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

func NewUnsignedExchangeWithProofs(v byte, buy, sell Order, price, amount, buyMatcherFee, sellMatcherFee, fee, timestamp uint64) *ExchangeWithProofs {
	return &ExchangeWithProofs{
		Type:           ExchangeTransaction,
		Version:        v,
		SenderPK:       buy.GetMatcherPK(),
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

func (tx *ExchangeWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 2 || tx.Version > MaxExchangeTransactionVersion {
		return tx, errors.Errorf("unexpected transaction version %d for ExchangeWithProofs transaction", tx.Version)
	}
	ok, err := tx.Order1.Valid()
	if !ok {
		return tx, errors.Wrap(err, "invalid first order")
	}
	ok, err = tx.Order2.Valid()
	if !ok {
		return tx, errors.Wrap(err, "invalid second order")
	}
	if (tx.Order1.GetOrderType() == Buy && tx.Order2.GetOrderType() != Sell) || (tx.Order1.GetOrderType() == Sell && tx.Order2.GetOrderType() != Buy) {
		return tx, errors.New("incorrect combination of orders types")
	}
	if tx.Order2.GetMatcherPK() != tx.Order1.GetMatcherPK() {
		return tx, errors.New("unmatched matcher's public keys")
	}
	if tx.Order2.GetAssetPair() != tx.Order1.GetAssetPair() {
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
	bo, err := tx.GetBuyOrder()
	if err != nil {
		return tx, err
	}
	so, err := tx.GetSellOrder()
	if err != nil {
		return tx, err
	}
	if tx.Version < 3 && (tx.Price > bo.GetPrice() || tx.Price < so.GetPrice()) {
		if tx.Price > bo.GetPrice() {
			return tx, errors.Errorf("invalid price: tx.Price %d > bo.GetPrice() %d", tx.Price, bo.GetPrice())
		}
		if tx.Price < so.GetPrice() {
			return tx, errors.Errorf("invalid price: tx.Price %d < so.GetPrice() %d", tx.Price, so.GetPrice())
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
	if tx.Order1.GetExpiration() < tx.Timestamp {
		return tx, errors.New("invalid first order expiration")
	}
	if tx.Order1.GetExpiration()-tx.Timestamp > MaxOrderTTL {
		return tx, errors.New("first order expiration should be earlier than 30 days")
	}
	if tx.Order2.GetExpiration() < tx.Timestamp {
		return tx, errors.New("invalid second order expiration")
	}
	if tx.Order2.GetExpiration()-tx.Timestamp > MaxOrderTTL {
		return tx, errors.New("second order expiration should be earlier than 30 days")
	}
	return tx, nil
}

func (tx *ExchangeWithProofs) marshalAsOrderV1(order Order) ([]byte, error) {
	o, ok := order.(*OrderV1)
	if !ok {
		return nil, errors.Errorf("failed to cast an order with version 1 to OrderV1, type %T", order)
	}
	b, err := o.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal OrderV1 to bytes")
	}
	l := len(b)
	buf := make([]byte, 4+1+l)
	binary.BigEndian.PutUint32(buf, uint32(l))
	buf[4] = 1
	copy(buf[5:], b)
	return buf, nil
}

func (tx *ExchangeWithProofs) marshalAsOrderV2(order Order) ([]byte, error) {
	o, ok := order.(*OrderV2)
	if !ok {
		return nil, errors.New("failed to cast an order with version 2 to OrderV2")
	}
	b, err := o.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal OrderV2 to bytes")
	}
	l := len(b)
	buf := make([]byte, 4+l)
	binary.BigEndian.PutUint32(buf, uint32(l))
	copy(buf[4:], b)
	return buf, nil
}

func (tx *ExchangeWithProofs) marshalAsOrderV3(order Order) ([]byte, error) {
	o, ok := order.(*OrderV3)
	if !ok {
		return nil, errors.New("failed to cast an order with version 3 to OrderV3")
	}
	b, err := o.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal OrderV3 to bytes")
	}
	l := len(b)
	buf := make([]byte, 4+l)
	binary.BigEndian.PutUint32(buf, uint32(l))
	copy(buf[4:], b)
	return buf, nil
}

func (tx *ExchangeWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	var o1b []byte
	var o2b []byte
	var err error
	switch tx.Order1.GetVersion() {
	case 1:
		o1b, err = tx.marshalAsOrderV1(tx.Order1)
	case 2:
		o1b, err = tx.marshalAsOrderV2(tx.Order1)
	case 3:
		o1b, err = tx.marshalAsOrderV3(tx.Order1)
	default:
		err = errors.Errorf("invalid Order1 version %d", tx.Order1.GetVersion())
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal buy order to bytes")
	}
	o1l := uint32(len(o1b))
	switch tx.Order2.GetVersion() {
	case 1:
		o2b, err = tx.marshalAsOrderV1(tx.Order2)
	case 2:
		o2b, err = tx.marshalAsOrderV2(tx.Order2)
	case 3:
		o2b, err = tx.marshalAsOrderV3(tx.Order2)
	default:
		err = errors.Errorf("invalid Order2 version %d", tx.Order2.GetVersion())
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal sell order to bytes")
	}
	o2l := uint32(len(o2b))
	var p uint32
	buf := make([]byte, exchangeWithProofsFixedBodyLen+(o1l-4)+(o2l-4))
	buf[0] = 0
	buf[1] = byte(tx.Type)
	buf[2] = tx.Version
	p += 3
	copy(buf[p:], o1b)
	p += o1l
	copy(buf[p:], o2b)
	p += o2l
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

func (tx *ExchangeWithProofs) unmarshalOrder(data []byte) (int, Order, error) {
	var r Order
	n := 0
	ol := binary.BigEndian.Uint32(data)
	n += 4
	switch data[n] {
	case 1:
		n++
		o := new(OrderV1)
		err := o.UnmarshalBinary(data[n:])
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to unmarshal OrderV1")
		}
		n += int(ol)
		r = o
	case 2:
		o := new(OrderV2)
		err := o.UnmarshalBinary(data[n:])
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to unmarshal OrderV2")
		}
		n += int(ol)
		r = o
	case 3:
		o := new(OrderV3)
		err := o.UnmarshalBinary(data[n:])
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to unmarshal OrderV3")
		}
		n += int(ol)
		r = o
	default:
		return 0, nil, errors.Errorf("unexpected order version %d", data[n])
	}
	return n, r, nil
}

func (tx *ExchangeWithProofs) bodyUnmarshalBinary(data []byte) (int, error) {
	n := 0
	if l := len(data); l < exchangeWithProofsFixedBodyLen {
		return 0, errors.Errorf("not enough data for ExchangeWithProofs body, expected not less then %d, received %d", exchangeWithProofsFixedBodyLen, l)
	}
	if v := data[n]; v != 0 {
		return 0, errors.Errorf("unexpected first byte %d of ExchangeWithProofs body, expected 0", v)
	}
	n++
	tx.Type = TransactionType(data[n])
	if tx.Type != ExchangeTransaction {
		return 0, errors.Errorf("unexpected transaction type %d for ExchangeWithProofs transaction", tx.Type)
	}
	n++
	tx.Version = data[n]
	n++
	l, o, err := tx.unmarshalOrder(data[n:])
	if err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal buy order")
	}
	tx.Order1 = o
	n += l
	l, o, err = tx.unmarshalOrder(data[n:])
	if err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal sell order")
	}
	tx.Order2 = o
	n += l
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
	tx.SenderPK = tx.Order1.GetMatcherPK()
	return n, nil
}

// Sign calculates transaction signature using given secret key.
func (tx *ExchangeWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeWithProofs transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that the transaction signature is valid for given public key.
func (tx *ExchangeWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of ExchangeWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary saves the transaction to its binary representation.
func (tx *ExchangeWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ExchangeWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal ExchangeWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ExchangeWithProofs transaction to bytes")
	}
	buf := make([]byte, bl+len(pb))
	copy(buf, bb)
	copy(buf[bl:], pb)
	return buf, nil
}

// UnmarshalBinary loads the transaction from its binary representation.
func (tx *ExchangeWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < exchangeWithProofsMinLen {
		return errors.Errorf("not enough data for ExchangeWithProofs transaction, expected not less then %d, received %d", exchangeWithProofsMinLen, l)
	}
	bl, err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeWithProofs transaction from bytes")
	}
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *ExchangeWithProofs) UnmarshalJSON(data []byte) (err error) {
	type orderRecognizer struct {
		Version         byte               `json:"version"`
		Eip712Signature *EthereumSignature `json:"eip712Signature"`
	}
	orderVersions := struct {
		Order1Recognizer orderRecognizer `json:"order1"`
		Order2Recognizer orderRecognizer `json:"order2"`
	}{}
	guessOrderVersionAndType := func(orderInfo orderRecognizer) (order Order, err error) {
		switch version := orderInfo.Version; version {
		case 1:
			order = new(OrderV1)
		case 2:
			order = new(OrderV2)
		case 3:
			order = new(OrderV3)
		case 4:
			if orderInfo.Eip712Signature != nil {
				ethOrder := new(EthereumOrderV4)
				ethOrder.Proofs = NewProofs()
				order = ethOrder
			} else {
				order = new(OrderV4)
			}
		default:
			err = errors.Errorf("invalid order version %d", version)
		}
		return order, err
	}
	orderUnmarshalHelper := struct {
		Type           TransactionType  `json:"type"`
		Version        byte             `json:"version,omitempty"`
		ID             *crypto.Digest   `json:"id,omitempty"`
		Proofs         *ProofsV1        `json:"proofs,omitempty"`
		SenderPK       crypto.PublicKey `json:"senderPublicKey"`
		Order1         Order            `json:"order1"`
		Order2         Order            `json:"order2"`
		Price          uint64           `json:"price"`
		Amount         uint64           `json:"amount"`
		BuyMatcherFee  uint64           `json:"buyMatcherFee"`
		SellMatcherFee uint64           `json:"sellMatcherFee"`
		Fee            uint64           `json:"fee"`
		Timestamp      uint64           `json:"timestamp,omitempty"`
	}{}

	err = json.Unmarshal(data, &orderVersions)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal orders versions of ExchangeWithProofs transaction from JSON")
	}

	// TODO: check that Order1.GetProofs() != nil
	// TODO: support EthereumOrderV4: generate senderPK from Eip712Signature
	orderUnmarshalHelper.Order1, err = guessOrderVersionAndType(orderVersions.Order1Recognizer)
	if err != nil {
		return errors.Wrap(err, "failed to guess order1 version and type from JSON")
	}

	// TODO: check that Order1.GetProofs() != nil
	// TODO: support EthereumOrderV4: generate senderPK from Eip712Signature
	orderUnmarshalHelper.Order2, err = guessOrderVersionAndType(orderVersions.Order2Recognizer)
	if err != nil {
		return errors.Wrap(err, "failed to guess order2 version and type from JSON")
	}

	err = json.Unmarshal(data, &orderUnmarshalHelper)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeWithProofs from JSON")
	}

	tx.Type = orderUnmarshalHelper.Type
	tx.Version = orderUnmarshalHelper.Version
	tx.ID = orderUnmarshalHelper.ID
	// TODO: check that orderUnmarshalHelper.Proofs != nil
	tx.Proofs = orderUnmarshalHelper.Proofs
	tx.SenderPK = orderUnmarshalHelper.SenderPK
	tx.Order1 = orderUnmarshalHelper.Order1
	tx.Order2 = orderUnmarshalHelper.Order2
	tx.Price = orderUnmarshalHelper.Price
	tx.Amount = orderUnmarshalHelper.Amount
	tx.BuyMatcherFee = orderUnmarshalHelper.BuyMatcherFee
	tx.SellMatcherFee = orderUnmarshalHelper.SellMatcherFee
	tx.Fee = orderUnmarshalHelper.Fee
	tx.Timestamp = orderUnmarshalHelper.Timestamp
	return nil
}

// LeaseWithProofs is a second version of the LeaseWithSig transaction.
type LeaseWithProofs struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Lease
}

func (tx *LeaseWithProofs) Validate(scheme Scheme) (Transaction, error) {
	if tx.Version < 2 || tx.Version > MaxLeaseTransactionVersion {
		return tx, errors.Errorf("unexpected transaction version %d for LeaseWithProofs transaction", tx.Version)
	}
	ok, err := tx.Lease.Valid(scheme)
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx LeaseWithProofs) BinarySize() int {
	return 4 + tx.Proofs.BinarySize() + tx.Lease.BinarySize()
}

func (tx *LeaseWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *LeaseWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	leaseTx, ok := t.(*LeaseWithProofs)
	if !ok {
		return errors.New("failed to convert result to LeaseWithProofs")
	}
	*tx = *leaseTx
	return nil
}

func (tx *LeaseWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *LeaseWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	leaseTx, ok := t.(*LeaseWithProofs)
	if !ok {
		return errors.New("failed to convert result to LeaseWithProofs")
	}
	*tx = *leaseTx
	return nil
}

func (tx *LeaseWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
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

func (tx *LeaseWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx LeaseWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx LeaseWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *LeaseWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *LeaseWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *LeaseWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *LeaseWithProofs) Clone() *LeaseWithProofs {
	out := &LeaseWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedLeaseWithProofs creates new LeaseWithSig transaction without signature and ID set.
func NewUnsignedLeaseWithProofs(v byte, senderPK crypto.PublicKey, recipient Recipient, amount, fee, timestamp uint64) *LeaseWithProofs {
	l := Lease{
		SenderPK:  senderPK,
		Recipient: recipient,
		Amount:    amount,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &LeaseWithProofs{Type: LeaseTransaction, Version: v, Lease: l}
}

func (tx *LeaseWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	rl := tx.Recipient.BinarySize()
	buf := make([]byte, leaseWithProofsBodyLen+rl)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = 0 //Always zero, reserved for future extension of leasing assets.
	b, err := tx.Lease.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseWithSig transaction to bytes")
	}
	copy(buf[3:], b)
	return buf, nil
}

func (tx *LeaseWithProofs) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseWithProofsBodyLen {
		return errors.Errorf("not enough data for LeaseWithProofs transaction, expected not less then %d, received %d", leaseWithProofsBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != LeaseTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseWithProofs transaction", tx.Type)
	}
	tx.Version = data[1]
	var l Lease
	err := l.UnmarshalBinary(data[3:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseWithProofs transaction from bytes")
	}
	tx.Lease = l
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *LeaseWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseWithProofs transaction")
	}
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *LeaseWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of LeaseWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary saves the transaction to its binary representation.
func (tx *LeaseWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal LeaseWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads the transaction from bytes slice.
func (tx *LeaseWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < leaseWithProofsMinLen {
		return errors.Errorf("not enough data for LeaseWithProofs transaction, expected not less then %d, received %d", leaseWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseWithProofs transaction from bytes")
	}
	bl := leaseWithProofsBodyLen + tx.Recipient.BinarySize()
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

// LeaseCancelWithProofs same as LeaseCancelWithSig but with proofs.
type LeaseCancelWithProofs struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	LeaseCancel
}

func (tx LeaseCancelWithProofs) BinarySize() int {
	return 4 + tx.Proofs.BinarySize() + tx.LeaseCancel.BinarySize()
}

func (tx *LeaseCancelWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *LeaseCancelWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	leaseCancelTx, ok := t.(*LeaseCancelWithProofs)
	if !ok {
		return errors.New("failed to convert result to LeaseCancelWithProofs")
	}
	*tx = *leaseCancelTx
	return nil
}

func (tx *LeaseCancelWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *LeaseCancelWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	leaseCancelTx, ok := t.(*LeaseCancelWithProofs)
	if !ok {
		return errors.New("failed to convert result to LeaseCancelWithProofs")
	}
	*tx = *leaseCancelTx
	return nil
}

func (tx *LeaseCancelWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.LeaseCancel.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *LeaseCancelWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx LeaseCancelWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx LeaseCancelWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *LeaseCancelWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *LeaseCancelWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *LeaseCancelWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *LeaseCancelWithProofs) Clone() *LeaseCancelWithProofs {
	out := &LeaseCancelWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedLeaseCancelWithProofs creates new LeaseCancelWithProofs transaction structure without a signature and an ID.
func NewUnsignedLeaseCancelWithProofs(v byte, senderPK crypto.PublicKey, leaseID crypto.Digest, fee, timestamp uint64) *LeaseCancelWithProofs {
	lc := LeaseCancel{
		SenderPK:  senderPK,
		LeaseID:   leaseID,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &LeaseCancelWithProofs{Type: LeaseCancelTransaction, Version: v, LeaseCancel: lc}
}

func (tx *LeaseCancelWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 2 || tx.Version > MaxLeaseCancelTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for LeaseCancelWithProofs", tx.Version)
	}
	ok, err := tx.LeaseCancel.Valid()
	if !ok {
		return tx, err
	}
	// we don't need to validate scheme here because scheme is included in binary representation of tx
	// so if scheme is invalid == signature is invalid
	return tx, nil
}

func (tx *LeaseCancelWithProofs) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	buf := make([]byte, leaseCancelWithProofsBodyLen)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = scheme
	b, err := tx.LeaseCancel.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelWithProofs to bytes")
	}
	copy(buf[3:], b)
	return buf, nil
}

func (tx *LeaseCancelWithProofs) bodyUnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < leaseCancelWithProofsBodyLen {
		return errors.Errorf("not enough data for LeaseCancelWithProofs transaction, expected not less then %d, received %d", leaseCancelWithProofsBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != LeaseCancelTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseCancelWithProofs transaction", tx.Type)

	}
	tx.Version = data[1]
	if unmarshalledScheme := data[2]; unmarshalledScheme != scheme {
		return errors.Errorf("scheme mismatch: got %d, want %d", unmarshalledScheme, scheme)
	}
	var lc LeaseCancel
	err := lc.UnmarshalBinary(data[3:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelWithProofs from bytes")
	}
	tx.LeaseCancel = lc
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *LeaseCancelWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelWithProofs transaction")
	}
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *LeaseCancelWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of LeaseCancelWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary saves the transaction to its binary representation.
func (tx *LeaseCancelWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal LeaseCancelWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads the transaction from bytes slice.
func (tx *LeaseCancelWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < leaseCancelWithProofsMinLen {
		return errors.Errorf("not enough data for LeaseCancelWithProofs transaction, expected not less then %d, received %d", leaseCancelWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data, scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelWithProofs transaction from bytes")
	}
	data = data[leaseCancelWithProofsBodyLen:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

type CreateAliasWithProofs struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	CreateAlias
}

func (tx *CreateAliasWithProofs) Validate(scheme Scheme) (Transaction, error) {
	if tx.Version < 2 || tx.Version > MaxCreateAliasTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for CreateAliasWithProofs", tx.Version)
	}
	ok, err := tx.CreateAlias.Valid(scheme)
	if !ok {
		return tx, err
	}
	return tx, nil
}

func (tx CreateAliasWithProofs) BinarySize() int {
	return 3 + tx.Proofs.BinarySize() + tx.CreateAlias.BinarySize()
}

func (tx *CreateAliasWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *CreateAliasWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	createAliasTx, ok := t.(*CreateAliasWithProofs)
	if !ok {
		return errors.New("failed to convert result to CreateAliasWithProofs")
	}
	*tx = *createAliasTx
	return nil
}

func (tx *CreateAliasWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *CreateAliasWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	createAliasTx, ok := t.(*CreateAliasWithProofs)
	if !ok {
		return errors.New("failed to convert result to CreateAliasWithProofs")
	}
	*tx = *createAliasTx
	return nil
}

func (tx *CreateAliasWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := tx.CreateAlias.ToProtobuf()
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *CreateAliasWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx CreateAliasWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx CreateAliasWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *CreateAliasWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *CreateAliasWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *CreateAliasWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *CreateAliasWithProofs) Clone() *CreateAliasWithProofs {
	out := &CreateAliasWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

func NewUnsignedCreateAliasWithProofs(v byte, senderPK crypto.PublicKey, alias Alias, fee, timestamp uint64) *CreateAliasWithProofs {
	ca := CreateAlias{
		SenderPK:  senderPK,
		Alias:     alias,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &CreateAliasWithProofs{Type: CreateAliasTransaction, Version: v, CreateAlias: ca}
}

func (tx *CreateAliasWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	buf := make([]byte, createAliasWithProofsFixedBodyLen+len(tx.Alias.Alias))
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	b, err := tx.CreateAlias.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasWithProofs transaction body to bytes")
	}
	copy(buf[2:], b)
	return buf, nil
}

func (tx *CreateAliasWithProofs) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < createAliasWithProofsFixedBodyLen {
		return errors.Errorf("not enough data for CreateAliasWithProofs transaction, expected not less then %d, received %d", createAliasWithProofsFixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != CreateAliasTransaction {
		return errors.Errorf("unexpected transaction type %d for CreateAliasWithProofs transaction", tx.Type)
	}
	tx.Version = data[1]
	var ca CreateAlias
	err := ca.UnmarshalBinary(data[2:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasWithProofs transaction from bytes")
	}
	tx.CreateAlias = ca
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *CreateAliasWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasWithProofs transaction")
	}
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *CreateAliasWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of CreateAliasWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary saves the transaction to its binary representation.
func (tx *CreateAliasWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal CreateAliasWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads the transaction from bytes slice.
func (tx *CreateAliasWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < createAliasWithProofsMinLen {
		return errors.Errorf("not enough data for CreateAliasWithProofs transaction, expected not less then %d, received %d", createAliasWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasWithProofs transaction from bytes")
	}
	data = data[createAliasWithProofsFixedBodyLen+len(tx.Alias.Alias):]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

// Deprecated: use UnmarshalJSONWithScheme.
func (tx *CreateAliasWithProofs) UnmarshalJSON(data []byte) error {
	const ignoreChainID Scheme = 0
	return tx.UnmarshalJSONWithScheme(data, ignoreChainID)
}

func (tx *CreateAliasWithProofs) UnmarshalJSONWithScheme(data []byte, scheme Scheme) error {
	tmp := struct {
		Type      TransactionType  `json:"type"`
		Version   byte             `json:"version,omitempty"`
		Proofs    *ProofsV1        `json:"proofs,omitempty"`
		SenderPK  crypto.PublicKey `json:"senderPublicKey"`
		Alias     string           `json:"alias"`
		Fee       uint64           `json:"fee"`
		Timestamp uint64           `json:"timestamp,omitempty"`
	}{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasWithSig from JSON")
	}
	tx.Type = tmp.Type
	tx.Version = tmp.Version
	tx.Proofs = tmp.Proofs
	tx.SenderPK = tmp.SenderPK
	tx.Alias = *NewAlias(scheme, tmp.Alias)
	tx.Fee = tmp.Fee
	tx.Timestamp = tmp.Timestamp
	return nil
}

func (tx *CreateAliasWithProofs) MarshalJSON() ([]byte, error) {
	type shadowed CreateAliasWithProofs
	tmp := struct {
		Alias string `json:"alias"`
		*shadowed
	}{tx.Alias.Alias, (*shadowed)(tx)}
	return json.Marshal(tmp)
}

type MassTransferEntry struct {
	Recipient Recipient `json:"recipient"`
	Amount    uint64    `json:"amount"`
}

func (e *MassTransferEntry) BinarySize() int {
	return e.Recipient.BinarySize() + 8
}

func (e *MassTransferEntry) ToProtobuf() (*g.MassTransferTransactionData_Transfer, error) {
	rcpProto, err := e.Recipient.ToProtobuf()
	if err != nil {
		return nil, err
	}
	return &g.MassTransferTransactionData_Transfer{Recipient: rcpProto, Amount: int64(e.Amount)}, nil
}

func (e *MassTransferEntry) MarshalBinary() ([]byte, error) {
	rb, err := e.Recipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferEntry")
	}
	rl := e.Recipient.BinarySize()
	buf := make([]byte, massTransferEntryLen+rl)
	copy(buf, rb)
	binary.BigEndian.PutUint64(buf[rl:], e.Amount)
	return buf, nil
}

func (e *MassTransferEntry) UnmarshalBinary(data []byte) error {
	if l := len(data); l < massTransferEntryLen {
		return errors.Errorf("not enough data to unmarshal MassTransferEntry from byte, expected %d, received %d bytes", massTransferEntryLen, l)
	}
	err := e.Recipient.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferEntry from bytes")
	}
	e.Amount = binary.BigEndian.Uint64(data[e.Recipient.BinarySize():])
	return nil
}

// MassTransferWithProofs is a transaction that performs multiple transfers of one asset to the accounts at once.
type MassTransferWithProofs struct {
	Type       TransactionType     `json:"type"`
	Version    byte                `json:"version,omitempty"`
	ID         *crypto.Digest      `json:"id,omitempty"`
	Proofs     *ProofsV1           `json:"proofs,omitempty"`
	SenderPK   crypto.PublicKey    `json:"senderPublicKey"`
	Asset      OptionalAsset       `json:"assetId"`
	Transfers  []MassTransferEntry `json:"transfers"`
	Timestamp  uint64              `json:"timestamp,omitempty"`
	Fee        uint64              `json:"fee"`
	Attachment Attachment          `json:"attachment,omitempty"`
}

func (tx MassTransferWithProofs) BinarySize() int {
	size := 2 + tx.Proofs.BinarySize() + crypto.PublicKeySize + tx.Asset.BinarySize() + 16 + 2 + tx.attachmentSize()
	size += 2
	for _, tr := range tx.Transfers {
		size += tr.BinarySize()
	}
	return size
}

func (tx MassTransferWithProofs) HasRecipient(rcp Recipient) bool {
	for _, tr := range tx.Transfers {
		if tr.Recipient.Eq(rcp) {
			return true
		}
	}
	return false
}

func (tx MassTransferWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx MassTransferWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *MassTransferWithProofs) Clone() *MassTransferWithProofs {
	out := &MassTransferWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

func (tx *MassTransferWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *MassTransferWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx MassTransferWithProofs) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx MassTransferWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx *MassTransferWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx MassTransferWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx MassTransferWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

// NewUnsignedMassTransferWithProofs creates new MassTransferWithProofs transaction structure without signature and ID.
func NewUnsignedMassTransferWithProofs(v byte, senderPK crypto.PublicKey, asset OptionalAsset, transfers []MassTransferEntry, fee, timestamp uint64, attachment Attachment) *MassTransferWithProofs {
	return &MassTransferWithProofs{Type: MassTransferTransaction, Version: v, SenderPK: senderPK, Asset: asset, Transfers: transfers, Fee: fee, Timestamp: timestamp, Attachment: attachment}
}

func (tx *MassTransferWithProofs) Validate(scheme Scheme) (Transaction, error) {
	if tx.Version < 1 || tx.Version > MaxMassTransferTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for MassTransferWithProofs", tx.Version)
	}
	if len(tx.Transfers) > maxTransfers {
		return tx, errs.NewTxValidationError(fmt.Sprintf("Number of transfers %d is greater than %d", len(tx.Transfers), maxTransfers))
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	total := tx.Fee
	for _, t := range tx.Transfers {
		if !validJVMLong(t.Amount) {
			return tx, errors.New("at least one of the transfers amount is bigger than JVM long")
		}
		total += t.Amount
		if !validJVMLong(total) {
			return tx, errors.New("sum of amounts of transfers and transaction fee is bigger than JVM long")
		}
		if ok, err := t.Recipient.Valid(scheme); !ok {
			return tx, errors.Wrap(err, "invalid recipient")
		}
	}
	if tx.attachmentSize() > maxAttachmentLengthBytes {
		return tx, errs.NewTooBigArray("attachment too long")
	}
	return tx, nil
}

func (tx *MassTransferWithProofs) attachmentSize() int {
	if tx.Attachment != nil {
		return tx.Attachment.Size()
	}
	return 0
}

func (tx *MassTransferWithProofs) bodyAndAssetLen() (int, int) {
	n := len(tx.Transfers)
	l := 0
	if tx.Asset.Present {
		l += crypto.DigestSize
	}
	rls := 0
	for _, e := range tx.Transfers {
		rls += e.Recipient.BinarySize()
	}
	al := tx.attachmentSize()
	return massTransferWithProofsFixedLen + l + n*massTransferEntryLen + rls + al, l
}

func (tx *MassTransferWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	var p int
	n := len(tx.Transfers)
	bl, al := tx.bodyAndAssetLen()
	buf := make([]byte, bl)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	ab, err := tx.Asset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferWithProofs transaction body to bytes")
	}
	copy(buf[p:], ab)
	p += 1 + al
	binary.BigEndian.PutUint16(buf[p:], uint16(n))
	p += 2
	for _, t := range tx.Transfers {
		tb, err := t.MarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal MassTransferWithProofs transaction body to bytes")
		}
		copy(buf[p:], tb)
		p += massTransferEntryLen + t.Recipient.BinarySize()
	}
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	if err := PutBytesWithUInt16Len(buf[p:], tx.Attachment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferWithProofs transaction body to bytes")
	}
	return buf, nil
}

func (tx *MassTransferWithProofs) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if l := len(data); l < massTransferWithProofsMinLen {
		return errors.Errorf("not enough data for MassTransferWithProofs transaction, expected not less then %d, received %d", massTransferWithProofsMinLen, l)
	}
	if tx.Type != MassTransferTransaction {
		return errors.Errorf("unexpected transaction type %d for MassTransferWithProofs transaction", tx.Type)
	}
	data = data[2:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	err := tx.Asset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferWithProofs from bytes")
	}
	data = data[1:]
	if tx.Asset.Present {
		data = data[crypto.DigestSize:]
	}
	n := int(binary.BigEndian.Uint16(data))
	data = data[2:]
	var entries []MassTransferEntry
	for i := 0; i < n; i++ {
		var e MassTransferEntry
		err := e.UnmarshalBinary(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal MassTransferWithProofs transaction body from bytes")
		}
		data = data[massTransferEntryLen+e.Recipient.BinarySize():]
		entries = append(entries, e)
	}
	tx.Transfers = entries
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	at, err := BytesWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferWithProofs transaction body from bytes")
	}
	tx.Attachment = at
	return nil
}

// Sign calculates signature and ID of the transaction.
func (tx *MassTransferWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign MassTransferWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign MassTransferWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign MassTransferWithProofs transaction")
	}
	return nil
}

// Verify checks that the signature is valid for the given public key.
func (tx *MassTransferWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of MassTransferWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary saves the transaction to its binary representation.
func (tx *MassTransferWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal MassTransferWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferWithProofs transaction to bytes")
	}
	pl := len(pb)
	buf := make([]byte, bl+pl)
	copy(buf[0:], bb)
	copy(buf[bl:], pb)
	return buf, nil
}

// UnmarshalBinary loads transaction from its binary representation.
func (tx *MassTransferWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < massTransferWithProofsMinLen {
		return errors.Errorf("not enough data for MassTransferWithProofs transaction, expected not less then %d, received %d", massTransferWithProofsMinLen, l)
	}
	if data[0] != byte(MassTransferTransaction) {
		return errors.Errorf("incorrect transaction type %d for MassTransferWithProofs transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferWithProofs transaction from bytes")
	}
	bl, _ := tx.bodyAndAssetLen()
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *MassTransferWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *MassTransferWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	massTransferTx, ok := t.(*MassTransferWithProofs)
	if !ok {
		return errors.New("failed to convert result to MassTransferWithProofs")
	}
	*tx = *massTransferTx
	return nil
}

func (tx *MassTransferWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *MassTransferWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	massTransferTx, ok := t.(*MassTransferWithProofs)
	if !ok {
		return errors.New("failed to convert result to MassTransferWithProofs")
	}
	*tx = *massTransferTx
	return nil
}

func (tx *MassTransferWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	var err error
	transfers := make([]*g.MassTransferTransactionData_Transfer, len(tx.Transfers))
	for i, tr := range tx.Transfers {
		transfers[i], err = tr.ToProtobuf()
		if err != nil {
			return nil, err
		}
	}
	txData := &g.Transaction_MassTransfer{MassTransfer: &g.MassTransferTransactionData{
		AssetId:    tx.Asset.ToID(),
		Transfers:  transfers,
		Attachment: tx.Attachment,
	}}
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *MassTransferWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx *MassTransferWithProofs) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Type       TransactionType     `json:"type"`
		Version    byte                `json:"version,omitempty"`
		ID         *crypto.Digest      `json:"id,omitempty"`
		Proofs     *ProofsV1           `json:"proofs,omitempty"`
		SenderPK   crypto.PublicKey    `json:"senderPublicKey"`
		Asset      OptionalAsset       `json:"assetId"`
		Transfers  []MassTransferEntry `json:"transfers"`
		Timestamp  uint64              `json:"timestamp,omitempty"`
		Fee        uint64              `json:"fee"`
		Attachment Attachment          `json:"attachment,omitempty"`
	}{}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	tx.Type = tmp.Type
	tx.Version = tmp.Version
	tx.ID = tmp.ID
	tx.Proofs = tmp.Proofs
	tx.SenderPK = tmp.SenderPK
	tx.Asset = tmp.Asset
	tx.Transfers = tmp.Transfers
	tx.Timestamp = tmp.Timestamp
	tx.Fee = tmp.Fee
	tx.Attachment = tmp.Attachment
	return nil
}

// DataWithProofs is first version of the transaction that puts data to the key-value storage of an account.
type DataWithProofs struct {
	Type      TransactionType  `json:"type"`
	Version   byte             `json:"version,omitempty"`
	ID        *crypto.Digest   `json:"id,omitempty"`
	Proofs    *ProofsV1        `json:"proofs,omitempty"`
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	Entries   DataEntries      `json:"data"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func (tx DataWithProofs) BinarySize() int {
	size := 3 + tx.Proofs.BinarySize() + crypto.PublicKeySize + 16
	size += 2
	size += tx.Entries.BinarySize()
	return size
}

func (tx DataWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx DataWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *DataWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *DataWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx DataWithProofs) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx DataWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx *DataWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx DataWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx DataWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

func (tx *DataWithProofs) Clone() *DataWithProofs {
	out := &DataWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

func NewUnsignedDataWithProofs(v byte, senderPK crypto.PublicKey, fee, timestamp uint64) *DataWithProofs {
	return &DataWithProofs{Type: DataTransaction, Version: v, SenderPK: senderPK, Fee: fee, Timestamp: timestamp}
}

func (tx *DataWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 1 || tx.Version > MaxDataTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for DataWithProofs", tx.Version)
	}
	if len(tx.Entries) > maxEntries {
		return tx, errs.NewTooBigArray(fmt.Sprintf("number of DataWithProofs entries is bigger than %d", maxEntries))
	}
	isPBTx := IsProtobufTx(tx)
	keys := make(map[string]struct{}, len(tx.Entries))
	for _, e := range tx.Entries {
		if !isPBTx && e.GetValueType() == DataDelete {
			return tx, errors.New("delete supported only for protobuf transaction")
		}
		key := e.GetKey()
		if _, ok := keys[key]; ok {
			return tx, errs.NewDuplicatedDataKeys(fmt.Sprintf("duplicate key %s", key))
		}
		keys[key] = struct{}{}
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if tx.Fee < MinFee {
		return tx, errs.NewTxValidationError(fmt.Sprintf("Fee %d does not exceed minimal value", tx.Fee))
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	// see tx size and entries validation in transactionChecker
	return tx, nil
}

// AppendEntry adds the entry to the transaction.
func (tx *DataWithProofs) AppendEntry(entry DataEntry) error {
	if len(entry.GetKey()) == 0 {
		return errors.Errorf("empty keys are not allowed")
	}
	key := entry.GetKey()
	for _, e := range tx.Entries {
		if e.GetKey() == key {
			return errors.Errorf("key '%s' already exist", key)
		}
	}
	tx.Entries = append(tx.Entries, entry)
	return nil
}

func (tx *DataWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	var p int
	n := len(tx.Entries)
	el := tx.Entries.BinarySize()
	buf := make([]byte, dataWithProofsFixedBodyLen+el)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	binary.BigEndian.PutUint16(buf[p:], uint16(n))
	p += 2
	for _, e := range tx.Entries {
		eb, err := e.MarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal DataWithProofs transaction body to bytes")
		}
		copy(buf[p:], eb)
		p += e.BinarySize()
	}
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	return buf, nil
}

func (tx *DataWithProofs) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if l := len(data); l < dataWithProofsFixedBodyLen {
		return errors.Errorf("not enough data for DataWithProofs transaction, expected not less then %d, received %d", dataWithProofsFixedBodyLen, l)
	}
	if tx.Type != DataTransaction {
		return errors.Errorf("unexpected transaction type %d for DataWithProofs transaction", tx.Type)
	}
	data = data[2:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	n := int(binary.BigEndian.Uint16(data))
	data = data[2:]
	for i := 0; i < n; i++ {
		var e DataEntry
		t, err := extractValueType(data)
		if err != nil {
			return errors.Errorf("failed to extract type of data entry")
		}
		switch t {
		case DataInteger:
			var ie IntegerDataEntry
			err = ie.UnmarshalBinary(data)
			e = &ie
		case DataBoolean:
			var be BooleanDataEntry
			err = be.UnmarshalBinary(data)
			e = &be
		case DataBinary:
			var be BinaryDataEntry
			err = be.UnmarshalBinary(data)
			e = &be
		case DataString:
			var se StringDataEntry
			err = se.UnmarshalBinary(data)
			e = &se
		default:
			return errors.Errorf("unsupported ValueType %d", t)
		}
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal DataWithProofs transaction body from bytes")
		}
		data = data[e.BinarySize():]
		err = tx.AppendEntry(e)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal DataWithProofs transaction body from bytes")
		}
	}
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	return nil
}

func extractValueType(data []byte) (DataValueType, error) {
	if l := len(data); l < 3 {
		return 0, errors.Errorf("not enough data to extract ValueType, expected not less than %d, received %d", 3, l)
	}
	kl := binary.BigEndian.Uint16(data)
	if l := len(data); l <= int(kl)+2 {
		return 0, errors.Errorf("not enough data to extract ValueType, expected more than %d, received %d", kl+2, l)
	}
	return DataValueType(data[kl+2]), nil
}

// Sign use given secret key to calculate signature of the transaction.
func (tx *DataWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign DataWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign DataWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign DataWithProofs transaction")
	}
	return nil
}

// Verify checks that the signature is valid for the given public key.
func (tx *DataWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of DataWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary saves the transaction to bytes.
func (tx *DataWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal DataWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal DataWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal DataWithProofs transaction to bytes")
	}
	pl := len(pb)
	buf := make([]byte, 1+bl+pl)
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads the transaction from the bytes.
func (tx *DataWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if len(data) > maxDataWithProofsTxBytes {
		return errors.Errorf("total size of DataWithProofs transaction is bigger than %d bytes", maxDataWithProofsTxBytes)
	}
	if l := len(data); l < dataWithProofsMinLen {
		return errors.Errorf("not enough data for DataWithProofs transaction, expected not less then %d, received %d", dataWithProofsMinLen, l)
	}
	if data[0] != 0 {
		return errors.Errorf("unexpected first byte %d for DataWithProofs transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal DataWithProofs transaction from bytes")
	}
	bl := dataWithProofsFixedBodyLen + tx.Entries.BinarySize()
	data = data[1+bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal DataWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *DataWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *DataWithProofs) UnmarshalFromProtobuf(data []byte) error {
	if len(data) > maxDataWithProofsTxBytes {
		return errors.Errorf("total size of DataWithProofs transaction is bigger than %d bytes", maxDataWithProofsTxBytes)
	}
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	dataTx, ok := t.(*DataWithProofs)
	if !ok {
		return errors.New("failed to convert result to DataWithProofs")
	}
	*tx = *dataTx
	return nil
}

func (tx *DataWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *DataWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	if len(data) > maxDataWithProofsTxBytes {
		return errors.Errorf("total size of DataWithProofs transaction is bigger than %d bytes", maxDataWithProofsTxBytes)
	}
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	dataTx, ok := t.(*DataWithProofs)
	if !ok {
		return errors.New("failed to convert result to DataWithProofs")
	}
	*tx = *dataTx
	return nil
}

func (tx *DataWithProofs) protobufDataTransactionData() *g.DataTransactionData {
	entries := make([]*g.DataTransactionData_DataEntry, len(tx.Entries))
	for i, entry := range tx.Entries {
		entries[i] = entry.ToProtobuf()
	}
	return &g.DataTransactionData{Data: entries}
}

func (tx *DataWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	txData := &g.Transaction_DataTransaction{DataTransaction: tx.protobufDataTransactionData()}
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *DataWithProofs) ProtoPayload() *g.DataTransactionData {
	return tx.protobufDataTransactionData()
}

func (tx *DataWithProofs) ProtoPayloadSize() int {
	// use this method to calculate PB binary size of payload
	return tx.ProtoPayload().SizeVT()
}

func (tx *DataWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

// SetScriptWithProofs is a transaction to set smart script on an account.
type SetScriptWithProofs struct {
	Type      TransactionType  `json:"type"`
	Version   byte             `json:"version,omitempty"`
	ID        *crypto.Digest   `json:"id,omitempty"`
	Proofs    *ProofsV1        `json:"proofs,omitempty"`
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	Script    Script           `json:"script"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func (tx SetScriptWithProofs) BinarySize() int {
	scriptSize := 1
	if len(tx.Script) > 0 {
		scriptSize += 2 + len(tx.Script)
	}
	return 3 + tx.Proofs.BinarySize() + 1 + crypto.PublicKeySize + 16 + scriptSize
}

func (tx SetScriptWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx SetScriptWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *SetScriptWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *SetScriptWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx SetScriptWithProofs) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx SetScriptWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx *SetScriptWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx SetScriptWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx SetScriptWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

// NewUnsignedSetScriptWithProofs creates new unsigned SetScriptWithProofs transaction.
func NewUnsignedSetScriptWithProofs(v byte, senderPK crypto.PublicKey, script []byte, fee, timestamp uint64) *SetScriptWithProofs {
	return &SetScriptWithProofs{Type: SetScriptTransaction, Version: v, SenderPK: senderPK, Script: script, Fee: fee, Timestamp: timestamp}
}

func (tx *SetScriptWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 1 || tx.Version > MaxSetScriptTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for SetScriptWithProofs", tx.Version)
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	// we don't need to validate scheme here because scheme is included in binary representation of tx
	// so if scheme is invalid == signature is invalid
	return tx, nil
}

// NonEmptyScript returns true if transaction contains non-empty script.
func (tx *SetScriptWithProofs) NonEmptyScript() bool {
	return len(tx.Script) != 0
}

func (tx *SetScriptWithProofs) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	var p int
	sl := 0
	if tx.NonEmptyScript() {
		sl = len(tx.Script) + 2
	}
	buf := make([]byte, setScriptWithProofsFixedBodyLen+sl)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	buf[p] = scheme
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	PutBool(buf[p:], tx.NonEmptyScript())
	p++
	if tx.NonEmptyScript() {
		if err := PutBytesWithUInt16Len(buf[p:], tx.Script); err != nil {
			return nil, errors.Wrap(err, "failed to marshal body of SetScriptWithProofs transaction")
		}
		p += sl
	}
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func (tx *SetScriptWithProofs) bodyUnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < setScriptWithProofsFixedBodyLen {
		return errors.Errorf("not enough data for SetScriptWithProofs transaction, expected not less then %d, received %d", setScriptWithProofsFixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if unmarshalledScheme := data[2]; unmarshalledScheme != scheme {
		return errors.Errorf("scheme mismatch: got %d, want %d", unmarshalledScheme, scheme)
	}
	if tx.Type != SetScriptTransaction {
		return errors.Errorf("unexpected transaction type %d for SetScriptWithProofs transaction", tx.Type)
	}
	data = data[3:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	p, err := Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetScripV1 transaction body from bytes")
	}
	data = data[1:]
	if p {
		s, err := BytesWithUInt16Len(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal SetScriptWithProofs transaction body from bytes")
		}
		tx.Script = s
		data = data[2+len(s):]
	}
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *SetScriptWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign SetScriptWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign SetScriptWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign SetScriptWithProofs transaction")
	}
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *SetScriptWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of SetScriptWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary writes SetScriptWithProofs transaction to its byte representation.
func (tx *SetScriptWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SetScriptWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal SetScriptWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SetScriptWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads SetScriptWithProofs transaction from its binary representation.
func (tx *SetScriptWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < setScriptWithProofsMinLen {
		return errors.Errorf("not enough data for SetScriptWithProofs transaction, expected not less then %d, received %d", setScriptWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data, scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetScriptWithProofs transaction from bytes")
	}
	sl := 0
	if tx.NonEmptyScript() {
		sl = len(tx.Script) + 2
	}
	bl := setScriptWithProofsFixedBodyLen + sl
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetScriptWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *SetScriptWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *SetScriptWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	setScriptTx, ok := t.(*SetScriptWithProofs)
	if !ok {
		return errors.New("failed to convert result to SetScriptWithProofs")
	}
	*tx = *setScriptTx
	return nil
}

func (tx *SetScriptWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *SetScriptWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	setScriptTx, ok := t.(*SetScriptWithProofs)
	if !ok {
		return errors.New("failed to convert result to SetScriptWithProofs")
	}
	*tx = *setScriptTx
	return nil
}

func (tx *SetScriptWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	txData := &g.Transaction_SetScript{SetScript: &g.SetScriptTransactionData{
		Script: tx.Script,
	}}
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *SetScriptWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

// SponsorshipWithProofs is a transaction to set up fee sponsorship for an asset.
type SponsorshipWithProofs struct {
	Type        TransactionType  `json:"type"`
	Version     byte             `json:"version,omitempty"`
	ID          *crypto.Digest   `json:"id,omitempty"`
	Proofs      *ProofsV1        `json:"proofs,omitempty"`
	SenderPK    crypto.PublicKey `json:"senderPublicKey"`
	AssetID     crypto.Digest    `json:"assetId"`
	MinAssetFee uint64           `json:"minSponsoredAssetFee"`
	Fee         uint64           `json:"fee"`
	Timestamp   uint64           `json:"timestamp,omitempty"`
}

func (tx SponsorshipWithProofs) BinarySize() int {
	return 5 + tx.Proofs.BinarySize() + crypto.PublicKeySize + crypto.DigestSize + 24
}

func (tx SponsorshipWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx SponsorshipWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *SponsorshipWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *SponsorshipWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx SponsorshipWithProofs) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx SponsorshipWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx *SponsorshipWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx SponsorshipWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx SponsorshipWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

func (tx *SponsorshipWithProofs) Clone() *SponsorshipWithProofs {
	out := &SponsorshipWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedSponsorshipWithProofs creates new unsigned SponsorshipWithProofs transaction
func NewUnsignedSponsorshipWithProofs(v byte, senderPK crypto.PublicKey, assetID crypto.Digest, minAssetFee, fee, timestamp uint64) *SponsorshipWithProofs {
	return &SponsorshipWithProofs{Type: SponsorshipTransaction, Version: v, SenderPK: senderPK, AssetID: assetID, MinAssetFee: minAssetFee, Fee: fee, Timestamp: timestamp}
}

func (tx *SponsorshipWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 1 || tx.Version > MaxSponsorshipTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for SponsorshipWithProofs", tx.Version)
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	if !validJVMLong(tx.MinAssetFee) {
		return tx, errors.New("min asset fee is too big")
	}
	return tx, nil
}

func (tx *SponsorshipWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	var p int
	buf := make([]byte, sponsorshipWithProofsBodyLen)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], tx.AssetID[:])
	p += crypto.DigestSize
	binary.BigEndian.PutUint64(buf[p:], tx.MinAssetFee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func (tx *SponsorshipWithProofs) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < sponsorshipWithProofsBodyLen {
		return errors.Errorf("not enough data for SponsorshipWithProofs transaction body, expected %d bytes, received %d", sponsorshipWithProofsBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if tx.Type != SponsorshipTransaction {
		return errors.Errorf("unexpected transaction type %d for SponsorshipWithProofs transaction", tx.Type)
	}
	data = data[2:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(tx.AssetID[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	tx.MinAssetFee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *SponsorshipWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign SponsorshipWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign SponsorshipWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign SponsorshipWithProofs transaction")
	}
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *SponsorshipWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of SponsorshipWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary writes SponsorshipWithProofs transaction to its byte representation.
func (tx *SponsorshipWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SponsorshipWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal SponsorshipWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SponsorshipWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+1+1+bl+len(pb))
	buf[0] = 0
	buf[1] = byte(tx.Type)
	buf[2] = tx.Version
	copy(buf[3:], bb)
	copy(buf[3+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads SponsorshipWithProofs from its byte representation.
func (tx *SponsorshipWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < sponsorshipWithProofsMinLen {
		return errors.Errorf("not enough data for SponsorshipWithProofs transaction, expected not less then %d, received %d", sponsorshipWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	if t := data[0]; t != byte(SponsorshipTransaction) {
		return errors.Errorf("unexpected transaction type %d, expected %d", t, SponsorshipTransaction)
	}
	data = data[1:]
	if v := data[0]; v != 1 {
		return errors.Errorf("unexpected transaction version %d, expected %d", v, 1)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SponsorshipWithProofs transaction from bytes")
	}
	bl := sponsorshipWithProofsBodyLen
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SponsorshipWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *SponsorshipWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *SponsorshipWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	sponsorshipTx, ok := t.(*SponsorshipWithProofs)
	if !ok {
		return errors.New("failed to convert result to SponsorshipWithProofs")
	}
	*tx = *sponsorshipTx
	return nil
}

func (tx *SponsorshipWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *SponsorshipWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	sponsorshipTx, ok := t.(*SponsorshipWithProofs)
	if !ok {
		return errors.New("failed to convert result to SponsorshipWithProofs")
	}
	*tx = *sponsorshipTx
	return nil
}

func (tx *SponsorshipWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	txData := &g.Transaction_SponsorFee{SponsorFee: &g.SponsorFeeTransactionData{
		MinFee: &g.Amount{AssetId: tx.AssetID.Bytes(), Amount: int64(tx.MinAssetFee)},
	}}
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *SponsorshipWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

// SetAssetScriptWithProofs is a transaction to set smart script on an asset.
type SetAssetScriptWithProofs struct {
	Type      TransactionType  `json:"type"`
	Version   byte             `json:"version,omitempty"`
	ID        *crypto.Digest   `json:"id,omitempty"`
	Proofs    *ProofsV1        `json:"proofs,omitempty"`
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	AssetID   crypto.Digest    `json:"assetId"`
	Script    Script           `json:"script"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func (tx SetAssetScriptWithProofs) BinarySize() int {
	scriptSize := 1
	if len(tx.Script) > 0 {
		scriptSize += 2 + len(tx.Script)
	}
	return 4 + crypto.DigestSize + tx.Proofs.BinarySize() + crypto.PublicKeySize + 16 + scriptSize
}

func (tx SetAssetScriptWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx SetAssetScriptWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *SetAssetScriptWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *SetAssetScriptWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx SetAssetScriptWithProofs) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx SetAssetScriptWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx *SetAssetScriptWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx SetAssetScriptWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx SetAssetScriptWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

func (tx *SetAssetScriptWithProofs) Clone() *SetAssetScriptWithProofs {
	out := &SetAssetScriptWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedSetAssetScriptWithProofs creates new unsigned SetAssetScriptWithProofs transaction.
func NewUnsignedSetAssetScriptWithProofs(v byte, senderPK crypto.PublicKey, assetID crypto.Digest, script []byte, fee, timestamp uint64) *SetAssetScriptWithProofs {
	return &SetAssetScriptWithProofs{Type: SetAssetScriptTransaction, Version: v, SenderPK: senderPK, AssetID: assetID, Script: script, Fee: fee, Timestamp: timestamp}
}

func (tx *SetAssetScriptWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 1 || tx.Version > MaxSetAssetScriptTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for SetAssetScriptWithProofs", tx.Version)
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	// we don't need to validate scheme here because scheme is included in binary representation of tx
	// so if scheme is invalid == signature is invalid
	return tx, nil
}

// NonEmptyScript returns true if transaction contains non-empty script.
func (tx *SetAssetScriptWithProofs) NonEmptyScript() bool {
	return len(tx.Script) != 0
}

func (tx *SetAssetScriptWithProofs) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	var p int
	sl := 0
	if tx.NonEmptyScript() {
		sl = len(tx.Script) + 2
	}
	buf := make([]byte, setAssetScriptWithProofsFixedBodyLen+sl)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	buf[p] = scheme
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], tx.AssetID[:])
	p += crypto.DigestSize
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	PutBool(buf[p:], tx.NonEmptyScript())
	p++
	if tx.NonEmptyScript() {
		if err := PutBytesWithUInt16Len(buf[p:], tx.Script); err != nil {
			return nil, errors.Wrap(err, "failed to marshal body of SetAssetScriptWithProofs transaction")
		}
	}
	return buf, nil
}

func (tx *SetAssetScriptWithProofs) bodyUnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < setAssetScriptWithProofsFixedBodyLen {
		return errors.Errorf("not enough data for SetAssetScriptWithProofs transaction, expected not less then %d, received %d", setAssetScriptWithProofsFixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if unmarshalledScheme := data[2]; unmarshalledScheme != scheme {
		return errors.Errorf("scheme mismatch: got %d, want %d", unmarshalledScheme, scheme)
	}
	if tx.Type != SetAssetScriptTransaction {
		return errors.Errorf("unexpected transaction type %d for SetAssetScriptWithProofs transaction", tx.Type)
	}
	data = data[3:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(tx.AssetID[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	p, err := Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetAssetScripV1 transaction body from bytes")
	}
	data = data[1:]
	if p {
		s, err := BytesWithUInt16Len(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal SetAssetScriptWithProofs transaction body from bytes")
		}
		tx.Script = s
	}
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *SetAssetScriptWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign SetAssetScriptWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign SetAssetScriptWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign SetAssetScriptWithProofs transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *SetAssetScriptWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of SetAssetScriptWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary writes SetAssetScriptWithProofs transaction to its byte representation.
func (tx *SetAssetScriptWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SetAssetScriptWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal SetAssetScriptWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SetAssetScriptWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads SetAssetScriptWithProofs transaction from its binary representation.
func (tx *SetAssetScriptWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < setAssetScriptWithProofsMinLen {
		return errors.Errorf("not enough data for SetAssetScriptWithProofs transaction, expected not less then %d, received %d", setAssetScriptWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data, scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetAssetScriptWithProofs transaction from bytes")
	}
	sl := 0
	if tx.NonEmptyScript() {
		sl = len(tx.Script) + 2
	}
	bl := setAssetScriptWithProofsFixedBodyLen + sl
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetAssetScriptWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *SetAssetScriptWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *SetAssetScriptWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	setAssetScriptTx, ok := t.(*SetAssetScriptWithProofs)
	if !ok {
		return errors.New("failed to convert result to SetAssetScripV1")
	}
	*tx = *setAssetScriptTx
	return nil
}

func (tx *SetAssetScriptWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *SetAssetScriptWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	setAssetScriptTx, ok := t.(*SetAssetScriptWithProofs)
	if !ok {
		return errors.New("failed to convert result to SetAssetScriptWithProofs")
	}
	*tx = *setAssetScriptTx
	return nil
}

func (tx *SetAssetScriptWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	txData := &g.Transaction_SetAssetScript{SetAssetScript: &g.SetAssetScriptTransactionData{
		AssetId: tx.AssetID.Bytes(),
		Script:  tx.Script,
	}}
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *SetAssetScriptWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

type InvokeScriptWithProofs struct {
	Type            TransactionType  `json:"type"`
	Version         byte             `json:"version,omitempty"`
	ID              *crypto.Digest   `json:"id,omitempty"`
	Proofs          *ProofsV1        `json:"proofs,omitempty"`
	SenderPK        crypto.PublicKey `json:"senderPublicKey"`
	ScriptRecipient Recipient        `json:"dApp"`
	FunctionCall    FunctionCall     `json:"call"`
	Payments        ScriptPayments   `json:"payment"`
	FeeAsset        OptionalAsset    `json:"feeAssetId"`
	Fee             uint64           `json:"fee"`
	Timestamp       uint64           `json:"timestamp,omitempty"`
}

func (tx *InvokeScriptWithProofs) BinarySize() int {
	return 4 + tx.Proofs.BinarySize() + crypto.PublicKeySize + tx.FunctionCall.BinarySize() + tx.ScriptRecipient.BinarySize() + tx.Payments.BinarySize() + tx.FeeAsset.BinarySize() + 16
}

func (tx *InvokeScriptWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *InvokeScriptWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx InvokeScriptWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx InvokeScriptWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx InvokeScriptWithProofs) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx InvokeScriptWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx *InvokeScriptWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx InvokeScriptWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx InvokeScriptWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

func (tx *InvokeScriptWithProofs) Clone() *InvokeScriptWithProofs {
	out := &InvokeScriptWithProofs{}
	if err := copier.Copy(out, tx); err != nil {
		panic(err.Error())
	}
	return out
}

// NewUnsignedInvokeScriptWithProofs creates new unsigned InvokeScriptWithProofs transaction.
func NewUnsignedInvokeScriptWithProofs(v byte, senderPK crypto.PublicKey, scriptRecipient Recipient, call FunctionCall, payments ScriptPayments, feeAsset OptionalAsset, fee, timestamp uint64) *InvokeScriptWithProofs {
	return &InvokeScriptWithProofs{
		Type:            InvokeScriptTransaction,
		Version:         v,
		SenderPK:        senderPK,
		ScriptRecipient: scriptRecipient,
		FunctionCall:    call,
		Payments:        payments,
		FeeAsset:        feeAsset,
		Fee:             fee,
		Timestamp:       timestamp,
	}
}

func (tx *InvokeScriptWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 1 || tx.Version > MaxInvokeScriptTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for InvokeScriptWithProofs", tx.Version)
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	if len(tx.FunctionCall.Arguments) > maxArguments {
		return tx, errors.New("too many arguments")
	}
	if len(tx.FunctionCall.Name) > maxFunctionNameBytes {
		return tx, errors.New("function name is too big")
	}
	for _, p := range tx.Payments {
		if p.Amount == 0 {
			return tx, errors.New("at least one payment has a non-positive amount")
		}
		if !validJVMLong(p.Amount) {
			return tx, errors.New("at least one payment has a too big amount")
		}
	}
	// we don't need to validate scheme here because scheme is included in binary representation of tx
	// so if scheme is invalid == signature is invalid
	return tx, nil
}

func (tx *InvokeScriptWithProofs) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	p := 0
	buf := make([]byte, invokeScriptWithProofsFixedBodyLen+tx.ScriptRecipient.BinarySize()+tx.FunctionCall.BinarySize()+tx.Payments.BinarySize()+tx.FeeAsset.BinarySize())
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	buf[p] = scheme
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	rb, err := tx.ScriptRecipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal InvokeScriptWithProofs body")
	}
	copy(buf[p:], rb)
	p += tx.ScriptRecipient.BinarySize()
	fcb, err := tx.FunctionCall.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(buf[p:], fcb)
	if len(fcb) != tx.FunctionCall.BinarySize() {
		panic("INVALID FUNCTION CALL")
	}
	p += len(fcb)
	psb, err := tx.Payments.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(buf[p:], psb)
	if len(psb) != tx.Payments.BinarySize() {
		panic("INVALID PAYMENTS")
	}
	p += len(psb)
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	fab, err := tx.FeeAsset.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(buf[p:], fab)
	p += len(fab)
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func (tx *InvokeScriptWithProofs) bodyUnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l < invokeScriptWithProofsFixedBodyLen {
		return errors.Errorf("not enough data for InvokeScriptWithProofs transaction, expected not less then %d, received %d", invokeScriptWithProofsFixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if unmarshalledScheme := data[2]; unmarshalledScheme != scheme {
		return errors.Errorf("scheme mismatch: got %d, want %d", unmarshalledScheme, scheme)
	}
	if tx.Type != InvokeScriptTransaction {
		return errors.Errorf("unexpected transaction type %d for InvokeScriptWithProofs transaction", tx.Type)
	}
	data = data[3:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var recipient Recipient
	err := recipient.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal InvokeScriptWithProofs transaction")
	}
	tx.ScriptRecipient = recipient
	data = data[tx.ScriptRecipient.BinarySize():]
	functionCall := FunctionCall{}
	err = functionCall.UnmarshalBinary(data)
	if err != nil {
		return err
	}
	tx.FunctionCall = functionCall
	data = data[functionCall.BinarySize():]
	payments := ScriptPayments{}
	err = payments.UnmarshalBinary(data)
	if err != nil {
		return err
	}
	tx.Payments = payments
	data = data[payments.BinarySize():]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	var asset OptionalAsset
	err = asset.UnmarshalBinary(data)
	if err != nil {
		return err
	}
	tx.FeeAsset = asset
	data = data[asset.BinarySize():]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

// Sign adds signature as a proof at first position.
func (tx *InvokeScriptWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeScriptWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeScriptWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeScriptWithProofs transaction")
	}
	tx.ID = &d
	return nil
}

// Verify checks that first proof is a valid signature.
func (tx *InvokeScriptWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of InvokeScriptWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

// MarshalBinary writes InvokeScriptWithProofs transaction to its byte representation.
func (tx *InvokeScriptWithProofs) MarshalBinary(scheme Scheme) ([]byte, error) {
	bb, err := tx.BodyMarshalBinary(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal InvokeScriptWithProofs transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal InvokeScriptWithProofs transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal InvokeScriptWithProofs transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

// UnmarshalBinary reads InvokeScriptWithProofs transaction from its binary representation.
func (tx *InvokeScriptWithProofs) UnmarshalBinary(data []byte, scheme Scheme) error {
	if len(data) > maxInvokeScriptWithProofsBinaryTransactionsBytes {
		return errors.New("invoke script transaction is too big")
	}
	if l := len(data); l < invokeScriptWithProofsMinLen {
		return errors.Errorf("not enough data for InvokeScriptWithProofs transaction, expected not less then %d, received %d", invokeScriptWithProofsMinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data, scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal InvokeScriptWithProofs transaction from bytes")
	}
	bl := invokeScriptWithProofsFixedBodyLen + tx.ScriptRecipient.BinarySize() + tx.FunctionCall.BinarySize() + tx.Payments.BinarySize() + tx.FeeAsset.BinarySize()
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal InvokeScriptWithProofs transaction from bytes")
	}
	tx.Proofs = &p
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *InvokeScriptWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *InvokeScriptWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	invokeScriptTx, ok := t.(*InvokeScriptWithProofs)
	if !ok {
		return errors.New("failed to convert result to InvokeScripV1")
	}
	protoPayloadSize, err := invokeScriptTx.protoPayloadSize()
	if err != nil {
		return err
	}
	if protoPayloadSize > maxInvokeScriptWithProofsProtobufPayloadBytes {
		return errors.New("invoke script transaction is too big")
	}
	*tx = *invokeScriptTx
	return nil
}

func (tx *InvokeScriptWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *InvokeScriptWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	invokeScriptTx, ok := t.(*InvokeScriptWithProofs)
	if !ok {
		return errors.New("failed to convert result to InvokeScriptWithProofs")
	}
	protoPayloadSize, err := invokeScriptTx.protoPayloadSize()
	if err != nil {
		return err
	}
	if protoPayloadSize > maxInvokeScriptWithProofsProtobufPayloadBytes {
		return errors.New("invoke script transaction is too big")
	}
	*tx = *invokeScriptTx
	return nil
}

func (tx *InvokeScriptWithProofs) protobufInvokeScriptTransactionData() (*g.InvokeScriptTransactionData, error) {
	fcBytes, err := tx.FunctionCall.MarshalBinary()
	if err != nil {
		return nil, err
	}
	payments := make([]*g.Amount, len(tx.Payments))
	for i := range tx.Payments {
		payments[i] = &g.Amount{AssetId: tx.Payments[i].Asset.ToID(), Amount: int64(tx.Payments[i].Amount)}
	}
	rcpProto, err := tx.ScriptRecipient.ToProtobuf()
	if err != nil {
		return nil, err
	}
	txData := &g.InvokeScriptTransactionData{
		DApp:         rcpProto,
		FunctionCall: fcBytes,
		Payments:     payments,
	}
	return txData, nil
}

func (tx *InvokeScriptWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	invokeTxData, err := tx.protobufInvokeScriptTransactionData()
	if err != nil {
		return nil, err
	}
	txData := &g.Transaction_InvokeScript{InvokeScript: invokeTxData}
	fee := &g.Amount{AssetId: tx.FeeAsset.ToID(), Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *InvokeScriptWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

func (tx *InvokeScriptWithProofs) protoPayloadSize() (int, error) {
	invokeTxData, err := tx.protobufInvokeScriptTransactionData()
	if err != nil {
		return 0, err
	}
	// use this method to calculate PB binary size of payload
	return invokeTxData.SizeVT(), err
}

type UpdateAssetInfoWithProofs struct {
	Type        TransactionType  `json:"type"`
	Version     byte             `json:"version,omitempty"`
	ID          *crypto.Digest   `json:"id,omitempty"`
	Proofs      *ProofsV1        `json:"proofs,omitempty"`
	SenderPK    crypto.PublicKey `json:"senderPublicKey"`
	AssetID     crypto.Digest    `json:"assetId"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	FeeAsset    OptionalAsset    `json:"feeAssetId"`
	Fee         uint64           `json:"fee"`
	Timestamp   uint64           `json:"timestamp,omitempty"`
}

func (tx UpdateAssetInfoWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx UpdateAssetInfoWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *UpdateAssetInfoWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx UpdateAssetInfoWithProofs) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx UpdateAssetInfoWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx UpdateAssetInfoWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx UpdateAssetInfoWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

func (tx *UpdateAssetInfoWithProofs) Validate(_ Scheme) (Transaction, error) {
	if tx.Version < 1 || tx.Version > MaxUpdateAssetInfoTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for UpdateAssetInfoWithProofs", tx.Version)
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	if l := len(tx.Name); l < MinAssetNameLen || l > MaxAssetNameLen {
		return tx, errs.NewInvalidName("incorrect number of bytes in the asset's name")
	}
	if l := len(tx.Description); l > MaxDescriptionLen {
		return tx, errs.NewTooBigArray("incorrect number of bytes in the asset's description")
	}
	return tx, nil
}

func (tx *UpdateAssetInfoWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *UpdateAssetInfoWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *UpdateAssetInfoWithProofs) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign UpdateAssetInfoWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign UpdateAssetInfoWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign UpdateAssetInfoWithProofs transaction")
	}
	return nil
}

func (tx *UpdateAssetInfoWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of UpdateAssetInfoWithProofs transaction")
	}
	return tx.Proofs.Verify(publicKey, b)
}

func NewUnsignedUpdateAssetInfoWithProofs(v byte, assetID crypto.Digest, senderPK crypto.PublicKey, name, description string, timestamp uint64, feeAsset OptionalAsset, fee uint64) *UpdateAssetInfoWithProofs {
	return &UpdateAssetInfoWithProofs{
		Type:        UpdateAssetInfoTransaction,
		Version:     v,
		SenderPK:    senderPK,
		AssetID:     assetID,
		Name:        name,
		Description: description,
		Timestamp:   timestamp,
		FeeAsset:    feeAsset,
		Fee:         fee,
	}
}

func (tx *UpdateAssetInfoWithProofs) MarshalBinary(Scheme) ([]byte, error) {
	return nil, errors.New("binary format is not defined for UpdateAssetInfoTransaction")
}

func (tx *UpdateAssetInfoWithProofs) UnmarshalBinary(_ []byte, _ Scheme) error {
	return errors.New("binary format is not defined for UpdateAssetInfoTransaction")
}

func (tx *UpdateAssetInfoWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	return nil, errors.New("binary format is not defined for UpdateAssetInfoTransaction")
}

func (tx *UpdateAssetInfoWithProofs) BinarySize() int {
	return 0
}

func (tx *UpdateAssetInfoWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *UpdateAssetInfoWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	updateAssetInfoTx, ok := t.(*UpdateAssetInfoWithProofs)
	if !ok {
		return errors.New("failed to convert result to UpdateAssetInfoWithProofs")
	}
	*tx = *updateAssetInfoTx
	return nil
}

func (tx *UpdateAssetInfoWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *UpdateAssetInfoWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	updateAssetInfoTx, ok := t.(*UpdateAssetInfoWithProofs)
	if !ok {
		return errors.New("failed to convert result to UpdateAssetInfoWithProofs")
	}
	*tx = *updateAssetInfoTx
	return nil
}

func (tx *UpdateAssetInfoWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	txData := &g.Transaction_UpdateAssetInfo{
		UpdateAssetInfo: &g.UpdateAssetInfoTransactionData{AssetId: tx.AssetID.Bytes(), Name: tx.Name, Description: tx.Description},
	}
	fee := &g.Amount{AssetId: tx.FeeAsset.ToID(), Amount: int64(tx.Fee)}
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *UpdateAssetInfoWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}

type InvokeExpressionTransactionWithProofs struct {
	ID         *crypto.Digest   `json:"id,omitempty"`
	Type       TransactionType  `json:"type"`
	Version    byte             `json:"version,omitempty"`
	SenderPK   crypto.PublicKey `json:"senderPublicKey"`
	Fee        uint64           `json:"fee"`
	FeeAsset   OptionalAsset    `json:"feeAssetId"`
	Timestamp  uint64           `json:"timestamp,omitempty"`
	Proofs     *ProofsV1        `json:"proofs,omitempty"`
	Expression B64Bytes         `json:"expression,omitempty"`
}

// NewUnsignedInvokeExpressionWithProofs creates new unsigned InvokeExpressionTransactionWithProofs transaction.
func NewUnsignedInvokeExpressionWithProofs(v byte, senderPK crypto.PublicKey, expression B64Bytes, feeAsset OptionalAsset, fee, timestamp uint64) *InvokeExpressionTransactionWithProofs {
	return &InvokeExpressionTransactionWithProofs{
		Type:       InvokeExpressionTransaction,
		Version:    v,
		SenderPK:   senderPK,
		FeeAsset:   feeAsset,
		Fee:        fee,
		Timestamp:  timestamp,
		Expression: expression,
	}
}

func (tx *InvokeExpressionTransactionWithProofs) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of InvokeExpressionTransactionWithProofs")
	}
	return tx.Proofs.Verify(publicKey, b)
}

func (tx InvokeExpressionTransactionWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx InvokeExpressionTransactionWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *InvokeExpressionTransactionWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx InvokeExpressionTransactionWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx InvokeExpressionTransactionWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx InvokeExpressionTransactionWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

func (tx *InvokeExpressionTransactionWithProofs) Validate(_ Scheme) (Transaction, error) {
	//TODO: Check specification on size check of InvokeExpression transaction
	if tx.Version < 1 || tx.Version > MaxInvokeScriptTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for InvokeExpressionWithProofs", tx.Version)
	}
	if l := len(tx.Expression); l > MaxContractScriptSizeV1V5 {
		return tx, errors.Errorf("size of the expression %d is exceeded limit %d", l, MaxContractScriptSizeV1V5)
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}
	return tx, nil
}

func (tx *InvokeExpressionTransactionWithProofs) GenerateID(scheme Scheme) error {
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

func (tx *InvokeExpressionTransactionWithProofs) Sign(scheme Scheme, sk crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeExpressionWithProofs transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = NewProofs()
	}
	err = tx.Proofs.Sign(sk, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeExpressionWithProofs transaction")
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeExpressionWithProofs transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *InvokeExpressionTransactionWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalBinary(Scheme) ([]byte, error) {
	panic("MarshalBinary is not implemented")
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalBinary([]byte, Scheme) error {
	panic("UnmarshalBinary is not implemented")
}

func (tx *InvokeExpressionTransactionWithProofs) BodyMarshalBinary(Scheme) ([]byte, error) {
	panic("BodyMarshalBinary is not implemented")
}

func (tx *InvokeExpressionTransactionWithProofs) BinarySize() int {
	panic("BinarySize is not implemented")
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	invokeExpressionTx, ok := t.(*InvokeExpressionTransactionWithProofs)
	if !ok {
		return errors.New("failed to convert result to InvokeExpressionTransactionWithProofs")
	}
	*tx = *invokeExpressionTx
	return nil
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	invokeExpressionTx, ok := t.(*InvokeExpressionTransactionWithProofs)
	if !ok {
		return errors.New("failed to convert result to InvokeExpressionTransactionWithProofs")
	}
	*tx = *invokeExpressionTx
	return nil
}
func (tx *InvokeExpressionTransactionWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	txData := &g.Transaction_InvokeExpression{InvokeExpression: &g.InvokeExpressionTransactionData{
		Expression: []byte(tx.Expression),
	}}
	fee := &g.Amount{AssetId: tx.FeeAsset.ToID(), Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}
func (tx *InvokeExpressionTransactionWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}
