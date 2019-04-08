package proto

import (
	"encoding/binary"
	"reflect"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

//TransactionType
type TransactionType byte

//All transaction types supported.
const (
	GenesisTransaction TransactionType = iota + 1
	PaymentTransaction
	IssueTransaction
	TransferTransaction
	ReissueTransaction
	BurnTransaction
	ExchangeTransaction
	LeaseTransaction
	LeaseCancelTransaction
	CreateAliasTransaction
	MassTransferTransaction
	DataTransaction
	SetScriptTransaction
	SponsorshipTransaction
	SetAssetScriptTransaction
)

const (
	maxAttachmentLengthBytes = 140
	maxDescriptionLen        = 1000
	maxAssetNameLen          = 16
	minAssetNameLen          = 4
	maxDecimals              = 8

	genesisBodyLen = 1 + 8 + AddressSize + 8
	paymentBodyLen = 1 + 8 + crypto.PublicKeySize + AddressSize + 8 + 8
	transferLen    = crypto.PublicKeySize + 1 + 1 + 8 + 8 + 8 + 2
	reissueLen     = crypto.PublicKeySize + crypto.DigestSize + 8 + 1 + 8 + 8
	burnLen        = crypto.PublicKeySize + crypto.DigestSize + 8 + 8 + 8
	leaseLen       = crypto.PublicKeySize + 8 + 8 + 8
	leaseCancelLen = crypto.PublicKeySize + 8 + 8 + crypto.DigestSize
	createAliasLen = crypto.PublicKeySize + 2 + 8 + 8 + aliasFixedSize
)

var (
	bytesToTransactionsV2 = map[TransactionType]reflect.Type{
		IssueTransaction:          reflect.TypeOf(IssueV2{}),
		TransferTransaction:       reflect.TypeOf(TransferV2{}),
		ReissueTransaction:        reflect.TypeOf(ReissueV2{}),
		BurnTransaction:           reflect.TypeOf(BurnV2{}),
		ExchangeTransaction:       reflect.TypeOf(ExchangeV2{}),
		LeaseTransaction:          reflect.TypeOf(LeaseV2{}),
		LeaseCancelTransaction:    reflect.TypeOf(LeaseCancelV2{}),
		CreateAliasTransaction:    reflect.TypeOf(CreateAliasV2{}),
		DataTransaction:           reflect.TypeOf(DataV1{}),
		SetScriptTransaction:      reflect.TypeOf(SetScriptV1{}),
		SponsorshipTransaction:    reflect.TypeOf(SponsorshipV1{}),
		SetAssetScriptTransaction: reflect.TypeOf(SetAssetScriptV1{}),
	}

	bytesToTransactionsV1 = map[TransactionType]reflect.Type{
		GenesisTransaction:      reflect.TypeOf(Genesis{}),
		PaymentTransaction:      reflect.TypeOf(Payment{}),
		IssueTransaction:        reflect.TypeOf(IssueV1{}),
		TransferTransaction:     reflect.TypeOf(TransferV1{}),
		ReissueTransaction:      reflect.TypeOf(ReissueV1{}),
		BurnTransaction:         reflect.TypeOf(BurnV1{}),
		ExchangeTransaction:     reflect.TypeOf(ExchangeV1{}),
		LeaseTransaction:        reflect.TypeOf(LeaseV1{}),
		LeaseCancelTransaction:  reflect.TypeOf(LeaseCancelV1{}),
		CreateAliasTransaction:  reflect.TypeOf(CreateAliasV1{}),
		MassTransferTransaction: reflect.TypeOf(MassTransferV1{}),
	}
)

type Transaction interface {
	GetID() []byte
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
}

func BytesToTransaction(tx []byte) (Transaction, error) {
	if len(tx) < 2 {
		return nil, errors.New("invalid size of transation's bytes slice")
	}
	if tx[0] == 0 {
		transactionType, ok := bytesToTransactionsV2[TransactionType(tx[1])]
		if !ok {
			return nil, errors.New("invalid transaction type")
		}
		transaction, ok := reflect.New(transactionType).Interface().(Transaction)
		if !ok {
			panic("This transaction type does not implement marshal/unmarshal functions")
		}
		if err := transaction.UnmarshalBinary(tx); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal transaction")
		}
		return Transaction(transaction), nil
	} else {
		transactionType, ok := bytesToTransactionsV1[TransactionType(tx[0])]
		if !ok {
			return nil, errors.New("invalid transaction type")
		}
		transaction, ok := reflect.New(transactionType).Interface().(Transaction)
		if !ok {
			panic("This transaction type does not implement marshal/unmarshal functions")
		}
		if err := transaction.UnmarshalBinary(tx); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal transaction")
		}
		return transaction, nil
	}
}

