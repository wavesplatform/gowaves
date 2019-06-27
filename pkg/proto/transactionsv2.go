package proto

import (
	"encoding/binary"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	issueV2FixedBodyLen       = 1 + 1 + 1 + crypto.PublicKeySize + 2 + 2 + 8 + 1 + 1 + 8 + 8 + 1
	issueV2MinBodyLen         = issueV2FixedBodyLen + 4 // 4 because of the shortest allowed Asset name of 4 bytes
	issueV2MinLen             = 1 + issueV2MinBodyLen + proofsMinLen
	transferV2FixedBodyLen    = 1 + 1 + transferLen
	transferV2MinLen          = 1 + transferV2FixedBodyLen + proofsMinLen
	reissueV2BodyLen          = 3 + reissueLen
	reissueV2MinLen           = 1 + reissueV2BodyLen + proofsMinLen
	burnV2BodyLen             = 1 + 1 + 1 + burnLen
	burnV2Len                 = 1 + burnV2BodyLen + proofsMinLen
	exchangeV2FixedBodyLen    = 1 + 1 + 1 + 4 + 4 + 8 + 8 + 8 + 8 + 8 + 8
	exchangeV2MinLen          = exchangeV2FixedBodyLen + orderV2MinLen + orderV2MinLen + proofsMinLen
	leaseV2BodyLen            = 1 + 1 + 1 + leaseLen
	leaseV2MinLen             = leaseV2BodyLen + proofsMinLen
	leaseCancelV2BodyLen      = 1 + 1 + 1 + leaseCancelLen
	leaseCancelV2MinLen       = 1 + leaseCancelV2BodyLen + proofsMinLen
	createAliasV2FixedBodyLen = 1 + 1 + createAliasLen
	createAliasV2MinLen       = 1 + createAliasV2FixedBodyLen + proofsMinLen
)

