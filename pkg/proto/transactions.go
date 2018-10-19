package proto

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type TransactionType byte

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
)

const (
	maxAttachmentLengthBytes      = 140
	maxDescriptionLen             = 1000
	maxAssetNameLen               = 16
	minAssetNameLen               = 4
	maxDecimals                   = 8
	proofsVersion            byte = 1

	genesisBodyLen            = 1 + 8 + AddressSize + 8
	paymentBodyLen            = 1 + 8 + crypto.PublicKeySize + AddressSize + 8 + 8
	issueV1FixedBodyLen       = 1 + crypto.PublicKeySize + 2 + 2 + 8 + 1 + 1 + 8 + 8
	issueV1MinBodyLen         = issueV1FixedBodyLen + 4 // 4 because of the shortest allowed Asset name of 4 bytes
	issueV1MinLen             = 1 + crypto.SignatureSize + issueV1MinBodyLen
	transferV1FixedBodyLen    = 1 + crypto.PublicKeySize + 1 + 1 + 8 + 8 + 8 + AddressSize + 2
	transferV1MinLen          = 1 + crypto.SignatureSize + transferV1FixedBodyLen
	reissueV1BodyLen          = 1 + crypto.PublicKeySize + crypto.DigestSize + 8 + 1 + 8 + 8
	reissueV1MinLen           = 1 + crypto.SignatureSize + reissueV1BodyLen
	burnV1BodyLen             = 1 + crypto.PublicKeySize + crypto.DigestSize + 8 + 8 + 8
	burnV1MinLen              = 1 + crypto.SignatureSize + burnV1BodyLen
	exchangeV1FixedBodyLen    = 1 + 4 + 4 + orderMinLen + orderMinLen + 8 + 8 + 8 + 8 + 8 + 8
	exchangeV1MinLen          = exchangeV1FixedBodyLen + crypto.SignatureSize
	leaseV1BodyLen            = 1 + crypto.PublicKeySize + AddressSize + 8 + 8 + 8
	leaseV1MinLen             = leaseV1BodyLen + crypto.SignatureSize
	leaseCancelV1BodyLen      = 1 + crypto.PublicKeySize + 8 + 8 + crypto.DigestSize
	leaseCancelV1MinLen       = leaseCancelV1BodyLen + crypto.SignatureSize
	createAliasV1FixedBodyLen = 1 + crypto.PublicKeySize + 2 + 8 + 8
	createAliasV1MinLen       = createAliasV1FixedBodyLen + crypto.SignatureSize
)

type Genesis struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Signature `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Timestamp uint64            `json:"timestamp"`
	Recipient Address           `json:"recipient"`
	Amount    uint64            `json:"amount"`
}

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

func (tx *Genesis) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Genesis transaction to bytes")
	}
	return b, nil
}

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

func (tx *Payment) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {

	}
	buf := make([]byte, paymentBodyLen+crypto.SignatureSize)
	copy(buf[:], b)
	copy(buf[paymentBodyLen:], tx.Signature[:])
	return buf, nil
}

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

type IssueV1 struct {
	Type        TransactionType   `json:"type"`
	Version     byte              `json:"version,omitempty"`
	ID          *crypto.Digest    `json:"id,omitempty"`
	Signature   *crypto.Signature `json:"signature,omitempty"`
	SenderPK    crypto.PublicKey  `json:"senderPublicKey"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Quantity    uint64            `json:"quantity"`
	Decimals    byte              `json:"decimals"`
	Reissuable  bool              `json:"reissuable"`
	Timestamp   uint64            `json:"timestamp,omitempty"`
	Fee         uint64            `json:"fee"`
}

