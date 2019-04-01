package proto

import (
	"encoding/binary"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	issueV1FixedBodyLen          = 1 + crypto.PublicKeySize + 2 + 2 + 8 + 1 + 1 + 8 + 8
	issueV1MinBodyLen            = issueV1FixedBodyLen + 4 // 4 because of the shortest allowed Asset name of 4 bytes
	issueV1MinLen                = 1 + crypto.SignatureSize + issueV1MinBodyLen
	transferV1FixedBodyLen       = 1 + transferLen
	transferV1MinLen             = 1 + crypto.SignatureSize + transferV1FixedBodyLen
	reissueV1BodyLen             = 1 + reissueLen
	reissueV1MinLen              = 1 + crypto.SignatureSize + reissueV1BodyLen
	burnV1BodyLen                = 1 + burnLen
	burnV1Len                    = burnV1BodyLen + crypto.SignatureSize
	exchangeV1FixedBodyLen       = 1 + 4 + 4 + 8 + 8 + 8 + 8 + 8 + 8
	exchangeV1MinLen             = exchangeV1FixedBodyLen + orderV1MinLen + orderV1MinLen + crypto.SignatureSize
	leaseV1BodyLen               = 1 + leaseLen
	leaseV1MinLen                = leaseV1BodyLen + crypto.SignatureSize
	leaseCancelV1BodyLen         = 1 + leaseCancelLen
	leaseCancelV1MinLen          = leaseCancelV1BodyLen + crypto.SignatureSize
	createAliasV1FixedBodyLen    = 1 + createAliasLen
	createAliasV1MinLen          = createAliasV1FixedBodyLen + crypto.SignatureSize
	massTransferEntryLen         = 8
	massTransferV1FixedLen       = 1 + 1 + crypto.PublicKeySize + 1 + 2 + 8 + 8 + 2
	massTransferV1MinLen         = massTransferV1FixedLen + proofsMinLen
	dataV1FixedBodyLen           = 1 + 1 + crypto.PublicKeySize + 2 + 8 + 8
	dataV1MinLen                 = dataV1FixedBodyLen + proofsMinLen
	setScriptV1FixedBodyLen      = 1 + 1 + 1 + crypto.PublicKeySize + 1 + 8 + 8
	setScriptV1MinLen            = 1 + setScriptV1FixedBodyLen + proofsMinLen
	sponsorshipV1BodyLen         = 1 + 1 + crypto.PublicKeySize + crypto.DigestSize + 8 + 8 + 8
	sponsorshipV1MinLen          = 1 + 1 + 1 + sponsorshipV1BodyLen + proofsMinLen
	setAssetScriptV1FixedBodyLen = 1 + 1 + 1 + crypto.PublicKeySize + crypto.DigestSize + 8 + 8 + 1
	setAssetScriptV1MinLen       = 1 + setScriptV1FixedBodyLen + proofsMinLen
)

//IssueV1 transaction is a transaction to issue new asset.
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

func (tx IssueV1) GetID() []byte {
	return tx.ID.Bytes()
}

func (tx IssueV1) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx IssueV1) GetName() string {
	return tx.Name
}

func (tx IssueV1) GetDescription() string {
	return tx.Description
}

func (tx IssueV1) GetQuantity() uint64 {
	return tx.Quantity
}

func (tx IssueV1) GetDecimals() byte {
	return tx.Decimals
}

func (tx IssueV1) GetReissuable() bool {
	return tx.Reissuable
}

func (tx IssueV1) GetScript() Script {
	return Script{}
}

func (tx IssueV1) GetTimestamp() uint64 {
	return tx.Timestamp
}

func (tx IssueV1) GetFee() uint64 {
	return tx.Fee
}

//NewUnsignedIssueV1 creates new IssueV1 transaction without signature and ID.
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
		return errors.Wrap(err, "failed to unmarshal AppName")
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

//Sign uses secretKey to sing the transaction.
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

//Verify checks that the signature of transaction is a valid signature for given public key.
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

//MarshalBinary saves transaction's binary representation to slice of bytes.
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

//UnmarshalBinary reads transaction from its binary representation.
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

//TransferV1 transaction to transfer any token from one account to another. Version 1.
type TransferV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Transfer
}

func (tx TransferV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedTransferV1 creates new TransferV1 transaction without signature and ID.
func NewUnsignedTransferV1(senderPK crypto.PublicKey, amountAsset, feeAsset OptionalAsset, timestamp, amount, fee uint64, recipient Address, attachment string) (*TransferV1, error) {
	t, err := newTransfer(senderPK, amountAsset, feeAsset, timestamp, amount, fee, recipient, attachment)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create TransferV1 transaction")
	}
	return &TransferV1{Type: TransferTransaction, Version: 1, Transfer: *t}, nil
}

func (tx *TransferV1) bodyMarshalBinary() ([]byte, error) {
	b, err := tx.Transfer.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal TransferV1 body")
	}
	buf := make([]byte, 1+len(b))
	buf[0] = byte(tx.Type)
	copy(buf[1:], b)
	return buf, nil
}