// IssueV2 is a transaction to issue new asset, second version.
type IssueV2 struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version"`
	ChainID byte            `json:"-"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Script  Script          `json:"script"`
	Issue
}

func (tx IssueV2) GetTypeVersion() TransactionTypeVersion {
	return TransactionTypeVersion{tx.Type, tx.Version}
}

func (tx *IssueV2) GenerateID() {
	if tx.ID == nil {
		body, err := tx.bodyMarshalBinary()
		if err != nil {
			panic(err.Error())
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}

}

func (tx IssueV2) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedIssueV2 creates a new IssueV2 transaction with empty Proofs.
func NewUnsignedIssueV2(chainID byte, senderPK crypto.PublicKey, name, description string, quantity uint64, decimals byte, reissuable bool, script []byte, timestamp, fee uint64) *IssueV2 {
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
	return &IssueV2{Type: IssueTransaction, Version: 2, ChainID: chainID, Script: script, Issue: i}
}

func (tx IssueV2) Valid() (bool, error) {
	ok, err := tx.Issue.Valid()
	if !ok {
		return false, err
	}
	//TODO: add script and scheme validations
	return true, nil
}

//NonEmptyScript returns true if the script of the transaction is not empty, otherwise false.
func (tx *IssueV2) NonEmptyScript() bool {
	return len(tx.Script) != 0
}

func (tx *IssueV2) bodyMarshalBinary() ([]byte, error) {
	var p int
	nl := len(tx.Name)
	dl := len(tx.Description)
	sl := len(tx.Script)
	if sl > 0 {
		sl += 2
	}
	buf := make([]byte, issueV2FixedBodyLen+nl+dl+sl)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = tx.ChainID
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
		PutBytesWithUInt16Len(buf[p:], tx.Script)
	}
	return buf, nil
}

func (tx *IssueV2) bodyUnmarshalBinary(data []byte) error {
	const message = "failed to unmarshal field %q of IssueV2 transaction"
	if l := len(data); l < issueV2MinBodyLen {
		return errors.Errorf("not enough data for IssueV2 transaction %d, expected not less then %d", l, issueV2MinBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != IssueTransaction {
		return errors.Errorf("unexpected transaction type %d for IssueV2 transaction", tx.Type)
	}
	tx.Version = data[1]
	tx.ChainID = data[2]
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

//Sign calculates transaction signature using given secret key.
func (tx *IssueV2) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueV2 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueV2 transaction")
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueV2 transaction")
	}
	tx.ID = &d
	return nil
}

//Verify checks that the transaction signature is valid for given public key.
func (tx *IssueV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of IssueV2 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary converts transaction to its binary representation.
func (tx *IssueV2) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal IssueV2 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal IssueV2 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal IssueV2 transaction to bytes")
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

//UnmarshalBinary reads transaction from its binary representation.
func (tx *IssueV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < issueV2MinLen {
		return errors.Errorf("not enough data for IssueV2 transaction, expected not less then %d, received %d", issueV2MinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d for IssueV2 transaction, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IssueV2 transaction")
	}
	sl := len(tx.Script)
	if sl > 0 {
		sl += 2
	}
	bl := issueV2FixedBodyLen + len(tx.Name) + len(tx.Description) + sl
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IssueV2 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IssueV2 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//TransferV2 transaction to transfer any token from one account to another. Version 2.
type TransferV2 struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Transfer
}

func (tx TransferV2) GetTypeVersion() TransactionTypeVersion {
	return TransactionTypeVersion{tx.Type, tx.Version}
}

func (tx *TransferV2) GenerateID() {
	if tx.ID == nil {
		body, err := tx.BodyMarshalBinary()
		if err != nil {
			panic(err.Error())
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}

}

func (tx TransferV2) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedTransferV2 creates new TransferV2 transaction without proofs and ID.
func NewUnsignedTransferV2(senderPK crypto.PublicKey, amountAsset, feeAsset OptionalAsset, timestamp, amount, fee uint64, recipient Recipient, attachment string) *TransferV2 {
	t := Transfer{
		SenderPK:    senderPK,
		Recipient:   recipient,
		AmountAsset: amountAsset,
		Amount:      amount,
		FeeAsset:    feeAsset,
		Fee:         fee,
		Timestamp:   timestamp,
		Attachment:  Attachment(attachment),
	}
	return &TransferV2{Type: TransferTransaction, Version: 2, Transfer: t}
}

func (tx TransferV2) Valid() (bool, error) {
	ok, err := tx.Transfer.Valid()
	if !ok {
		return false, err
	}
	//TODO: validate script and scheme
	return true, nil
}

func (tx *TransferV2) BodyMarshalBinary() ([]byte, error) {
	b, err := tx.Transfer.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferV2 body")
	}
	buf := make([]byte, 2+len(b))
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	copy(buf[2:], b)
	return buf, nil
}

func (tx *TransferV2) BodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < transferV2FixedBodyLen {
		return errors.Errorf("%d bytes is not enough for TransferV2 transaction, expected not less then %d bytes", l, transferV2FixedBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != TransferTransaction {
		return errors.Errorf("unexpected transaction type %d for TransferV2 transaction", tx.Type)
	}
	tx.Version = data[1]
	if v := tx.Version; v != 2 {
		return errors.Errorf("unexpected version %d for TransferV2 transaction, expected 2", v)
	}
	var t Transfer
	err := t.unmarshalBinary(data[2:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV2 body from bytes")
	}
	tx.Transfer = t
	return nil
}

//Sign adds signature as a proof at first position.
func (tx *TransferV2) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.BodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferV2 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferV2 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferV2 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *TransferV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.BodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of TransferV2 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes TransferV2 transaction to its bytes representation.
func (tx *TransferV2) MarshalBinary() ([]byte, error) {
	bb, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferV2 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal TransferV2 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferV2 transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads TransferV2 from its bytes representation.
func (tx *TransferV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < transferV2MinLen {
		return errors.Errorf("not enough data for TransferV2 transaction, expected not less then %d, received %d", transferV2MinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.BodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV2 transaction from bytes")
	}
	aal := 0
	if tx.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	fal := 0
	if tx.FeeAsset.Present {
		fal += crypto.DigestSize
	}
	atl := len(tx.Attachment)
	rl := tx.Recipient.len
	bl := transferV2FixedBodyLen + aal + fal + atl + rl
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV2 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV2 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//ReissueV2 same as ReissueV1 but version 2 with Proofs.
type ReissueV2 struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ChainID byte            `json:"-"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Reissue
}

func (tx ReissueV2) GetTypeVersion() TransactionTypeVersion {
	return TransactionTypeVersion{tx.Type, tx.Version}
}