func NewUnsignedIssueV1(senderPK crypto.PublicKey, name, description string, quantity uint64, decimals byte, reissuable bool, timestamp, fee uint64) (*IssueV1, error) {
	if l := len(name); l < minAssetNameLen || l > maxAssetNameLen {
		return nil, errors.New("incorrect number of bytes in the asset's name")
	}
	if l := len(description); l > maxDescriptionLen {
		return nil, errors.New("incorrect number of bytes in the asset's description")
	}
	if quantity <= 0 {
		return nil, errors.New("quantity should be positive")
	}
	if decimals > maxDecimals {
		return nil, errors.Errorf("incorrect decimals, should be no more then %d", maxDecimals)
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &IssueV1{Type: IssueTransaction, Version: 1, SenderPK: senderPK, Name: name, Description: description, Quantity: quantity, Decimals: decimals, Reissuable: reissuable, Timestamp: timestamp, Fee: fee}, nil
}

func (tx *IssueV1) bodyMarshalBinary() ([]byte, error) {
	kl := crypto.PublicKeySize
	nl := len(tx.Name)
	dl := len(tx.Description)
	buf := make([]byte, issueV1FixedBodyLen+nl+dl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.SenderPK[:])
	PutStringWithUInt16Len(buf[1+kl:], tx.Name)
	PutStringWithUInt16Len(buf[3+kl+nl:], tx.Description)
	binary.BigEndian.PutUint64(buf[5+kl+nl+dl:], tx.Quantity)
	buf[13+kl+nl+dl] = tx.Decimals
	PutBool(buf[14+kl+nl+dl:], tx.Reissuable)
	binary.BigEndian.PutUint64(buf[15+kl+nl+dl:], tx.Fee)
	binary.BigEndian.PutUint64(buf[23+kl+nl+dl:], tx.Timestamp)
	return buf, nil
}

func (tx *IssueV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < issueV1MinBodyLen {
		return errors.Errorf("not enough data for IssueV1 transaction %d, expected not less then %d", l, issueV1MinBodyLen)
	}
	if tx.Type != IssueTransaction {
		return errors.Errorf("unexpected transaction type %d for IssueV1 transaction", tx.Type)
	}
	data = data[1:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	tx.Name, err = StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Name")
	}
	data = data[2+len(tx.Name):]
	tx.Description, err = StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Description")
	}
	data = data[2+len(tx.Description):]
	tx.Quantity = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Decimals = data[0]
	data = data[1:]
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

func (tx *IssueV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign IssueV1 transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *IssueV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of IssueV1 transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *IssueV1) MarshalBinary() ([]byte, error) {
	sl := crypto.SignatureSize
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal IssueV1 transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, 1+sl+bl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.Signature[:])
	copy(buf[1+sl:], b)
	return buf, nil
}

func (tx *IssueV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < issueV1MinLen {
		return errors.Errorf("not enough data for IssueV1 transaction, expected not less then %d, received %d", issueV1MinLen, l)
	}
	if data[0] != byte(IssueTransaction) {
		return errors.Errorf("incorrect transaction type %d for IssueV1 transaction", data[0])
	}
	data = data[1:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	data = data[crypto.SignatureSize:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IssueV1 transaction")
	}
	d, err := crypto.FastHash(data)
	if err != nil {
		return errors.Wrap(err, "failed to hash IssueV1 transaction")
	}
	tx.ID = &d
	return nil
}

//type IssueV2 struct {
//	Type        TransactionType  `json:"type"`
//	Version     byte             `json:"version"`
//	ID          *crypto.Digest   `json:"id,omitempty"`
//	SenderPK    crypto.PublicKey `json:"senderPublicKey"`
//	Name        string           `json:"name"`
//	Description string           `json:"description"`
//	Quantity    uint64           `json:"quantity"`
//	Decimals    byte             `json:"decimals"`
//	Reissuable  bool             `json:"reissuable"`
//	Script      OptionalScript   `json:"script"`
//	Timestamp   uint64           `json:"timestamp,omitempty"`
//	Fee         uint64           `json:"fee"`
//}

type TransferV1 struct {
	Type        TransactionType   `json:"type"`
	Version     byte              `json:"version,omitempty"`
	ID          *crypto.Digest    `json:"id,omitempty"`
	Signature   *crypto.Signature `json:"signature,omitempty"`
	SenderPK    crypto.PublicKey  `json:"senderPublicKey"`
	AmountAsset OptionalAsset     `json:"assetId"`
	FeeAsset    OptionalAsset     `json:"feeAssetId"`
	Timestamp   uint64            `json:"timestamp,omitempty"`
	Amount      uint64            `json:"amount"`
	Fee         uint64            `json:"fee"`
	Recipient   Address           `json:"recipient"`
	Attachment  Attachment        `json:"attachment,omitempty"`
}

func NewUnsignedTransferV1(senderPK crypto.PublicKey, amountAsset, feeAsset OptionalAsset, timestamp, amount, fee uint64, recipient Address, attachment string) (*TransferV1, error) {
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
	return &TransferV1{Type: TransferTransaction, Version: 1, SenderPK: senderPK, AmountAsset: amountAsset, FeeAsset: feeAsset, Timestamp: timestamp, Amount: amount, Fee: fee, Recipient: recipient, Attachment: Attachment(attachment)}, nil
}

func (tx *TransferV1) bodyMarshalBinary() ([]byte, error) {
	kl := crypto.PublicKeySize
	aal := 0
	fal := 0
	if tx.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	if tx.FeeAsset.Present {
		fal += crypto.DigestSize
	}
	atl := len(tx.Attachment)
	buf := make([]byte, transferV1FixedBodyLen+aal+fal+atl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.SenderPK[:])
	aab, err := tx.AmountAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferV1 body")
	}
	copy(buf[1+kl:], aab)
	fab, err := tx.FeeAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferV1 body")
	}
	copy(buf[1+kl+1+aal:], fab)
	binary.BigEndian.PutUint64(buf[1+kl+1+aal+1+fal:], tx.Timestamp)
	binary.BigEndian.PutUint64(buf[1+kl+1+aal+1+fal+8:], tx.Amount)
	binary.BigEndian.PutUint64(buf[1+kl+1+aal+1+fal+8+8:], tx.Fee)
	copy(buf[1+kl+1+aal+1+fal+8+8+8:], tx.Recipient[:])
	PutStringWithUInt16Len(buf[1+kl+1+aal+1+fal+8+8+8+AddressSize:], tx.Attachment.String())
	return buf, nil
}