func (tx *TransferV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < transferV1FixedBodyLen {
		return errors.Errorf("%d bytes is not enough for TransferV1 transaction, expected not less then %d bytes", l, transferV1FixedBodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != TransferTransaction {
		return errors.Errorf("unexpected transaction type %d for TransferV1 transaction", tx.Type)
	}
	tx.Version = 1
	var t Transfer
	err := t.unmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal TransferV1 body from bytes")
	}
	tx.Transfer = t
	return nil
}

//Sign calculates a signature and a digest as an ID of the transaction.
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

//Verify use given public key to verify that the signature is valid.
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

//MarshalBinary saves transaction to its binary representation.
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

//UnmarshalBinary reads transaction from its binary representation.
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

//ReissueV1 is a transaction that allows to issue new amount of existing token, if it was issued as reissuable.
type ReissueV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Reissue
}

func (tx ReissueV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedReissueV1 creates new ReissueV1 transaction without signature and ID.
func NewUnsignedReissueV1(senderPK crypto.PublicKey, assetID crypto.Digest, quantity uint64, reissuable bool, timestamp, fee uint64) (*ReissueV1, error) {
	r, err := newReissue(senderPK, assetID, quantity, reissuable, timestamp, fee)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ReissueV1 transaction")
	}
	return &ReissueV1{Type: ReissueTransaction, Version: 1, Reissue: *r}, nil
}

func (tx *ReissueV1) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, reissueV1BodyLen)
	buf[0] = byte(tx.Type)
	b, err := tx.Reissue.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal ReissueV1 transaction to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *ReissueV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < reissueV1BodyLen {
		return errors.Errorf("not enough data for ReissueV1 transaction %d, expected not less then %d", l, reissueV1BodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != ReissueTransaction {
		return errors.Errorf("unexpected transaction type %d for ReissueV1 transaction", tx.Type)
	}
	tx.Version = 1
	var r Reissue
	err := r.unmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ReissueV1 transaction body")
	}
	tx.Reissue = r
	return nil
}

//Sign use given private key to calculate signature of the transaction.
//This function also calculates digest of transaction data and assigns it to ID field.
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

//Verify checks that the signature of the transaction is valid for given public key.
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

//MarshalBinary saves the transaction to its binary representation.
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

//UnmarshalBinary reads transaction from its binary representation.
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

//BurnV1 transaction allows to decrease the total supply of the existing asset. Asset must be reissuable.
type BurnV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Burn
}

func (tx BurnV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedBurnV1 creates new BurnV1 transaction with no signature and ID.
func NewUnsignedBurnV1(senderPK crypto.PublicKey, assetID crypto.Digest, amount, timestamp, fee uint64) (*BurnV1, error) {
	b, err := newBurn(senderPK, assetID, amount, timestamp, fee)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create BurnV1 transaction")
	}
	return &BurnV1{Type: BurnTransaction, Version: 1, Burn: *b}, nil
}

func (tx *BurnV1) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, burnV1BodyLen)
	buf[0] = byte(tx.Type)
	b, err := tx.Burn.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnV1 transaction to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *BurnV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < burnV1BodyLen {
		return errors.Errorf("%d bytes is not enough for BurnV1 transaction, expected not less then %d", l, burnV1BodyLen)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != BurnTransaction {
		return errors.Errorf("unexpected transaction type %d for BurnV1 transaction", tx.Type)
	}
	tx.Version = 1
	var b Burn
	err := b.unmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnV1 transaction body")
	}
	tx.Burn = b
	return nil
}

//Sign calculates and sets signature and ID of the transaction.
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

//Verify checks that the signature of the transaction is valid for the given public key.
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

//MarshalBinary saves transaction to
func (tx *BurnV1) MarshalBinary() ([]byte, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal BurnV1 transaction to bytes")
	}
	buf := make([]byte, burnV1Len)
	copy(buf, b)
	copy(buf[burnV1BodyLen:], tx.Signature[:])
	return buf, nil
}

//UnmarshalBinary reads transaction form its binary representation.
func (tx *BurnV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < burnV1Len {
		return errors.Errorf("not enough data for BurnV1 transaction, expected not less then %d, received %d", burnV1Len, l)
	}
	err := tx.bodyUnmarshalBinary(data[:burnV1BodyLen])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BurnV1 transaction")
	}
	var s crypto.Signature
	copy(s[:], data[burnV1BodyLen:burnV1BodyLen+crypto.SignatureSize])
	tx.Signature = &s
	d, err := crypto.FastHash(data[:burnV1BodyLen])
	if err != nil {
		return errors.Wrap(err, "failed to hash BurnV1 transaction")
	}
	tx.ID = &d
	return nil
}