func (tx *ReissueV2) GenerateID() {
	if tx.ID == nil {
		body, err := tx.bodyMarshalBinary()
		if err != nil {
			panic(err.Error())
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}

}

func (tx ReissueV2) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedReissueV2 creates new ReissueV2 transaction without signature and ID.
func NewUnsignedReissueV2(chainID byte, senderPK crypto.PublicKey, assetID crypto.Digest, quantity uint64, reissuable bool, timestamp, fee uint64) *ReissueV2 {
	r := Reissue{
		SenderPK:   senderPK,
		AssetID:    assetID,
		Quantity:   quantity,
		Reissuable: reissuable,
		Fee:        fee,
		Timestamp:  timestamp,
	}
	return &ReissueV2{Type: ReissueTransaction, Version: 2, ChainID: chainID, Reissue: r}
}

func (tx ReissueV2) Valid() (bool, error) {
	ok, err := tx.Reissue.Valid()
	if !ok {
		return false, err
	}
	//TODO: add current blockchain scheme validation
	return true, nil
}

func (tx *ReissueV2) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, reissueV2BodyLen)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = tx.ChainID
	b, err := tx.Reissue.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueV2 body")
	}
	copy(buf[3:], b)
	return buf, nil
}

func (tx *ReissueV2) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < reissueV2BodyLen {
		return errors.Errorf("%d bytes is not enough for ReissueV2 transaction, expected not less then %d bytes", l, reissueV2BodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != ReissueTransaction {
		return errors.Errorf("unexpected transaction type %d for ReissueV2 transaction", tx.Type)
	}
	tx.Version = data[1]
	if v := tx.Version; v != 2 {
		return errors.Errorf("unexpected version %d for ReissueV2 transaction, expected 2", v)
	}
	tx.ChainID = data[2]
	var r Reissue
	err := r.unmarshalBinary(data[3:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueV2 body from bytes")
	}
	tx.Reissue = r
	return nil
}

//Sign adds signature as a proof at first position.
func (tx *ReissueV2) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueV2 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueV2 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueV2 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *ReissueV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of ReissueV2 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes ReissueV2 transaction to its bytes representation.
func (tx *ReissueV2) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueV2 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal ReissueV2 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueV2 transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads ReissueV2 from its bytes representation.
func (tx *ReissueV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < reissueV2MinLen {
		return errors.Errorf("not enough data for ReissueV2 transaction, expected not less then %d, received %d", reissueV2MinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueV2 transaction from bytes")
	}
	bb := data[:reissueV2BodyLen]
	data = data[reissueV2BodyLen:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueV2 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueV2 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//BurnV2 same as BurnV1 but version 2 with Proofs.
type BurnV2 struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ChainID byte            `json:"-"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Burn
}

func (tx BurnV2) GetTypeVersion() TransactionTypeVersion {
	return TransactionTypeVersion{tx.Type, tx.Version}
}

func (tx *BurnV2) GenerateID() {
	if tx.ID == nil {
		body, err := tx.bodyMarshalBinary()
		if err != nil {
			panic(err.Error())
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}

}

func (tx BurnV2) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedBurnV2 creates new BurnV2 transaction without proofs and ID.
func NewUnsignedBurnV2(chainID byte, senderPK crypto.PublicKey, assetID crypto.Digest, amount, timestamp, fee uint64) *BurnV2 {
	b := Burn{
		SenderPK:  senderPK,
		AssetID:   assetID,
		Amount:    amount,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &BurnV2{Type: BurnTransaction, Version: 2, ChainID: chainID, Burn: b}
}

func (tx BurnV2) Valid() (bool, error) {
	ok, err := tx.Burn.Valid()
	if !ok {
		return false, err
	}
	//TODO: check current blockchain scheme
	return true, nil
}

func (tx *BurnV2) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, burnV2BodyLen)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = tx.ChainID
	b, err := tx.Burn.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnV2 body")
	}
	copy(buf[3:], b)
	return buf, nil
}

func (tx *BurnV2) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < burnV2BodyLen {
		return errors.Errorf("%d bytes is not enough for BurnV2 transaction, expected not less then %d bytes", l, burnV2BodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != BurnTransaction {
		return errors.Errorf("unexpected transaction type %d for BurnV2 transaction", tx.Type)
	}
	tx.Version = data[1]
	if v := tx.Version; v != 2 {
		return errors.Errorf("unexpected version %d for BurnV2 transaction, expected 2", v)
	}
	tx.ChainID = data[2]
	var b Burn
	err := b.unmarshalBinary(data[3:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnV2 body from bytes")
	}
	tx.Burn = b
	return nil
}

//Sign adds signature as a proof at first position.
func (tx *BurnV2) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnV2 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnV2 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnV2 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *BurnV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of BurnV2 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes BurnV2 transaction to its bytes representation.
func (tx *BurnV2) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnV2 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal BurnV2 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnV2 transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads BurnV2 from its bytes representation.
func (tx *BurnV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < burnV2Len {
		return errors.Errorf("not enough data for BurnV2 transaction, expected not less then %d, received %d", burnV2BodyLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnV2 transaction from bytes")
	}
	bb := data[:burnV2BodyLen]
	data = data[burnV2BodyLen:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnV2 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnV2 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//ExchangeV2 is a transaction to store settlement on blockchain.
type ExchangeV2 struct {
	Type           TransactionType  `json:"type"`
	Version        byte             `json:"version,omitempty"`
	ID             *crypto.Digest   `json:"id,omitempty"`
	Proofs         *ProofsV1        `json:"proofs,omitempty"`
	SenderPK       crypto.PublicKey `json:"senderPublicKey"`
	BuyOrder       Order            `json:"order1"`
	SellOrder      Order            `json:"order2"`
	Price          uint64           `json:"price"`
	Amount         uint64           `json:"amount"`
	BuyMatcherFee  uint64           `json:"buyMatcherFee"`
	SellMatcherFee uint64           `json:"sellMatcherFee"`
	Fee            uint64           `json:"fee"`
	Timestamp      uint64           `json:"timestamp,omitempty"`
}

func (tx ExchangeV2) GetTypeVersion() TransactionTypeVersion {
	return TransactionTypeVersion{tx.Type, tx.Version}
}

func (tx *ExchangeV2) GenerateID() {
	if tx.ID == nil {
		body, err := tx.bodyMarshalBinary()
		if err != nil {
			panic(err.Error())
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}

}

func (tx ExchangeV2) GetID() []byte {
	return tx.ID.Bytes()
}

func (tx ExchangeV2) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx ExchangeV2) GetBuyOrder() (OrderBody, error) {
	return OrderToOrderBody(tx.BuyOrder)
}

func (tx ExchangeV2) GetSellOrder() (OrderBody, error) {
	return OrderToOrderBody(tx.SellOrder)
}

func (tx ExchangeV2) GetPrice() uint64 {
	return tx.Price
}

func (tx ExchangeV2) GetAmount() uint64 {
	return tx.Amount
}

func (tx ExchangeV2) GetBuyMatcherFee() uint64 {
	return tx.BuyMatcherFee
}

func (tx ExchangeV2) GetSellMatcherFee() uint64 {
	return tx.SellMatcherFee
}
func (tx ExchangeV2) GetFee() uint64 {
	return tx.Fee
}

func (tx ExchangeV2) GetTimestamp() uint64 {
	return tx.Timestamp
}

func NewUnsignedExchangeV2(buy, sell Order, price, amount, buyMatcherFee, sellMatcherFee, fee, timestamp uint64) *ExchangeV2 {
	return &ExchangeV2{
		Type:           ExchangeTransaction,
		Version:        2,
		SenderPK:       buy.GetMatcherPK(),
		BuyOrder:       buy,
		SellOrder:      sell,
		Price:          price,
		Amount:         amount,
		BuyMatcherFee:  buyMatcherFee,
		SellMatcherFee: sellMatcherFee,
		Fee:            fee,
		Timestamp:      timestamp,
	}
}

func (tx ExchangeV2) Valid() (bool, error) {
	ok, err := tx.BuyOrder.Valid()
	if !ok {
		return false, errors.Wrap(err, "invalid buy order")
	}
	ok, err = tx.SellOrder.Valid()
	if !ok {
		return false, errors.Wrap(err, "invalid sell order")
	}
	if tx.BuyOrder.GetOrderType() != Buy {
		return false, errors.New("incorrect order type of buy order")
	}
	if tx.SellOrder.GetOrderType() != Sell {
		return false, errors.New("incorrect order type of sell order")
	}
	if tx.SellOrder.GetMatcherPK() != tx.BuyOrder.GetMatcherPK() {
		return false, errors.New("unmatched matcher's public keys")
	}
	if tx.SellOrder.GetAssetPair() != tx.BuyOrder.GetAssetPair() {
		return false, errors.New("different asset pairs")
	}
	if tx.Amount <= 0 {
		return false, errors.New("amount should be positive")
	}
	if !validJVMLong(tx.Amount) {
		return false, errors.New("amount is too big")
	}
	if tx.Price <= 0 {
		return false, errors.New("price should be positive")
	}
	if !validJVMLong(tx.Price) {
		return false, errors.New("price is too big")
	}
	if tx.Price > tx.BuyOrder.GetPrice() || tx.Price < tx.SellOrder.GetPrice() {
		return false, errors.New("invalid price")
	}
	if tx.Fee <= 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return false, errors.New("fee is too big")
	}
	if tx.BuyMatcherFee <= 0 {
		return false, errors.New("buy matcher's fee should be positive")
	}
	if !validJVMLong(tx.BuyMatcherFee) {
		return false, errors.New("buy matcher's fee is too big")
	}
	if tx.SellMatcherFee <= 0 {
		return false, errors.New("sell matcher's fee should be positive")
	}
	if !validJVMLong(tx.SellMatcherFee) {
		return false, errors.New("sell matcher's fee is too big")
	}
	if tx.BuyOrder.GetExpiration() < tx.Timestamp {
		return false, errors.New("invalid buy order expiration")
	}
	if tx.BuyOrder.GetExpiration()-tx.Timestamp > MaxOrderTTL {
		return false, errors.New("buy order expiration should be earlier than 30 days")
	}
	if tx.SellOrder.GetExpiration() < tx.Timestamp {
		return false, errors.New("invalid sell order expiration")
	}
	if tx.SellOrder.GetExpiration()-tx.Timestamp > MaxOrderTTL {
		return false, errors.New("sell order expiration should be earlier than 30 days")
	}
	return true, nil
}

func (tx *ExchangeV2) marshalAsOrderV1(order Order) ([]byte, error) {
	o, ok := order.(OrderV1)
	if !ok {
		return nil, errors.New("failed to cast an order with version 1 to OrderV1")
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

func (tx *ExchangeV2) marshalAsOrderV2(order Order) ([]byte, error) {
	o, ok := order.(OrderV2)
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

func (tx *ExchangeV2) marshalAsOrderV3(order Order) ([]byte, error) {
	o, ok := order.(OrderV3)
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

func (tx *ExchangeV2) bodyMarshalBinary() ([]byte, error) {
	var bob []byte
	var sob []byte
	var err error
	switch tx.BuyOrder.GetVersion() {
	case 1:
		bob, err = tx.marshalAsOrderV1(tx.BuyOrder)
	case 2:
		bob, err = tx.marshalAsOrderV2(tx.BuyOrder)
	case 3:
		bob, err = tx.marshalAsOrderV3(tx.BuyOrder)
	default:
		err = errors.Errorf("invalid BuyOrder version %d", tx.BuyOrder.GetVersion())
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal buy order to bytes")
	}
	bol := uint32(len(bob))
	switch tx.SellOrder.GetVersion() {
	case 1:
		sob, err = tx.marshalAsOrderV1(tx.SellOrder)
	case 2:
		sob, err = tx.marshalAsOrderV2(tx.SellOrder)
	case 3:
		sob, err = tx.marshalAsOrderV3(tx.SellOrder)
	default:
		err = errors.Errorf("invalid SellOrder version %d", tx.SellOrder.GetVersion())
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal sell order to bytes")
	}
	sol := uint32(len(sob))
	var p uint32
	buf := make([]byte, exchangeV2FixedBodyLen+(bol-4)+(sol-4))
	buf[0] = 0
	buf[1] = byte(tx.Type)
	buf[2] = tx.Version
	p += 3
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

func (tx *ExchangeV2) unmarshalOrder(data []byte) (int, *Order, error) {
	var r Order
	n := 0
	ol := binary.BigEndian.Uint32(data)
	n += 4
	switch data[n] {
	case 1:
		n++
		var o OrderV1
		err := o.UnmarshalBinary(data[n:])
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to unmarshal OrderV1")
		}
		n += int(ol)
		r = o
	case 2:
		var o OrderV2
		err := o.UnmarshalBinary(data[n:])
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to unmarshal OrderV2")
		}
		n += int(ol)
		r = o
	default:
		return 0, nil, errors.Errorf("unexpected order version %d", data[n])
	}
	return n, &r, nil
}

func (tx *ExchangeV2) bodyUnmarshalBinary(data []byte) (int, error) {
	n := 0
	if l := len(data); l < exchangeV2FixedBodyLen {
		return 0, errors.Errorf("not enough data for ExchangeV2 body, expected not less then %d, received %d", exchangeV2FixedBodyLen, l)
	}
	if v := data[n]; v != 0 {
		return 0, errors.Errorf("unexpected first byte %d of ExchangeV2 body, expected 0", v)
	}
	n++
	tx.Type = TransactionType(data[n])
	if tx.Type != ExchangeTransaction {
		return 0, errors.Errorf("unexpected transaction type %d for ExchangeV2 transaction", tx.Type)
	}
	n++
	tx.Version = data[n]
	if tx.Version != 2 {
		return 0, errors.Errorf("unexpected transaction version %d for ExchangeV2 transaction", tx.Version)
	}
	n++
	l, o, err := tx.unmarshalOrder(data[n:])
	if err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal buy order")
	}
	tx.BuyOrder = *o
	n += l
	l, o, err = tx.unmarshalOrder(data[n:])
	if err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal sell order")
	}
	tx.SellOrder = *o
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
	tx.SenderPK = tx.BuyOrder.GetMatcherPK()
	return n, nil
}

//Sign calculates transaction signature using given secret key.
func (tx *ExchangeV2) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeV2 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeV2 transaction")
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeV2 transaction")
	}
	tx.ID = &d
	return nil
}

//Verify checks that the transaction signature is valid for given public key.
func (tx *ExchangeV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of ExchangeV2 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary saves the transaction to its binary representation.
func (tx *ExchangeV2) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ExchangeV2 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal ExchangeV2 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ExchangeV2 transaction to bytes")
	}
	buf := make([]byte, bl+len(pb))
	copy(buf, bb)
	copy(buf[bl:], pb)
	return buf, nil
}

//UnmarshalBinary loads the transaction from its binary representation.
func (tx *ExchangeV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < exchangeV2MinLen {
		return errors.Errorf("not enough data for ExchangeV2 transaction, expected not less then %d, received %d", exchangeV2MinLen, l)
	}
	bl, err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeV2 transaction from bytes")
	}
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeV2 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeV2 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

func (tx *ExchangeV2) UnmarshalJSON(data []byte) error {
	guessOrderVersion := func(version byte) Order {
		var r Order
		switch version {
		case 2:
			r = &OrderV2{}
		default:
			r = &OrderV1{}
		}
		return r
	}

	orderVersions := struct {
		BuyOrderVersion  OrderVersion `json:"order1"`
		SellOrderVersion OrderVersion `json:"order2"`
	}{}
	if err := json.Unmarshal(data, &orderVersions); err != nil {
		return errors.Wrap(err, "failed to unmarshal orders versions of ExchangeV2 transaction from JSON")
	}
	tmp := struct {
		Type           TransactionType  `json:"type"`
		Version        byte             `json:"version,omitempty"`
		ID             *crypto.Digest   `json:"id,omitempty"`
		Proofs         *ProofsV1        `json:"proofs,omitempty"`
		SenderPK       crypto.PublicKey `json:"senderPublicKey"`
		BuyOrder       Order            `json:"order1"`
		SellOrder      Order            `json:"order2"`
		Price          uint64           `json:"price"`
		Amount         uint64           `json:"amount"`
		BuyMatcherFee  uint64           `json:"buyMatcherFee"`
		SellMatcherFee uint64           `json:"sellMatcherFee"`
		Fee            uint64           `json:"fee"`
		Timestamp      uint64           `json:"timestamp,omitempty"`
	}{}
	tmp.BuyOrder = guessOrderVersion(orderVersions.BuyOrderVersion.Version)
	tmp.SellOrder = guessOrderVersion(orderVersions.SellOrderVersion.Version)

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeV2 from JSON")
	}
	tx.Type = tmp.Type
	tx.Version = tmp.Version
	tx.ID = tmp.ID
	tx.Proofs = tmp.Proofs
	tx.SenderPK = tmp.SenderPK
	tx.BuyOrder = tmp.BuyOrder
	tx.SellOrder = tmp.SellOrder
	tx.Price = tmp.Price
	tx.Amount = tmp.Amount
	tx.BuyMatcherFee = tmp.BuyMatcherFee
	tx.SellMatcherFee = tmp.SellMatcherFee
	tx.Fee = tmp.Fee
	tx.Timestamp = tmp.Timestamp
	return nil
}

//LeaseV2 is a second version of the LeaseV1 transaction.
type LeaseV2 struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	Lease
}

func (tx LeaseV2) GetTypeVersion() TransactionTypeVersion {
	return TransactionTypeVersion{tx.Type, tx.Version}
}

func (tx *LeaseV2) GenerateID() {
	if tx.ID == nil {
		body, err := tx.bodyMarshalBinary()
		if err != nil {
			panic(err.Error())
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}

}

func (tx LeaseV2) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedLeaseV2 creates new LeaseV1 transaction without signature and ID set.
func NewUnsignedLeaseV2(senderPK crypto.PublicKey, recipient Recipient, amount, fee, timestamp uint64) *LeaseV2 {
	l := Lease{
		SenderPK:  senderPK,
		Recipient: recipient,
		Amount:    amount,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &LeaseV2{Type: LeaseTransaction, Version: 2, Lease: l}
}

func (tx *LeaseV2) bodyMarshalBinary() ([]byte, error) {
	rl := tx.Recipient.len
	buf := make([]byte, leaseV2BodyLen+rl)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = 0 //Always zero, reserved for future extension of leasing assets.
	b, err := tx.Lease.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseV1 transaction to bytes")
	}
	copy(buf[3:], b)
	return buf, nil
}

func (tx *LeaseV2) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseV2BodyLen {
		return errors.Errorf("not enough data for LeaseV2 transaction, expected not less then %d, received %d", leaseV2BodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != LeaseTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseV2 transaction", tx.Type)
	}
	tx.Version = data[1]
	if tx.Version != 2 {
		return errors.Errorf("unexpected version %d for LeaseV2 transaction, expected 2", tx.Version)
	}
	var l Lease
	err := l.unmarshalBinary(data[3:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseV2 transaction from bytes")
	}
	tx.Lease = l
	return nil
}

//Sign adds signature as a proof at first position.
func (tx *LeaseV2) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseV2 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseV2 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseV2 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *LeaseV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of LeaseV2 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary saves the transaction to its binary representation.
func (tx *LeaseV2) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseV2 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal LeaseV2 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseV2 transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads the transaction from bytes slice.
func (tx *LeaseV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseV2MinLen {
		return errors.Errorf("not enough data for LeaseV2 transaction, expected not less then %d, received %d", leaseV2MinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseV2 transaction from bytes")
	}
	bl := leaseV2BodyLen + tx.Recipient.len
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseV2 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseV2 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//LeaseCancelV2 same as LeaseCancelV1 but with proofs.
type LeaseCancelV2 struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ChainID byte            `json:"-"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	LeaseCancel
}

func (tx LeaseCancelV2) GetTypeVersion() TransactionTypeVersion {
	return TransactionTypeVersion{tx.Type, tx.Version}
}

func (tx *LeaseCancelV2) GenerateID() {
	if tx.ID == nil {
		body, err := tx.bodyMarshalBinary()
		if err != nil {
			panic(err.Error())
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}

}

func (tx LeaseCancelV2) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedLeaseCancelV2 creates new LeaseCancelV2 transaction structure without a signature and an ID.
func NewUnsignedLeaseCancelV2(chainID byte, senderPK crypto.PublicKey, leaseID crypto.Digest, fee, timestamp uint64) *LeaseCancelV2 {
	lc := LeaseCancel{
		SenderPK:  senderPK,
		LeaseID:   leaseID,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &LeaseCancelV2{Type: LeaseCancelTransaction, Version: 2, ChainID: chainID, LeaseCancel: lc}
}

func (tx LeaseCancelV2) Valid() (bool, error) {
	ok, err := tx.LeaseCancel.Valid()
	if !ok {
		return false, err
	}
	//TODO: add scheme validation
	return true, nil
}

func (tx *LeaseCancelV2) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, leaseCancelV2BodyLen)
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	buf[2] = tx.ChainID
	b, err := tx.LeaseCancel.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelV2 to bytes")
	}
	copy(buf[3:], b)
	return buf, nil
}

func (tx *LeaseCancelV2) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseCancelV2BodyLen {
		return errors.Errorf("not enough data for LeaseCancelV2 transaction, expected not less then %d, received %d", leaseCancelV2BodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != LeaseCancelTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseCancelV2 transaction", tx.Type)

	}
	tx.Version = data[1]
	if tx.Version != 2 {
		return errors.Errorf("unexpected version %d for LeaseCancelV2, expected 2", tx.Version)
	}
	tx.ChainID = data[2]
	var lc LeaseCancel
	err := lc.unmarshalBinary(data[3:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelV2 from bytes")
	}
	tx.LeaseCancel = lc
	return nil
}

//Sign adds signature as a proof at first position.
func (tx *LeaseCancelV2) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelV2 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelV2 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelV2 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *LeaseCancelV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of LeaseCancelV2 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary saves the transaction to its binary representation.
func (tx *LeaseCancelV2) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelV2 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal LeaseCancelV2 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelV2 transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads the transaction from bytes slice.
func (tx *LeaseCancelV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseCancelV2MinLen {
		return errors.Errorf("not enough data for LeaseCancelV2 transaction, expected not less then %d, received %d", leaseCancelV2MinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelV2 transaction from bytes")
	}
	bb := data[:leaseCancelV2BodyLen]
	data = data[leaseCancelV2BodyLen:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelV2 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelV2 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

type CreateAliasV2 struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
	ID      *crypto.Digest  `json:"id,omitempty"`
	Proofs  *ProofsV1       `json:"proofs,omitempty"`
	CreateAlias
}

func (tx CreateAliasV2) GetTypeVersion() TransactionTypeVersion {
	return TransactionTypeVersion{tx.Type, tx.Version}
}

func (tx *CreateAliasV2) GenerateID() {
	if tx.ID == nil {
		id, err := tx.CreateAlias.id()
		if err != nil {
			panic(err.Error())
		}
		tx.ID = id
	}

}

func (tx CreateAliasV2) GetID() []byte {
	return tx.ID.Bytes()
}

func NewUnsignedCreateAliasV2(senderPK crypto.PublicKey, alias Alias, fee, timestamp uint64) *CreateAliasV2 {
	ca := CreateAlias{
		SenderPK:  senderPK,
		Alias:     alias,
		Fee:       fee,
		Timestamp: timestamp,
	}
	return &CreateAliasV2{Type: CreateAliasTransaction, Version: 2, CreateAlias: ca}
}

func (tx *CreateAliasV2) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, createAliasV2FixedBodyLen+len(tx.Alias.Alias))
	buf[0] = byte(tx.Type)
	buf[1] = tx.Version
	b, err := tx.CreateAlias.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasV2 transaction body to bytes")
	}
	copy(buf[2:], b)
	return buf, nil
}

func (tx *CreateAliasV2) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < createAliasV2FixedBodyLen {
		return errors.Errorf("not enough data for CreateAliasV2 transaction, expected not less then %d, received %d", createAliasV2FixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != CreateAliasTransaction {
		return errors.Errorf("unexpected transaction type %d for CreateAliasV2 transaction", tx.Type)
	}
	tx.Version = data[1]
	if tx.Version != 2 {
		return errors.Errorf("unexpected version %d for CreateAliasV2 transaction", tx.Version)
	}
	var ca CreateAlias
	err := ca.unmarshalBinary(data[2:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV2 transaction from bytes")
	}
	tx.CreateAlias = ca
	return nil
}

//Sign adds signature as a proof at first position.
func (tx *CreateAliasV2) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasV2 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasV2 transaction")
	}
	tx.ID, err = tx.CreateAlias.id()
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasV2 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *CreateAliasV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of CreateAliasV2 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary saves the transaction to its binary representation.
func (tx *CreateAliasV2) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasV2 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal CreateAliasV2 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasV2 transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads the transaction from bytes slice.
func (tx *CreateAliasV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < createAliasV2MinLen {
		return errors.Errorf("not enough data for CreateAliasV2 transaction, expected not less then %d, received %d", createAliasV2MinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV2 transaction from bytes")
	}
	data = data[createAliasV2FixedBodyLen+len(tx.Alias.Alias):]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV2 transaction from bytes")
	}
	tx.Proofs = &p
	tx.ID, err = tx.CreateAlias.id()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV2 transaction from bytes")
	}
	return nil
}

func (tx *CreateAliasV2) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Type      TransactionType  `json:"type"`
		Version   byte             `json:"version,omitempty"`
		ID        *crypto.Digest   `json:"id,omitempty"`
		Proofs    *ProofsV1        `json:"proofs,omitempty"`
		SenderPK  crypto.PublicKey `json:"senderPublicKey"`
		Alias     string           `json:"alias"`
		Fee       uint64           `json:"fee"`
		Timestamp uint64           `json:"timestamp,omitempty"`
	}{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV1 from JSON")
	}
	tx.Type = tmp.Type
	tx.Version = tmp.Version
	tx.ID = tmp.ID
	tx.Proofs = tmp.Proofs
	tx.SenderPK = tmp.SenderPK
	tx.Alias = Alias{aliasVersion, TestNetScheme, tmp.Alias}
	tx.Fee = tmp.Fee
	tx.Timestamp = tmp.Timestamp
	return nil
}