func (tx *TransferV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < transferV1FixedBodyLen {
		return errors.Errorf("%d bytes is not enough for TransferV1 transaction, expected not less then %d bytes", l, transferV1FixedBodyLen)
	}
	if tx.Type != TransferTransaction {
		return errors.Errorf("unexpected transaction type %d for TransferV1 transaction", tx.Type)
	}
	data = data[1:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	err = tx.AmountAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV1 body from bytes")
	}
	data = data[1:]
	if tx.AmountAsset.Present {
		data = data[crypto.DigestSize:]
	}
	err = tx.FeeAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV1 body from bytes")
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
	a, err := StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV1 body from bytes")
	}
	tx.Attachment = Attachment(a)
	return nil
}

func (tx *TransferV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign TransferV1 transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *TransferV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of TransferV1 transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *TransferV1) MarshalBinary() ([]byte, error) {
	sl := crypto.SignatureSize
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferV1 transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, 1+sl+bl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.Signature[:])
	copy(buf[1+sl:], b)
	return buf, nil
}

func (tx *TransferV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < transferV1MinLen {
		return errors.Errorf("not enough data for TransferV1 transaction, expected not less then %d, received %d", transferV1MinLen, l)
	}
	if data[0] != byte(TransferTransaction) {
		return errors.Errorf("incorrect transaction type %d for TransferV1 transaction", data[0])
	}
	data = data[1:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	data = data[crypto.SignatureSize:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV1 transaction")
	}
	d, err := crypto.FastHash(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV1 transaction")
	}
	tx.ID = &d
	return nil
}

type ReissueV1 struct {
	Type       TransactionType   `json:"type"`
	Version    byte              `json:"version,omitempty"`
	ID         *crypto.Digest    `json:"id,omitempty"`
	Signature  *crypto.Signature `json:"signature,omitempty"`
	SenderPK   crypto.PublicKey  `json:"senderPublicKey"`
	AssetId    crypto.Digest     `json:"assetId"`
	Quantity   uint64            `json:"quantity"`
	Reissuable bool              `json:"reissuable"`
	Timestamp  uint64            `json:"timestamp,omitempty"`
	Fee        uint64            `json:"fee"`
}

func NewUnsignedReissueV1(senderPK crypto.PublicKey, assetId crypto.Digest, quantity uint64, reissuable bool, timestamp, fee uint64) (*ReissueV1, error) {
	if quantity <= 0 {
		return nil, errors.New("quantity should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &ReissueV1{Type: ReissueTransaction, Version: 1, SenderPK: senderPK, AssetId: assetId, Quantity: quantity, Reissuable: reissuable, Timestamp: timestamp, Fee: fee}, nil
}

func (tx *ReissueV1) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, reissueV1BodyLen)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.SenderPK[:])
	copy(buf[1+crypto.PublicKeySize:], tx.AssetId[:])
	binary.BigEndian.PutUint64(buf[1+crypto.PublicKeySize+crypto.DigestSize:], tx.Quantity)
	PutBool(buf[9+crypto.PublicKeySize+crypto.DigestSize:], tx.Reissuable)
	binary.BigEndian.PutUint64(buf[10+crypto.PublicKeySize+crypto.DigestSize:], tx.Fee)
	binary.BigEndian.PutUint64(buf[18+crypto.PublicKeySize+crypto.DigestSize:], tx.Timestamp)
	return buf, nil
}