//ExchangeV1 is a transaction to store settlement on blockchain.
type ExchangeV1 struct {
	Type           TransactionType   `json:"type"`
	Version        byte              `json:"version,omitempty"`
	ID             *crypto.Digest    `json:"id,omitempty"`
	Signature      *crypto.Signature `json:"signature,omitempty"`
	SenderPK       crypto.PublicKey  `json:"senderPublicKey"`
	BuyOrder       OrderV1           `json:"order1"`
	SellOrder      OrderV1           `json:"order2"`
	Price          uint64            `json:"price"`
	Amount         uint64            `json:"amount"`
	BuyMatcherFee  uint64            `json:"buyMatcherFee"`
	SellMatcherFee uint64            `json:"sellMatcherFee"`
	Fee            uint64            `json:"fee"`
	Timestamp      uint64            `json:"timestamp,omitempty"`
}

func (tx ExchangeV1) GetID() []byte {
	return tx.ID.Bytes()
}

func (tx ExchangeV1) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx ExchangeV1) GetBuyOrder() (OrderBody, error) {
	return tx.BuyOrder.OrderBody, nil
}

func (tx ExchangeV1) GetSellOrder() (OrderBody, error) {
	return tx.SellOrder.OrderBody, nil
}

func (tx ExchangeV1) GetPrice() uint64 {
	return tx.Price
}

func (tx ExchangeV1) GetAmount() uint64 {
	return tx.Amount
}

func (tx ExchangeV1) GetBuyMatcherFee() uint64 {
	return tx.BuyMatcherFee
}

func (tx ExchangeV1) GetSellMatcherFee() uint64 {
	return tx.SellMatcherFee
}
func (tx ExchangeV1) GetFee() uint64 {
	return tx.Fee
}

func (tx ExchangeV1) GetTimestamp() uint64 {
	return tx.Timestamp
}