//Genesis is a transaction used to initial balances distribution. This transactions allowed only in the first block.
type Genesis struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Signature `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Timestamp uint64            `json:"timestamp"`
	Recipient Address           `json:"recipient"`
	Amount    uint64            `json:"amount"`
}

func (tx Genesis) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedGenesis returns a new unsigned Genesis transaction. Actually Genesis transaction could not be signed.
//That is why it doesn't implement Sing method. Instead it has GenerateSigID method, which calculates ID and uses it also as a signature.
func NewUnsignedGenesis(recipient Address, amount, timestamp uint64) (*Genesis, error) {
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if ok, err := recipient.Validate(); !ok {
		return nil, errors.Wrapf(err, "invalid recipient address '%s'", recipient.String())
	}
	return &Genesis{Type: GenesisTransaction, Version: 1, Timestamp: timestamp, Recipient: recipient, Amount: amount}, nil
}

func (tx *Genesis) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, genesisBodyLen)
	buf[0] = byte(tx.Type)
	binary.BigEndian.PutUint64(buf[1:], tx.Timestamp)
	copy(buf[9:], tx.Recipient[:])
	binary.BigEndian.PutUint64(buf[9+AddressSize:], tx.Amount)
	return buf, nil
}

func (tx *Genesis) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if tx.Type != GenesisTransaction {
		return errors.Errorf("unexpected transaction type %d for Genesis transaction", tx.Type)
	}
	data = data[1:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	copy(tx.Recipient[:], data[:AddressSize])
	data = data[AddressSize:]
	tx.Amount = binary.BigEndian.Uint64(data)
	return nil
}

//GenerateSigID calculates hash of the transaction and use it as an ID. Also doubled hash is used as a signature.
func (tx *Genesis) GenerateSigID() error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to generate signature of Genesis transaction")
	}
	d := make([]byte, len(b)+3)
	copy(d[3:], b)
	h, err := crypto.FastHash(d)
	if err != nil {
		return errors.Wrap(err, "failed to generate signature of Genesis transaction")
	}
	var s crypto.Signature
	copy(s[0:], h[:])
	copy(s[crypto.DigestSize:], h[:])
	tx.ID = &s
	tx.Signature = &s
	return nil
}

//MarshalBinary writes transaction bytes to slice of bytes.
func (tx *Genesis) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Genesis transaction to bytes")
	}
	return b, nil
}

//UnmarshalBinary reads transaction values from the slice of bytes.
func (tx *Genesis) UnmarshalBinary(data []byte) error {
	if l := len(data); l != genesisBodyLen {
		return errors.Errorf("incorrect data lenght for Genesis transaction, expected %d, received %d", genesisBodyLen, l)
	}
	if data[0] != byte(GenesisTransaction) {
		return errors.Errorf("incorrect transaction type %d for Genesis transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Genesis transaction from bytes")
	}
	err = tx.GenerateSigID()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Genesis transaction from bytes")
	}
	return nil
}

//Payment transaction is deprecated and can be used only for validation of blockchain.
type Payment struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version"`
	ID        *crypto.Signature `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	SenderPK  crypto.PublicKey  `json:"senderPublicKey"`
	Recipient Address           `json:"recipient"`
	Amount    uint64            `json:"amount"`
	Fee       uint64            `json:"fee"`
	Timestamp uint64            `json:"timestamp"`
}

func (tx Payment) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedPayment creates new Payment transaction with empty Signature and ID fields.
func NewUnsignedPayment(senderPK crypto.PublicKey, recipient Address, amount, fee, timestamp uint64) (*Payment, error) {
	if ok, err := recipient.Validate(); !ok {
		return nil, errors.Wrapf(err, "invalid recipient address '%s'", recipient.String())
	}
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &Payment{Type: PaymentTransaction, Version: 1, SenderPK: senderPK, Recipient: recipient, Amount: amount, Fee: fee, Timestamp: timestamp}, nil
}

func (tx *Payment) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, paymentBodyLen)
	buf[0] = byte(tx.Type)
	binary.BigEndian.PutUint64(buf[1:], tx.Timestamp)
	copy(buf[9:], tx.SenderPK[:])
	copy(buf[9+crypto.PublicKeySize:], tx.Recipient[:])
	binary.BigEndian.PutUint64(buf[9+crypto.PublicKeySize+AddressSize:], tx.Amount)
	binary.BigEndian.PutUint64(buf[17+crypto.PublicKeySize+AddressSize:], tx.Fee)
	return buf, nil

}

func (tx *Payment) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l != paymentBodyLen {
		return errors.Errorf("incorrect data size %d for Payment transaction, expected %d", l, paymentBodyLen)
	}
	if tx.Type != PaymentTransaction {
		return errors.Errorf("unexpected transaction type %d for Payment transaction", tx.Type)
	}
	data = data[1:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(tx.Recipient[:], data[:AddressSize])
	data = data[AddressSize:]
	tx.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	return nil
}

//Sign calculates transaction signature and set it as an ID.
func (tx *Payment) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign Payment transaction")
	}
	d := make([]byte, len(b)+3)
	copy(d[3:], b)
	s := crypto.Sign(secretKey, d)
	tx.ID = &s
	tx.Signature = &s
	return nil
}

//Verify checks that the Signature is valid for given public key.
func (tx *Payment) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify Payment transaction")
	}
	d := make([]byte, len(b)+3)
	copy(d[3:], b)
	return crypto.Verify(publicKey, *tx.Signature, d), nil
}

//MarshalBinary returns a bytes representation of Payment transaction.
func (tx *Payment) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {

	}
	buf := make([]byte, paymentBodyLen+crypto.SignatureSize)
	copy(buf, b)
	copy(buf[paymentBodyLen:], tx.Signature[:])
	return buf, nil
}

//UnmarshalBinary reads Payment transaction from its binary representation.
func (tx *Payment) UnmarshalBinary(data []byte) error {
	size := paymentBodyLen + crypto.SignatureSize
	if l := len(data); l != size {
		return errors.Errorf("not enough data for Payment transaction, expected %d, received %d", size, l)
	}
	if data[0] != byte(PaymentTransaction) {
		return errors.Errorf("incorrect transaction type %d for Payment transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data[:paymentBodyLen])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Payment transaction from bytes")
	}
	data = data[paymentBodyLen:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	tx.ID = &s
	return nil
}

type Transfer struct {
	SenderPK    crypto.PublicKey `json:"senderPublicKey"`
	AmountAsset OptionalAsset    `json:"assetId"`
	FeeAsset    OptionalAsset    `json:"feeAssetId"`
	Timestamp   uint64           `json:"timestamp,omitempty"`
	Amount      uint64           `json:"amount"`
	Fee         uint64           `json:"fee"`
	Recipient   Recipient        `json:"recipient"`
	Attachment  Attachment       `json:"attachment,omitempty"`
}

func newTransfer(senderPK crypto.PublicKey, amountAsset, feeAsset OptionalAsset, timestamp, amount, fee uint64, recipient Address, attachment string) (*Transfer, error) {
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	if len(attachment) > maxAttachmentLengthBytes {
		return nil, errors.New("attachment too long")
	}
	if ok, err := recipient.Validate(); !ok {
		return nil, errors.Wrapf(err, "invalid recipient address '%s'", recipient.String())
	}
	return &Transfer{SenderPK: senderPK, AmountAsset: amountAsset, FeeAsset: feeAsset, Timestamp: timestamp, Amount: amount, Fee: fee, Recipient: NewRecipientFromAddress(recipient), Attachment: Attachment(attachment)}, nil
}

func (tx *Transfer) marshalBinary() ([]byte, error) {
	p := 0
	aal := 0
	if tx.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	fal := 0
	if tx.FeeAsset.Present {
		fal += crypto.DigestSize
	}
	rb, err := tx.Recipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	rl := len(rb)
	atl := len(tx.Attachment)
	buf := make([]byte, transferLen+aal+fal+atl+rl)
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	aab, err := tx.AmountAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	copy(buf[p:], aab)
	p += 1 + aal
	fab, err := tx.FeeAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	copy(buf[p:], fab)
	p += 1 + fal
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	copy(buf[p:], rb)
	p += rl
	PutStringWithUInt16Len(buf[p:], tx.Attachment.String())
	return buf, nil
}

func (tx *Transfer) unmarshalBinary(data []byte) error {
	if l := len(data); l < transferLen {
		return errors.Errorf("%d bytes is not enough for Transfer body, expected not less then %d bytes", l, transferLen)
	}
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	err = tx.AmountAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Transfer body from bytes")
	}
	data = data[1:]
	if tx.AmountAsset.Present {
		data = data[crypto.DigestSize:]
	}
	err = tx.FeeAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Transfer body from bytes")
	}
	data = data[1:]
	if tx.FeeAsset.Present {
		data = data[crypto.DigestSize:]
	}
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	err = tx.Recipient.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Transfer body from bytes")
	}
	data = data[tx.Recipient.len:]
	a, err := StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Transfer body from bytes")
	}
	tx.Attachment = Attachment(a)
	return nil
}

type Issue interface {
	GetID() []byte
	GetSenderPK() crypto.PublicKey
	GetName() string
	GetDescription() string
	GetQuantity() uint64
	GetDecimals() byte
	GetReissuable() bool
	GetScript() Script
	GetTimestamp() uint64
	GetFee() uint64
}

type Reissue struct {
	SenderPK   crypto.PublicKey `json:"senderPublicKey"`
	AssetID    crypto.Digest    `json:"assetId"`
	Quantity   uint64           `json:"quantity"`
	Reissuable bool             `json:"reissuable"`
	Timestamp  uint64           `json:"timestamp,omitempty"`
	Fee        uint64           `json:"fee"`
}

func newReissue(senderPK crypto.PublicKey, assetID crypto.Digest, quantity uint64, reissuable bool, timestamp, fee uint64) (*Reissue, error) {
	if quantity <= 0 {
		return nil, errors.New("quantity should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &Reissue{SenderPK: senderPK, AssetID: assetID, Quantity: quantity, Reissuable: reissuable, Timestamp: timestamp, Fee: fee}, nil
}

func (tx *Reissue) marshalBinary() ([]byte, error) {
	p := 0
	buf := make([]byte, reissueLen)
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], tx.AssetID[:])
	p += crypto.DigestSize
	binary.BigEndian.PutUint64(buf[p:], tx.Quantity)
	p += 8
	PutBool(buf[p:], tx.Reissuable)
	p++
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func (tx *Reissue) unmarshalBinary(data []byte) error {
	if l := len(data); l < reissueLen {
		return errors.Errorf("%d bytes is not enough for Reissue body, expected not less then %d bytes", l, reissueLen)
	}
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(tx.AssetID[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	tx.Quantity = binary.BigEndian.Uint64(data)
	data = data[8:]
	var err error
	tx.Reissuable, err = Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Reissuable")
	}
	data = data[1:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

type Exchange interface {
	GetID() []byte
	GetSenderPK() crypto.PublicKey
	GetBuyOrder() (OrderBody, error)
	GetSellOrder() (OrderBody, error)
	GetPrice() uint64
	GetAmount() uint64
	GetBuyMatcherFee() uint64
	GetSellMatcherFee() uint64
	GetFee() uint64
	GetTimestamp() uint64
}

type Burn struct {
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	AssetID   crypto.Digest    `json:"assetId"`
	Amount    uint64           `json:"amount"`
	Timestamp uint64           `json:"timestamp,omitempty"`
	Fee       uint64           `json:"fee"`
}

func newBurn(senderPK crypto.PublicKey, assetID crypto.Digest, amount, timestamp, fee uint64) (*Burn, error) {
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &Burn{SenderPK: senderPK, AssetID: assetID, Amount: amount, Timestamp: timestamp, Fee: fee}, nil
}

func (b *Burn) marshalBinary() ([]byte, error) {
	buf := make([]byte, burnLen)
	p := 0
	copy(buf[p:], b.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], b.AssetID[:])
	p += crypto.DigestSize
	binary.BigEndian.PutUint64(buf[p:], b.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], b.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], b.Timestamp)
	return buf, nil
}

func (b *Burn) unmarshalBinary(data []byte) error {
	if l := len(data); l < burnLen {
		return errors.Errorf("%d bytes is not enough for burn, expected not less then %d", l, burnLen)
	}
	copy(b.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(b.AssetID[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	b.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	b.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	b.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

type Lease struct {
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	Recipient Recipient        `json:"recipient"`
	Amount    uint64           `json:"amount"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func newLease(senderPK crypto.PublicKey, recipient Address, amount, fee, timestamp uint64) (*Lease, error) {
	if ok, err := recipient.Validate(); !ok {
		return nil, errors.Wrap(err, "failed to create new unsigned Lease transaction")
	}
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &Lease{SenderPK: senderPK, Recipient: NewRecipientFromAddress(recipient), Amount: amount, Fee: fee, Timestamp: timestamp}, nil
}