func (tx *ReissueV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < reissueV1BodyLen {
		return errors.Errorf("not enough data for ReissueV1 transaction %d, expected not less then %d", l, reissueV1BodyLen)
	}
	if tx.Type != ReissueTransaction {
		return errors.Errorf("unexpected transaction type %d for ReissueV1 transaction", tx.Type)
	}
	data = data[1:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(tx.AssetId[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	var err error
	tx.Quantity = binary.BigEndian.Uint64(data)
	data = data[8:]
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

func (tx *ReissueV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ReissueV1 transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *ReissueV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of ReissueV1 transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *ReissueV1) MarshalBinary() ([]byte, error) {
	sl := crypto.SignatureSize
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueV1 transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, 1+sl+bl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.Signature[:])
	copy(buf[1+sl:], b)
	return buf, nil
}

func (tx *ReissueV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < reissueV1MinLen {
		return errors.Errorf("not enough data for ReissueV1 transaction, expected not less then %d, received %d", reissueV1MinLen, l)
	}
	if data[0] != byte(ReissueTransaction) {
		return errors.Errorf("incorrect transaction type %d for ReissueV1 transaction", data[0])
	}
	data = data[1:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	data = data[crypto.SignatureSize:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueV1 transaction")
	}
	d, err := crypto.FastHash(data)
	if err != nil {
		return errors.Wrap(err, "failed to hash ReissueV1 transaction")
	}
	tx.ID = &d
	return nil
}

type BurnV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	SenderPK  crypto.PublicKey  `json:"senderPublicKey"`
	AssetId   crypto.Digest     `json:"assetId"`
	Amount    uint64            `json:"amount"`
	Timestamp uint64            `json:"timestamp,omitempty"`
	Fee       uint64            `json:"fee"`
}

func NewUnsignedBurnV1(senderPK crypto.PublicKey, assetId crypto.Digest, amount, timestamp, fee uint64) (*BurnV1, error) {
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &BurnV1{Type: BurnTransaction, Version: 1, SenderPK: senderPK, AssetId: assetId, Amount: amount, Timestamp: timestamp, Fee: fee}, nil
}

func (tx *BurnV1) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, burnV1BodyLen)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.SenderPK[:])
	copy(buf[1+crypto.PublicKeySize:], tx.AssetId[:])
	binary.BigEndian.PutUint64(buf[1+crypto.PublicKeySize+crypto.DigestSize:], tx.Amount)
	binary.BigEndian.PutUint64(buf[9+crypto.PublicKeySize+crypto.DigestSize:], tx.Fee)
	binary.BigEndian.PutUint64(buf[17+crypto.PublicKeySize+crypto.DigestSize:], tx.Timestamp)
	return buf, nil
}