func NewUnsignedExchangeV1(buy, sell OrderV1, price, amount, buyMatcherFee, sellMatcherFee, fee, timestamp uint64) (*ExchangeV1, error) {
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
	buf := make([]byte, exchangeV1FixedBodyLen+bol+sol)
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

func (tx *ExchangeV1) bodyUnmarshalBinary(data []byte) (int, error) {
	const expectedLen = exchangeV1FixedBodyLen + orderV1MinLen + orderV1MinLen
	if l := len(data); l < expectedLen {
		return 0, errors.Errorf("not enough data for ExchangeV1 transaction, expected not less then %d, received %d", expectedLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != ExchangeTransaction {
		return 0, errors.Errorf("unexpected transaction type %d for ExchangeV1 transaction", tx.Type)
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
		return 0, errors.Wrapf(err, "failed to unmarshal ExchangeV1 body from bytes")
	}
	tx.BuyOrder = bo
	n += int(bol)
	var so OrderV1
	err = so.UnmarshalBinary(data[n:])
	if err != nil {
		return 0, errors.Wrapf(err, "failed to unmarshal ExchangeV1 body from bytes")
	}
	tx.SellOrder = so
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
	tx.SenderPK = tx.BuyOrder.MatcherPK
	return n, nil
}

//Sing calculates ID and Signature of the transaction.
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

//Verify checks that signature of the transaction is valid.
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

//MarshalBinary saves the transaction to its binary representation.
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

//UnmarshalBinary loads the transaction from its binary representation.
func (tx *ExchangeV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < exchangeV1MinLen {
		return errors.Errorf("not enough data for ExchangeV1 transaction, expected not less then %d, received %d", exchangeV1MinLen, l)
	}
	if data[0] != byte(ExchangeTransaction) {
		return errors.Errorf("incorrect transaction type %d for ExchangeV1 transaction", data[0])
	}
	bl, err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeV1 transaction from bytes")
	}
	bb := data[:bl]
	data = data[bl:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	d, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ExchangeV1 transaction from bytes")
	}
	tx.ID = &d
	return nil
}

//LeaseV1 is a transaction that allows to lease Waves to other account.
type LeaseV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	Lease
}

func (tx LeaseV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedLeaseV1 creates new LeaseV1 transaction without signature and ID set.
func NewUnsignedLeaseV1(senderPK crypto.PublicKey, recipient Address, amount, fee, timestamp uint64) (*LeaseV1, error) {
	l, err := newLease(senderPK, recipient, amount, fee, timestamp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create LeaseV1 transaction")
	}
	return &LeaseV1{Type: LeaseTransaction, Version: 1, Lease: *l}, nil
}

func (tx *LeaseV1) bodyMarshalBinary() ([]byte, error) {
	rl := tx.Recipient.len
	buf := make([]byte, leaseV1BodyLen+rl)
	buf[0] = byte(tx.Type)
	b, err := tx.Lease.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseV1 transaction to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *LeaseV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseV1BodyLen {
		return errors.Errorf("not enough data for LeaseV1 transaction, expected not less then %d, received %d", leaseV1BodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != LeaseTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseV1 transaction", tx.Type)
	}
	tx.Version = 1
	var l Lease
	err := l.unmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseV1 transaction from bytes")
	}
	tx.Lease = l
	return nil
}

//Sign calculates ID and Signature of the transaction.
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

//Verify checks that the signature of the transaction is valid for the given public key.
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

//MarshalBinary saves the transaction to its binary representation.
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

//UnmarshalBinary reads the transaction from bytes slice.
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
	bl := leaseV1BodyLen + tx.Recipient.len
	b := data[:bl]
	data = data[bl:]
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

//LeaseCancelV1 transaction can be used to cancel previously created leasing.
type LeaseCancelV1 struct {
	Type      TransactionType   `json:"type"`
	Version   byte              `json:"version,omitempty"`
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	LeaseCancel
}

func (tx LeaseCancelV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedLeaseCancelV1 creates new LeaseCancelV1 transaction structure without a signature and an ID.
func NewUnsignedLeaseCancelV1(senderPK crypto.PublicKey, leaseID crypto.Digest, fee, timestamp uint64) (*LeaseCancelV1, error) {
	lc, err := newLeaseCancel(senderPK, leaseID, fee, timestamp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create LeaseCancelV1 transaction")
	}
	return &LeaseCancelV1{Type: LeaseCancelTransaction, Version: 1, LeaseCancel: *lc}, nil
}

func (tx *LeaseCancelV1) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, leaseCancelV1BodyLen)
	buf[0] = byte(tx.Type)
	b, err := tx.LeaseCancel.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal LeaseCancelV1 to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *LeaseCancelV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < leaseCancelV1BodyLen {
		return errors.Errorf("not enough data for LeaseCancelV1 transaction, expected not less then %d, received %d", leaseCancelV1BodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != LeaseCancelTransaction {
		return errors.Errorf("unexpected transaction type %d for LeaseCancelV1 transaction", tx.Type)

	}
	tx.Version = 1
	var lc LeaseCancel
	err := lc.unmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaseCancelV1 from bytes")
	}
	tx.LeaseCancel = lc
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

//Verify checks that signature of the transaction is valid for the given public key.
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

//MarshalBinary saves transaction to its binary representation.
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
	CreateAlias
}

func (tx CreateAliasV1) GetID() []byte {
	return tx.ID.Bytes()
}

func NewUnsignedCreateAliasV1(senderPK crypto.PublicKey, alias Alias, fee, timestamp uint64) (*CreateAliasV1, error) {
	ca, err := newCreateAlias(senderPK, alias, fee, timestamp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CreateAliasV1 transaction")
	}
	return &CreateAliasV1{Type: CreateAliasTransaction, Version: 1, CreateAlias: *ca}, nil
}

func (tx *CreateAliasV1) bodyMarshalBinary() ([]byte, error) {
	buf := make([]byte, createAliasV1FixedBodyLen+len(tx.Alias.Alias))
	buf[0] = byte(tx.Type)
	b, err := tx.CreateAlias.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateAliasV1 transaction body to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (tx *CreateAliasV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < createAliasV1FixedBodyLen {
		return errors.Errorf("not enough data for CreateAliasV1 transaction, expected not less then %d, received %d", createAliasV1FixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	if tx.Type != CreateAliasTransaction {
		return errors.Errorf("unexpected transaction type %d for CreateAliasV1 transaction", tx.Type)
	}
	tx.Version = 1
	var ca CreateAlias
	err := ca.unmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV1 transaction from bytes")
	}
	tx.CreateAlias = ca
	return nil
}

func (tx *CreateAliasV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasV1 transaction")
	}
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	tx.ID, err = tx.CreateAlias.id()
	if err != nil {
		return errors.Wrap(err, "failed to sign CreateAliasV1 transaction")
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
	bl := createAliasV1FixedBodyLen + len(tx.Alias.Alias)
	data = data[bl:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	tx.ID, err = tx.CreateAlias.id()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV1 transaction from bytes")
	}
	return nil
}

func (tx *CreateAliasV1) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Type      TransactionType   `json:"type"`
		Version   byte              `json:"version,omitempty"`
		ID        *crypto.Digest    `json:"id,omitempty"`
		Signature *crypto.Signature `json:"signature,omitempty"`
		SenderPK  crypto.PublicKey  `json:"senderPublicKey"`
		Alias     string            `json:"alias"`
		Fee       uint64            `json:"fee"`
		Timestamp uint64            `json:"timestamp,omitempty"`
	}{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateAliasV1 from JSON")
	}
	tx.Type = tmp.Type
	tx.Version = tmp.Version
	tx.ID = tmp.ID
	tx.Signature = tmp.Signature
	tx.SenderPK = tmp.SenderPK
	tx.Alias = Alias{aliasVersion, TestNetScheme, tmp.Alias}
	tx.Fee = tmp.Fee
	tx.Timestamp = tmp.Timestamp
	return nil
}

type MassTransferEntry struct {
	Recipient Recipient `json:"recipient"`
	Amount    uint64    `json:"amount"`
}

func (e *MassTransferEntry) MarshalBinary() ([]byte, error) {
	rb, err := e.Recipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferEntry")
	}
	rl := e.Recipient.len
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
	e.Amount = binary.BigEndian.Uint64(data[e.Recipient.len:])
	return nil
}

//MassTransferV1 is a transaction that performs multiple transfers of one asset to the accounts at once.
type MassTransferV1 struct {
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

func (tx MassTransferV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedMassTransferV1 creates new MassTransferV1 transaction structure without signature and ID.
func NewUnsignedMassTransferV1(senderPK crypto.PublicKey, asset OptionalAsset, transfers []MassTransferEntry, fee, timestamp uint64, attachment string) (*MassTransferV1, error) {
	if len(transfers) == 0 {
		return nil, errors.New("empty transfers")
	}
	for _, t := range transfers {
		if t.Amount <= 0 {
			return nil, errors.New("at least one of the transfers has non-positive amount")
		}
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	if len(attachment) > maxAttachmentLengthBytes {
		return nil, errors.New("attachment too long")
	}
	return &MassTransferV1{Type: MassTransferTransaction, Version: 1, SenderPK: senderPK, Asset: asset, Transfers: transfers, Fee: fee, Timestamp: timestamp, Attachment: Attachment(attachment)}, nil
}

func (tx *MassTransferV1) bodyAndAssetLen() (int, int) {
	n := len(tx.Transfers)
	l := 0
	if tx.Asset.Present {
		l += crypto.DigestSize
	}
	rls := 0
	for _, e := range tx.Transfers {
		rls += e.Recipient.len
	}
	al := len(tx.Attachment)
	return massTransferV1FixedLen + l + n*massTransferEntryLen + rls + al, l
}

func (tx *MassTransferV1) bodyMarshalBinary() ([]byte, error) {
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
		return nil, errors.Wrap(err, "failed to marshal MassTransferV1 transaction body to bytes")
	}
	copy(buf[p:], ab)
	p += 1 + al
	binary.BigEndian.PutUint16(buf[p:], uint16(n))
	p += 2
	for _, t := range tx.Transfers {
		tb, err := t.MarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal MassTransferV1 transaction body to bytes")
		}
		copy(buf[p:], tb)
		p += massTransferEntryLen + t.Recipient.len
	}
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	PutStringWithUInt16Len(buf[p:], tx.Attachment.String())
	return buf, nil
}

func (tx *MassTransferV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if l := len(data); l < massTransferV1MinLen {
		return errors.Errorf("not enough data for MassTransferV1 transaction, expected not less then %d, received %d", massTransferV1MinLen, l)
	}
	if tx.Type != MassTransferTransaction {
		return errors.Errorf("unexpected transaction type %d for MassTransferV1 transaction", tx.Type)
	}
	if tx.Version != 1 {
		return errors.Errorf("unexpected version %d for MassTransferV1 transaction", tx.Version)
	}
	data = data[2:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	err := tx.Asset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferV1 from bytes")
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
			return errors.Wrap(err, "failed to unmarshal MassTransferV1 transaction body from bytes")
		}
		data = data[massTransferEntryLen+e.Recipient.len:]
		entries = append(entries, e)
	}
	tx.Transfers = entries
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	at, err := StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferV1 transaction body from bytes")
	}
	tx.Attachment = Attachment(at)
	return nil
}