func (l *Lease) marshalBinary() ([]byte, error) {
	rl := l.Recipient.len
	buf := make([]byte, leaseLen+rl)
	p := 0
	copy(buf[p:], l.SenderPK[:])
	p += crypto.PublicKeySize
	rb, err := l.Recipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal lease to bytes")
	}
	copy(buf[p:], rb)
	p += rl
	binary.BigEndian.PutUint64(buf[p:], l.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], l.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], l.Timestamp)
	return buf, nil
}

func (l *Lease) unmarshalBinary(data []byte) error {
	if l := len(data); l < leaseLen {
		return errors.Errorf("not enough data for lease, expected not less then %d, received %d", leaseLen, l)
	}
	copy(l.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	err := l.Recipient.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal lease from bytes")
	}
	data = data[l.Recipient.len:]
	l.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	l.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	l.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

type LeaseCancel struct {
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	LeaseID   crypto.Digest    `json:"leaseId"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func newLeaseCancel(senderPK crypto.PublicKey, leaseID crypto.Digest, fee, timestamp uint64) (*LeaseCancel, error) {
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &LeaseCancel{SenderPK: senderPK, LeaseID: leaseID, Fee: fee, Timestamp: timestamp}, nil
}

func (lc *LeaseCancel) marshalBinary() ([]byte, error) {
	buf := make([]byte, leaseCancelLen)
	p := 0
	copy(buf[p:], lc.SenderPK[:])
	p += crypto.PublicKeySize
	binary.BigEndian.PutUint64(buf[p:], lc.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], lc.Timestamp)
	p += 8
	copy(buf[p:], lc.LeaseID[:])
	return buf, nil
}

func (lc *LeaseCancel) unmarshalBinary(data []byte) error {
	if l := len(data); l < leaseCancelLen {
		return errors.Errorf("not enough data for leaseCancel, expected not less then %d, received %d", leaseCancelLen, l)
	}
	copy(lc.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	lc.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	lc.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	copy(lc.LeaseID[:], data[:crypto.DigestSize])
	return nil
}

type CreateAlias struct {
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	Alias     Alias            `json:"alias"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func newCreateAlias(senderPK crypto.PublicKey, alias Alias, fee, timestamp uint64) (*CreateAlias, error) {
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &CreateAlias{SenderPK: senderPK, Alias: alias, Fee: fee, Timestamp: timestamp}, nil
}

func (ca *CreateAlias) marshalBinary() ([]byte, error) {
	p := 0
	buf := make([]byte, createAliasLen+len(ca.Alias.Alias))
	copy(buf[p:], ca.SenderPK[:])
	p += crypto.PublicKeySize
	ab, err := ca.Alias.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAlias to bytes")
	}
	al := len(ab)
	binary.BigEndian.PutUint16(buf[p:], uint16(al))
	p += 2
	copy(buf[p:], ab)
	p += al
	binary.BigEndian.PutUint64(buf[p:], ca.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], ca.Timestamp)
	return buf, nil
}

func (ca *CreateAlias) unmarshalBinary(data []byte) error {
	if l := len(data); l < createAliasLen {
		return errors.Errorf("not enough data for CreateAlias, expected not less then %d, received %d", createAliasLen, l)
	}
	copy(ca.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	al := binary.BigEndian.Uint16(data)
	data = data[2:]
	err := ca.Alias.UnmarshalBinary(data[:al])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAlias from bytes")
	}
	data = data[al:]
	ca.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	ca.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

func (ca *CreateAlias) id() (*crypto.Digest, error) {
	ab, err := ca.Alias.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get CreateAlias transaction ID")
	}
	al := len(ab)
	buf := make([]byte, 1+al)
	buf[0] = byte(CreateAliasTransaction)
	copy(buf[1:], ab)
	d, err := crypto.FastHash(buf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get CreateAlias transaction ID")
	}
	return &d, err
}