func (tx *BurnV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < burnV1BodyLen {
		return errors.Errorf("not enough data for BurnV1 transaction %d, expected not less then %d", l, burnV1BodyLen)
	}
	if tx.Type != BurnTransaction {
		return errors.Errorf("unexpected transaction type %d for BurnV1 transaction", tx.Type)
	}
	data = data[1:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(tx.AssetId[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	tx.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

func (tx *BurnV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign BurnV1 transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *BurnV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of BurnV1 transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *BurnV1) MarshalBinary() ([]byte, error) {
	sl := crypto.SignatureSize
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnV1 transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, 1+sl+bl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.Signature[:])
	copy(buf[1+sl:], b)
	return buf, nil
}

func (tx *BurnV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < burnV1MinLen {
		return errors.Errorf("not enough data for BurnV1 transaction, expected not less then %d, received %d", burnV1MinLen, l)
	}
	if data[0] != byte(BurnTransaction) {
		return errors.Errorf("incorrect transaction type %d for BurnV1 transaction", data[0])
	}
	data = data[1:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	data = data[crypto.SignatureSize:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnV1 transaction")
	}
	d, err := crypto.FastHash(data)
	if err != nil {
		return errors.Wrap(err, "failed to hash BurnV1 transaction")
	}
	tx.ID = &d
	return nil
}

type ExchangeV1 struct {
	Type           TransactionType   `json:"type"`
	Version        byte              `json:"version,omitempty"`
	ID             *crypto.Digest    `json:"id,omitempty"`
	Signature      *crypto.Signature `json:"signature,omitempty"`
	SenderPK       crypto.PublicKey  `json:"senderPublicKey"`
	BuyOrder       Order             `json:"order1"`
	SellOrder      Order             `json:"order2"`
	Price          uint64            `json:"price"`
	Amount         uint64            `json:"amount"`
	BuyMatcherFee  uint64            `json:"buyMatcherFee"`
	SellMatcherFee uint64            `json:"sellMatcherFee"`
	Fee            uint64            `json:"fee"`
	Timestamp      uint64            `json:"timestamp,omitempty"`
}

func NewUnsignedExchangeV1(buy, sell Order, price, amount, buyMatcherFee, sellMatcherFee, fee, timestamp uint64) (*ExchangeV1, error) {
	if buy.Signature == nil {
		return nil, errors.New("buy order should be signed")
	}
	if sell.Signature == nil {
		return nil, errors.New("sell order should be signed")
	}
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if price <= 0 {
		return nil, errors.New("price should be positive")
	}
	if buyMatcherFee <= 0 {
		return nil, errors.New("buy matcher's fee should be positive")
	}
	if sellMatcherFee <= 0 {
		return nil, errors.New("sell matcher's fee should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &ExchangeV1{Type: ExchangeTransaction, Version: 1, SenderPK: buy.MatcherPK, BuyOrder: buy, SellOrder: sell, Price: price, Amount: amount, BuyMatcherFee: buyMatcherFee, SellMatcherFee: sellMatcherFee, Fee: fee, Timestamp: timestamp}, nil
}

func (tx *ExchangeV1) bodyMarshalBinary() ([]byte, error) {
	bob, err := tx.BuyOrder.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal ExchangeV1 body to bytes")
	}
	bol := uint32(len(bob))
	sob, err := tx.SellOrder.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal ExchangeV1 body to bytes")
	}
	sol := uint32(len(sob))
	var p uint32
	buf := make([]byte, exchangeV1FixedBodyLen-orderMinLen*2+bol+sol)
	buf[0] = byte(tx.Type)
	p += 1
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

func (tx *ExchangeV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < exchangeV1FixedBodyLen {
		return errors.Errorf("not enough data for ExchangeV1 transaction, expected not less then %d, received %d", exchangeV1FixedBodyLen, l)
	}
	if tx.Type != ExchangeTransaction {
		return errors.Errorf("unexpected transaction type %d for ExchangeV1 transaction", tx.Type)
	}
	data = data[1:]
	bol := binary.BigEndian.Uint32(data)
	data = data[4:]
	sol := binary.BigEndian.Uint32(data)
	data = data[4:]
	var bo Order
	err := bo.UnmarshalBinary(data[:bol])
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal ExchangeV1 body from bytes")
	}
	tx.BuyOrder = bo
	data = data[bol:]
	var so Order
	err = so.UnmarshalBinary(data[:sol])
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal ExchangeV1 body from bytes")
	}
	tx.SellOrder = so
	data = data[sol:]
	tx.Price = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.BuyMatcherFee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.SellMatcherFee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.SenderPK = tx.BuyOrder.MatcherPK
	return nil
}

func (tx *ExchangeV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign ExchangeV1 transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *ExchangeV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of ExchangeV1 transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *ExchangeV1) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ExchangeV1 transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], tx.Signature[:])
	return buf, nil
}