//Sign calculates signature and ID of the transaction.
func (tx *MassTransferV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign MassTransferV1 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign MassTransferV1 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign MassTransferV1 transaction")
	}
	return nil
}

//Verify checks that the signature is valid for the given public key.
func (tx *MassTransferV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of MassTransferV1 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary saves the transaction to its binary representation.
func (tx *MassTransferV1) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferV1 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal MassTransferV1 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal MassTransferV1 transaction to bytes")
	}
	pl := len(pb)
	buf := make([]byte, bl+pl)
	copy(buf[0:], bb)
	copy(buf[bl:], pb)
	return buf, nil
}

//UnmarshalBinary loads transaction from its binary representation.
func (tx *MassTransferV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < massTransferV1MinLen {
		return errors.Errorf("not enough data for MassTransferV1 transaction, expected not less then %d, received %d", massTransferV1MinLen, l)
	}
	if data[0] != byte(MassTransferTransaction) {
		return errors.Errorf("incorrect transaction type %d for MassTransferV1 transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferV1 transaction from bytes")
	}
	bl, _ := tx.bodyAndAssetLen()
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferV1 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal MassTransferV1 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//DataV1 is first version of the transaction that puts data to the key-value storage of an account.
type DataV1 struct {
	Type      TransactionType  `json:"type"`
	Version   byte             `json:"version,omitempty"`
	ID        *crypto.Digest   `json:"id,omitempty"`
	Proofs    *ProofsV1        `json:"proofs,omitempty"`
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	Entries   DataEntries      `json:"data"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func (tx DataV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedData creates new Data transaction without proofs.
func NewUnsignedData(senderPK crypto.PublicKey, fee, timestamp uint64) (*DataV1, error) {
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &DataV1{Type: DataTransaction, Version: 1, SenderPK: senderPK, Fee: fee, Timestamp: timestamp}, nil
}

//AppendEntry adds the entry to the transaction.
func (tx *DataV1) AppendEntry(entry DataEntry) error {
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

func (tx *DataV1) entriesLen() int {
	r := 0
	for _, e := range tx.Entries {
		r += e.binarySize()
	}
	return r
}

func (tx *DataV1) BodyMarshalBinary() ([]byte, error) {
	var p int
	n := len(tx.Entries)
	el := tx.entriesLen()
	buf := make([]byte, dataV1FixedBodyLen+el)
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
			return nil, errors.Wrap(err, "failed to marshal DataV1 transaction body to bytes")
		}
		copy(buf[p:], eb)
		p += e.binarySize()
	}
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	return buf, nil
}

func (tx *DataV1) bodyUnmarshalBinary(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if l := len(data); l < dataV1FixedBodyLen {
		return errors.Errorf("not enough data for DataV1 transaction, expected not less then %d, received %d", dataV1FixedBodyLen, l)
	}
	if tx.Type != DataTransaction {
		return errors.Errorf("unexpected transaction type %d for DataV1 transaction", tx.Type)
	}
	if tx.Version != 1 {
		return errors.Errorf("unexpected version %d for DataV1 transaction", tx.Version)
	}
	data = data[2:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	n := int(binary.BigEndian.Uint16(data))
	data = data[2:]
	for i := 0; i < n; i++ {
		var e DataEntry
		t, err := tx.extractValueType(data)
		if err != nil {
			return errors.Errorf("failed to extract type of data entry")
		}
		switch ValueType(t) {
		case Integer:
			var ie IntegerDataEntry
			err = ie.UnmarshalBinary(data)
			e = ie
		case Boolean:
			var be BooleanDataEntry
			err = be.UnmarshalBinary(data)
			e = be
		case Binary:
			var be BinaryDataEntry
			err = be.UnmarshalBinary(data)
			e = be
		case String:
			var se StringDataEntry
			err = se.UnmarshalBinary(data)
			e = se
		default:
			return errors.Errorf("unsupported ValueType %d", t)
		}
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal DataV1 transaction body from bytes")
		}
		data = data[e.binarySize():]
		err = tx.AppendEntry(e)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal DataV1 transaction body from bytes")
		}
	}
	tx.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	return nil
}

func (tx *DataV1) extractValueType(data []byte) (ValueType, error) {
	if l := len(data); l < 3 {
		return 0, errors.Errorf("not enough data to extract ValueType, expected not less than %d, received %d", 3, l)
	}
	kl := binary.BigEndian.Uint16(data)
	if l := len(data); l < int(kl)+2 {
		return 0, errors.Errorf("not enough data to extract ValueType, expected not less than %d, received %d", kl+2, l)
	}
	return ValueType(data[kl+2]), nil
}

//Sign use given secret key to calculate signature of the transaction.
func (tx *DataV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.BodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign DataV1 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign DataV1 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign DataV1 transaction")
	}
	return nil
}

//Verify chechs that the signature is valid for the given public key.
func (tx *DataV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.BodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of DataV1 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary saves the transaction to bytes.
func (tx *DataV1) MarshalBinary() ([]byte, error) {
	bb, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal DataV1 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal DataV1 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal DataV1 transaction to bytes")
	}
	pl := len(pb)
	buf := make([]byte, 1+bl+pl)
	buf[0] = 0
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads the transaction from the bytes.
func (tx *DataV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < dataV1MinLen {
		return errors.Errorf("not enough data for DataV1 transaction, expected not less then %d, received %d", dataV1MinLen, l)
	}
	if data[0] != 0 {
		return errors.Errorf("unexpected first byte %d for DataV1 transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal DataV1 transaction from bytes")
	}
	bl := dataV1FixedBodyLen + tx.entriesLen()
	bb := data[1 : 1+bl]
	data = data[1+bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal DataV1 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal DataV1 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//SetScriptV1 is a transaction to set smart script on an account.
type SetScriptV1 struct {
	Type      TransactionType  `json:"type"`
	Version   byte             `json:"version,omitempty"`
	ID        *crypto.Digest   `json:"id,omitempty"`
	Proofs    *ProofsV1        `json:"proofs,omitempty"`
	ChainID   byte             `json:"-"`
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	Script    Script           `json:"script"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func (tx SetScriptV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedSetScriptV1 creates new unsigned SetScriptV1 transaction.
func NewUnsignedSetScriptV1(chain byte, senderPK crypto.PublicKey, script []byte, fee, timestamp uint64) (*SetScriptV1, error) {
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &SetScriptV1{Type: SetScriptTransaction, Version: 1, ChainID: chain, SenderPK: senderPK, Script: script, Fee: fee, Timestamp: timestamp}, nil
}

//NonEmptyScript returns true if transaction contains non-empty script.
func (tx *SetScriptV1) NonEmptyScript() bool {
	return len(tx.Script) != 0
}

func (tx *SetScriptV1) bodyMarshalBinary() ([]byte, error) {
	var p int
	sl := 0
	if tx.NonEmptyScript() {
		sl = len(tx.Script) + 2
	}
	buf := make([]byte, setScriptV1FixedBodyLen+sl)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	buf[p] = tx.ChainID
	p++
	copy(buf[p:], tx.SenderPK[:])
	p += crypto.PublicKeySize
	PutBool(buf[p:], tx.NonEmptyScript())
	p++
	if tx.NonEmptyScript() {
		PutBytesWithUInt16Len(buf[p:], tx.Script)
		p += sl
	}
	binary.BigEndian.PutUint64(buf[p:], tx.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tx.Timestamp)
	return buf, nil
}

func (tx *SetScriptV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < setScriptV1FixedBodyLen {
		return errors.Errorf("not enough data for SetScriptV1 transaction, expected not less then %d, received %d", setScriptV1FixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	tx.ChainID = data[2]
	if tx.Type != SetScriptTransaction {
		return errors.Errorf("unexpected transaction type %d for SetScriptV1 transaction", tx.Type)
	}
	if tx.Version != 1 {
		return errors.Errorf("unexpected version %d for SetScriptV1 transaction", tx.Version)
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
			return errors.Wrap(err, "failed to unmarshal SetScriptV1 transaction body from bytes")
		}
		tx.Script = s
		data = data[2+len(s):]
	}
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

//Sign adds signature as a proof at first position.
func (tx *SetScriptV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign SetScriptV1 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign SetScriptV1 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign SetScriptV1 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *SetScriptV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of SetScriptV1 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes SetScriptV1 transaction to its bytes representation.
func (tx *SetScriptV1) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SetScriptV1 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal SetScriptV1 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SetScriptV1 transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads SetScriptV1 transaction from its binary representation.
func (tx *SetScriptV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < setScriptV1MinLen {
		return errors.Errorf("not enough data for SetScriptV1 transaction, expected not less then %d, received %d", setScriptV1MinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetScriptV1 transaction from bytes")
	}
	sl := 0
	if tx.NonEmptyScript() {
		sl = len(tx.Script) + 2
	}
	bl := setScriptV1FixedBodyLen + sl
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetScriptV1 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetScriptV1 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//SponsorshipV1 is a transaction to setup fee sponsorship for an asset.
type SponsorshipV1 struct {
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

func (tx SponsorshipV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedSponsorshipV1 creates new unsigned SponsorshipV1 transaction
func NewUnsignedSponsorshipV1(senderPK crypto.PublicKey, assetID crypto.Digest, minAssetFee, fee, timestamp uint64) (*SponsorshipV1, error) {
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &SponsorshipV1{Type: SponsorshipTransaction, Version: 1, SenderPK: senderPK, AssetID: assetID, MinAssetFee: minAssetFee, Fee: fee, Timestamp: timestamp}, nil
}

func (tx *SponsorshipV1) bodyMarshalBinary() ([]byte, error) {
	var p int
	buf := make([]byte, sponsorshipV1BodyLen)
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

func (tx *SponsorshipV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < sponsorshipV1BodyLen {
		return errors.Errorf("not enough data for SponsorshipV1 transaction body, expected %d bytes, received %d", sponsorshipV1BodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	if tx.Type != SponsorshipTransaction {
		return errors.Errorf("unexpected transaction type %d for SponsorshipV1 transaction", tx.Type)
	}
	if tx.Version != 1 {
		return errors.Errorf("unexpected version %d for SponsorshipV1 transaction", tx.Version)
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

//Sign adds signature as a proof at first position.
func (tx *SponsorshipV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign SponsorshipV1 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign SponsorshipV1 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign SponsorshipV1 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *SponsorshipV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of SponsorshipV1 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes SponsorshipV1 transaction to its bytes representation.
func (tx *SponsorshipV1) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SponsorshipV1 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal SponsorshipV1 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SponsorshipV1 transaction to bytes")
	}
	buf := make([]byte, 1+1+1+bl+len(pb))
	buf[0] = 0
	buf[1] = byte(tx.Type)
	buf[2] = tx.Version
	copy(buf[3:], bb)
	copy(buf[3+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads SponsorshipV1 from its bytes representation.
func (tx *SponsorshipV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < sponsorshipV1MinLen {
		return errors.Errorf("not enough data for SponsorshipV1 transaction, expected not less then %d, received %d", sponsorshipV1MinLen, l)
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
		return errors.Wrap(err, "failed to unmarshal SponsorshipV1 transaction from bytes")
	}
	bl := sponsorshipV1BodyLen
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SponsorshipV1 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SponsorshipV1 transaction from bytes")
	}
	tx.ID = &id
	return nil
}

//SetAssetScriptV1 is a transaction to set smart script on an asset.
type SetAssetScriptV1 struct {
	Type      TransactionType  `json:"type"`
	Version   byte             `json:"version,omitempty"`
	ID        *crypto.Digest   `json:"id,omitempty"`
	Proofs    *ProofsV1        `json:"proofs,omitempty"`
	ChainID   byte             `json:"-"`
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	AssetID   crypto.Digest    `json:"assetId"`
	Script    Script           `json:"script"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func (tx SetAssetScriptV1) GetID() []byte {
	return tx.ID.Bytes()
}

//NewUnsignedSetAssetScriptV1 creates new unsigned SetAssetScriptV1 transaction.
func NewUnsignedSetAssetScriptV1(chain byte, senderPK crypto.PublicKey, assetID crypto.Digest, script []byte, fee, timestamp uint64) (*SetAssetScriptV1, error) {
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &SetAssetScriptV1{Type: SetAssetScriptTransaction, Version: 1, ChainID: chain, SenderPK: senderPK, AssetID: assetID, Script: script, Fee: fee, Timestamp: timestamp}, nil
}

//NonEmptyScript returns true if transaction contains non-empty script.
func (tx *SetAssetScriptV1) NonEmptyScript() bool {
	return len(tx.Script) != 0
}

func (tx *SetAssetScriptV1) bodyMarshalBinary() ([]byte, error) {
	var p int
	sl := 0
	if tx.NonEmptyScript() {
		sl = len(tx.Script) + 2
	}
	buf := make([]byte, setAssetScriptV1FixedBodyLen+sl)
	buf[p] = byte(tx.Type)
	p++
	buf[p] = tx.Version
	p++
	buf[p] = tx.ChainID
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
		PutBytesWithUInt16Len(buf[p:], tx.Script)
	}
	return buf, nil
}

func (tx *SetAssetScriptV1) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < setAssetScriptV1FixedBodyLen {
		return errors.Errorf("not enough data for SetAssetScriptV1 transaction, expected not less then %d, received %d", setAssetScriptV1FixedBodyLen, l)
	}
	tx.Type = TransactionType(data[0])
	tx.Version = data[1]
	tx.ChainID = data[2]
	if tx.Type != SetAssetScriptTransaction {
		return errors.Errorf("unexpected transaction type %d for SetAssetScriptV1 transaction", tx.Type)
	}
	if tx.Version != 1 {
		return errors.Errorf("unexpected version %d for SetAssetScriptV1 transaction", tx.Version)
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
			return errors.Wrap(err, "failed to unmarshal SetAssetScriptV1 transaction body from bytes")
		}
		tx.Script = s
		data = data[2+len(s):]
	}
	return nil
}

//Sign adds signature as a proof at first position.
func (tx *SetAssetScriptV1) Sign(secretKey crypto.SecretKey) error {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign SetAssetScriptV1 transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign SetAssetScriptV1 transaction")
	}
	d, err := crypto.FastHash(b)
	tx.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign SetAssetScriptV1 transaction")
	}
	return nil
}

//Verify checks that first proof is a valid signature.
func (tx *SetAssetScriptV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := tx.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of SetAssetScriptV1 transaction")
	}
	return tx.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes SetAssetScriptV1 transaction to its bytes representation.
func (tx *SetAssetScriptV1) MarshalBinary() ([]byte, error) {
	bb, err := tx.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SetAssetScriptV1 transaction to bytes")
	}
	bl := len(bb)
	if tx.Proofs == nil {
		return nil, errors.New("failed to marshal SetAssetScriptV1 transaction to bytes: no proofs")
	}
	pb, err := tx.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SetAssetScriptV1 transaction to bytes")
	}
	buf := make([]byte, 1+bl+len(pb))
	copy(buf[1:], bb)
	copy(buf[1+bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads SetAssetScriptV1 transaction from its binary representation.
func (tx *SetAssetScriptV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < setAssetScriptV1MinLen {
		return errors.Errorf("not enough data for SetAssetScriptV1 transaction, expected not less then %d, received %d", setAssetScriptV1MinLen, l)
	}
	if v := data[0]; v != 0 {
		return errors.Errorf("unexpected first byte value %d, expected 0", v)
	}
	data = data[1:]
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetAssetScriptV1 transaction from bytes")
	}
	sl := 0
	if tx.NonEmptyScript() {
		sl = len(tx.Script) + 2
	}
	bl := setAssetScriptV1FixedBodyLen + sl
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetAssetScriptV1 transaction from bytes")
	}
	tx.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SetAssetScriptV1 transaction from bytes")
	}
	tx.ID = &id
	return nil
}