func (tx *ExchangeV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < exchangeV1MinLen {
		return errors.Errorf("not enough data for ExchangeV1 transaction, expected not less then %d, received %d", exchangeV1MinLen, l)
	}
	if data[0] != byte(ExchangeTransaction) {
		return errors.Errorf("incorrect transaction type %d for ExchangeV1 transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeV1 transaction from bytes")
	}
	var bl int
	bl += exchangeV1FixedBodyLen
	if tx.BuyOrder.AssetPair.AmountAsset.Present {
		bl += crypto.DigestSize
	}
	if tx.BuyOrder.AssetPair.PriceAsset.Present {
		bl += crypto.DigestSize
	}
	if tx.SellOrder.AssetPair.AmountAsset.Present {
		bl += crypto.DigestSize
	}
	if tx.SellOrder.AssetPair.PriceAsset.Present {
		bl += crypto.DigestSize
	}
	b := data[:bl]
	data = data[bl:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeV1 transaction from bytes")
	}
	tx.ID = &d
	return nil
}

type LeaseV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	SenderPK  crypto.PublicKey  `json:"senderPublicKey"`
	Recipient Address           `json:"recipient"`
	Amount    uint64            `json:"amount"`
	Fee       uint64            `json:"fee"`
	Timestamp uint64            `json:"timestamp,omitempty"`
}

func NewUnsignedLeaseV1(senderPK crypto.PublicKey, recipient Address, amount, fee, timestamp uint64) (*LeaseV1, error) {
	if ok, err := recipient.Validate(); !ok {
		return nil, errors.Wrap(err, "failed to create new unsigned LeaseV1 transaction")
	}
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &LeaseV1{Type: LeaseTransaction, Version: 1, SenderPK: senderPK, Recipient: recipient, Amount: amount, Fee: fee, Timestamp: timestamp}, nil
}

func (tx *LeaseV1) bodyMarshalBinary() ([]byte, error) {
	var p uint32
	buf := make([]byte, leaseV1BodyLen)
	buf[0] = byte(tx.Type)
	p += 1
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], tx.Recipient[:])
	p += AddressSize
	binary.BigEndian.PutUint64(buf[p:], tx.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func (tx *LeaseV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < leaseV1BodyLen {
		return errors.Errorf("not enough data for LeaseV1 transaction, expected not less then %d, received %d", leaseV1BodyLen, l)
	}
	if tx.Type != LeaseTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseV1 transaction", tx.Type)
	}
	data = data[1:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(tx.Recipient[:], data[:AddressSize])
	data = data[AddressSize:]
	tx.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

func (tx *LeaseV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseV1 transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *LeaseV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of LeaseV1 transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *LeaseV1) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseV1 transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], tx.Signature[:])
	return buf, nil
}

func (tx *LeaseV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseV1MinLen {
		return errors.Errorf("not enough data for LeaseV1 transaction, expected not less then %d, received %d", leaseV1MinLen, l)
	}
	if data[0] != byte(LeaseTransaction) {
		return errors.Errorf("incorrect transaction type %d for LeaseV1 transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseV1 transaction from bytes")
	}
	b := data[:leaseV1BodyLen]
	data = data[leaseV1BodyLen:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseV1 transaction from bytes")
	}
	tx.ID = &d
	return nil
}

type LeaseCancelV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	SenderPK  crypto.PublicKey  `json:"senderPublicKey"`
	LeaseID   crypto.Digest     `json:"leaseId"`
	Fee       uint64            `json:"fee"`
	Timestamp uint64            `json:"timestamp,omitempty"`
}

func NewUnsignedLeaseCancelV1(senderPK crypto.PublicKey, leaseID crypto.Digest, fee, timestamp uint64) (*LeaseCancelV1, error) {
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &LeaseCancelV1{Type: LeaseCancelTransaction, Version: 1, SenderPK: senderPK, LeaseID: leaseID, Fee: fee, Timestamp: timestamp}, nil
}

func (tx *LeaseCancelV1) bodyMarshalBinary() ([]byte, error) {
	var p uint32
	buf := make([]byte, leaseCancelV1BodyLen)
	buf[0] = byte(tx.Type)
	p += 1
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	copy(buf[p:], tx.LeaseID[:])
	return buf, nil
}

func (tx *LeaseCancelV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < leaseCancelV1BodyLen {
		return errors.Errorf("not enough data for LeaseCancelV1 transaction, expected not less then %d, received %d", leaseCancelV1BodyLen, l)
	}
	if tx.Type != LeaseCancelTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseCancelV1 transaction", tx.Type)
	}
	data = data[1:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	copy(tx.LeaseID[:], data[:crypto.DigestSize])
	return nil
}

func (tx *LeaseCancelV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign LeaseCancelV1 transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *LeaseCancelV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of LeaseCancelV1 transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *LeaseCancelV1) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelV1 transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], tx.Signature[:])
	return buf, nil
}

func (tx *LeaseCancelV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseCancelV1MinLen {
		return errors.Errorf("not enough data for LeaseCancelV1 transaction, expected not less then %d, received %d", leaseCancelV1MinLen, l)
	}
	if data[0] != byte(LeaseCancelTransaction) {
		return errors.Errorf("incorrect transaction type %d for LeaseCancelV1 transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelV1 transaction from bytes")
	}
	b := data[:leaseCancelV1BodyLen]
	data = data[leaseCancelV1BodyLen:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelV1 transaction from bytes")
	}
	tx.ID = &d
	return nil
}

type CreateAliasV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	SenderPK  crypto.PublicKey  `json:"senderPublicKey"`
	Alias     Alias             `json:"alias"`
	Fee       uint64            `json:"fee"`
	Timestamp uint64            `json:"timestamp,omitempty"`
}

func NewUnsignedCreateAliasV1(senderPK crypto.PublicKey, alias Alias, fee, timestamp uint64) (*CreateAliasV1, error) {
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &CreateAliasV1{Type: CreateAliasTransaction, Version: 1, SenderPK: senderPK, Alias: alias, Fee: fee, Timestamp: timestamp}, nil
}

func (tx *CreateAliasV1) bodyMarshalBinary() ([]byte, error) {
	var p uint32
	buf := make([]byte, createAliasV1FixedBodyLen+len(tx.Alias.Alias)+4)
	buf[0] = byte(tx.Type)
	p += 1
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	ab, err := tx.Alias.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasV1 transaction body to bytes")
	}
	al := uint16(len(ab))
	binary.BigEndian.PutUint16(buf[p:], al)
	p += 2
	copy(buf[p:], ab)
	p += uint32(al)
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func (tx *CreateAliasV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < createAliasV1FixedBodyLen {
		return errors.Errorf("not enough data for CreateAliasV1 transaction, expected not less then %d, received %d", leaseCancelV1BodyLen, l)
	}
	if tx.Type != CreateAliasTransaction {
		return errors.Errorf("unexpected transaction type %d for CreateAliasV1 transaction", tx.Type)
	}
	data = data[1:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	al := binary.BigEndian.Uint16(data)
	data = data[2:]
	err := tx.Alias.UnmarshalBinary(data[:al])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV1 transaction body from bytes")
	}
	data = data[al:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

func (tx *CreateAliasV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	tx.ID, err = tx.id()
	if err != nil {
		return err
	}
	return nil
}

func (tx *CreateAliasV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of CreateAliasV1 transaction")
	}
	return crypto.Verify(publicKey, *tx.Signature, b), nil
}

func (tx *CreateAliasV1) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasV1 transaction to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], tx.Signature[:])
	return buf, nil
}

func (tx *CreateAliasV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < createAliasV1MinLen {
		return errors.Errorf("not enough data for CreateAliasV1 transaction, expected not less then %d, received %d", createAliasV1MinLen, l)
	}
	if data[0] != byte(CreateAliasTransaction) {
		return errors.Errorf("incorrect transaction type %d for CreateAliasV1 transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV1 transaction from bytes")
	}
	bl := createAliasV1FixedBodyLen + 4 + len(tx.Alias.Alias)
	data = data[bl:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	id, err := tx.id()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV1 transaction from bytes")
	}
	tx.ID = id
	return nil
}

func (tx *CreateAliasV1) id() (*crypto.Digest, error) {
	ab, err := tx.Alias.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get CreateAliasV1 transaction ID")
	}
	al := len(ab)
	buf := make([]byte, 1+al)
	buf[0] = byte(tx.Type)
	copy(buf[1:], ab)
	d, err := crypto.FastHash(buf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get CreateAliasV1 transaction ID")
	}
	return &d, err
}
