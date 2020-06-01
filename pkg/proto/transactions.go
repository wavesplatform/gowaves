package proto

import (
	"encoding/binary"
	"encoding/json"
	"reflect"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
)

//TransactionType
type TransactionType byte

//All transaction types supported.
const (
	GenesisTransaction         TransactionType = iota + 1 // 1 - Genesis transaction
	PaymentTransaction                                    // 2 - Payment transaction
	IssueTransaction                                      // 3 - Issue transaction
	TransferTransaction                                   // 4 - Transfer transaction
	ReissueTransaction                                    // 5 - Reissue transaction
	BurnTransaction                                       // 6 - Burn transaction
	ExchangeTransaction                                   // 7 - Exchange transaction
	LeaseTransaction                                      // 8 - Lease transaction
	LeaseCancelTransaction                                // 9 - LeaseCancel transaction
	CreateAliasTransaction                                // 10 - CreateAlias transaction
	MassTransferTransaction                               // 11 - MassTransfer transaction
	DataTransaction                                       // 12 - Data transaction
	SetScriptTransaction                                  // 13 - SetScript transaction
	SponsorshipTransaction                                // 14 - Sponsorship transaction
	SetAssetScriptTransaction                             // 15 - SetAssetScript transaction
	InvokeScriptTransaction                               // 16 - InvokeScript transaction
	UpdateAssetInfoTransaction                            // 17 - UpdateAssetInfoTransaction
)

const (
	maxAttachmentLengthBytes = 140
	maxDescriptionLen        = 1000
	maxAssetNameLen          = 16
	minAssetNameLen          = 4
	maxDecimals              = 8
	maxLongValue             = ^uint64(0) >> 1

	genesisBodyLen = 1 + 8 + AddressSize + 8
	paymentBodyLen = 1 + 8 + crypto.PublicKeySize + AddressSize + 8 + 8
	issueLen       = crypto.PublicKeySize + 2 + 2 + 8 + 1 + 1 + 8 + 8
	transferLen    = crypto.PublicKeySize + 1 + 1 + 8 + 8 + 8 + 2
	reissueLen     = crypto.PublicKeySize + crypto.DigestSize + 8 + 1 + 8 + 8
	burnLen        = crypto.PublicKeySize + crypto.DigestSize + 8 + 8 + 8
	leaseLen       = crypto.PublicKeySize + 8 + 8 + 8
	leaseCancelLen = crypto.PublicKeySize + 8 + 8 + crypto.DigestSize
	createAliasLen = crypto.PublicKeySize + 2 + 8 + 8 + aliasFixedSize

	// Max allowed versions of transactions.
	MaxGenesisTransactionVersion         = 2
	MaxPaymentTransactionVersion         = 2
	MaxTransferTransactionVersion        = 3
	MaxIssueTransactionVersion           = 3
	MaxReissueTransactionVersion         = 3
	MaxBurnTransactionVersion            = 3
	MaxExchangeTransactionVersion        = 3
	MaxLeaseTransactionVersion           = 3
	MaxLeaseCancelTransactionVersion     = 3
	MaxCreateAliasTransactionVersion     = 3
	MaxMassTransferTransactionVersion    = 2
	MaxDataTransactionVersion            = 2
	MaxSetScriptTransactionVersion       = 2
	MaxSponsorshipTransactionVersion     = 2
	MaxSetAssetScriptTransactionVersion  = 2
	MaxInvokeScriptTransactionVersion    = 2
	MaxUpdateAssetInfoTransactionVersion = 1
)

var (
	bytesToTransactionsV2 = map[TransactionType]reflect.Type{
		IssueTransaction:          reflect.TypeOf(IssueWithProofs{}),
		TransferTransaction:       reflect.TypeOf(TransferWithProofs{}),
		ReissueTransaction:        reflect.TypeOf(ReissueWithProofs{}),
		BurnTransaction:           reflect.TypeOf(BurnWithProofs{}),
		ExchangeTransaction:       reflect.TypeOf(ExchangeWithProofs{}),
		LeaseTransaction:          reflect.TypeOf(LeaseWithProofs{}),
		LeaseCancelTransaction:    reflect.TypeOf(LeaseCancelWithProofs{}),
		CreateAliasTransaction:    reflect.TypeOf(CreateAliasWithProofs{}),
		DataTransaction:           reflect.TypeOf(DataWithProofs{}),
		SetScriptTransaction:      reflect.TypeOf(SetScriptWithProofs{}),
		SponsorshipTransaction:    reflect.TypeOf(SponsorshipWithProofs{}),
		SetAssetScriptTransaction: reflect.TypeOf(SetAssetScriptWithProofs{}),
		InvokeScriptTransaction:   reflect.TypeOf(InvokeScriptWithProofs{}),
	}

	bytesToTransactionsV1 = map[TransactionType]reflect.Type{
		GenesisTransaction:      reflect.TypeOf(Genesis{}),
		PaymentTransaction:      reflect.TypeOf(Payment{}),
		IssueTransaction:        reflect.TypeOf(IssueWithSig{}),
		TransferTransaction:     reflect.TypeOf(TransferWithSig{}),
		ReissueTransaction:      reflect.TypeOf(ReissueWithSig{}),
		BurnTransaction:         reflect.TypeOf(BurnWithSig{}),
		ExchangeTransaction:     reflect.TypeOf(ExchangeWithSig{}),
		LeaseTransaction:        reflect.TypeOf(LeaseWithSig{}),
		LeaseCancelTransaction:  reflect.TypeOf(LeaseCancelWithSig{}),
		CreateAliasTransaction:  reflect.TypeOf(CreateAliasWithSig{}),
		MassTransferTransaction: reflect.TypeOf(MassTransferWithProofs{}),
	}

	ProtobufTransactionsVersions = map[TransactionType]byte{
		GenesisTransaction:         2,
		PaymentTransaction:         2,
		TransferTransaction:        3,
		IssueTransaction:           3,
		ReissueTransaction:         3,
		BurnTransaction:            3,
		ExchangeTransaction:        3,
		LeaseTransaction:           3,
		LeaseCancelTransaction:     3,
		CreateAliasTransaction:     3,
		MassTransferTransaction:    2,
		DataTransaction:            2,
		SetScriptTransaction:       2,
		SponsorshipTransaction:     2,
		SetAssetScriptTransaction:  2,
		InvokeScriptTransaction:    2,
		UpdateAssetInfoTransaction: 1,
	}
)

type TransactionProofVersion byte

const (
	Signature TransactionProofVersion = iota + 1
	Proof
)

type TransactionTypeInfo struct {
	Type         TransactionType
	ProofVersion TransactionProofVersion
}

// Transaction is a set of common transaction functions.
type Transaction interface {
	// Getters which are common for all transactions.

	// GetTypeInfo() returns information which describes which Golang structure
	// do we deal with (tx type + proof/signature flag).
	// <TODO>:
	// This is temporary workaround until we have the same struct for both
	// Signature and Proofs transactions.
	GetTypeInfo() TransactionTypeInfo
	GetVersion() byte
	GetID(scheme Scheme) ([]byte, error)
	GetSenderPK() crypto.PublicKey
	GetFee() uint64
	GetTimestamp() uint64

	// Check that all transaction fields are valid.
	// This includes ranges checks, and sanity checks specific for each transaction type:
	// for example, negative amounts for transfers.
	Valid() (bool, error)

	// Set transaction ID.
	// For most transacions ID is hash of transaction body.
	// For Payment transactions ID is Signature.
	GenerateID(scheme Scheme) error
	// Sign transaction with given secret key.
	// It also sets transaction ID.
	Sign(scheme Scheme, sk crypto.SecretKey) error

	// Functions for custom binary format.
	// Serialization.
	// MarshalBinary() is analogous to MarshalSignedToProtobuf() for Protobuf.
	MarshalBinary() ([]byte, error)
	// Parsing.
	UnmarshalBinary([]byte, Scheme) error
	// Bytes without signature.
	// BodyMarshalBinary() is analogous to MarshalToProtobuf() for Protobuf.
	BodyMarshalBinary() ([]byte, error)
	// Size in bytes in binary format.
	BinarySize() int

	// Protobuf-related functions.
	// Conversion to/from Protobuf wire byte format.
	MarshalToProtobuf(scheme Scheme) ([]byte, error)
	UnmarshalFromProtobuf([]byte) error
	MarshalSignedToProtobuf(scheme Scheme) ([]byte, error)
	UnmarshalSignedFromProtobuf([]byte) error
	// Conversion to Protobuf types.
	ToProtobuf(scheme Scheme) (*g.Transaction, error)
	ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error)
}

func IsProtobufTx(tx Transaction) bool {
	protobufVersion, ok := ProtobufTransactionsVersions[tx.GetTypeInfo().Type]
	if !ok {
		return false
	}
	if tx.GetVersion() < protobufVersion {
		return false
	}
	return true
}

func MarshalTx(scheme Scheme, tx Transaction) ([]byte, error) {
	if IsProtobufTx(tx) {
		return tx.MarshalSignedToProtobuf(scheme)
	}
	return tx.MarshalBinary()
}

func MarshalTxBody(scheme Scheme, tx Transaction) ([]byte, error) {
	if IsProtobufTx(tx) {
		return tx.MarshalToProtobuf(scheme)
	}
	return tx.BodyMarshalBinary()
}

// TransactionToProtobufCommon() converts to protobuf structure with fields
// that are common for all of the transaction types.
func TransactionToProtobufCommon(scheme Scheme, tx Transaction) *g.Transaction {
	pk := tx.GetSenderPK()
	return &g.Transaction{
		ChainId:         int32(scheme),
		SenderPublicKey: pk.Bytes(),
		Timestamp:       int64(tx.GetTimestamp()),
		Version:         int32(tx.GetVersion()),
	}
}

func BytesToTransaction(tx []byte, scheme Scheme) (Transaction, error) {
	if len(tx) < 2 {
		return nil, errors.New("invalid size of transaction's bytes slice")
	}
	if tx[0] == 0 {
		transactionType, ok := bytesToTransactionsV2[TransactionType(tx[1])]
		if !ok {
			return nil, errors.Errorf("invalid transaction type %v", tx[1])
		}
		transaction, ok := reflect.New(transactionType).Interface().(Transaction)
		if !ok {
			panic("This transaction type does not implement marshal/unmarshal functions")
		}
		if err := transaction.UnmarshalBinary(tx, scheme); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal transaction")
		}
		return transaction, nil
	} else {
		transactionType, ok := bytesToTransactionsV1[TransactionType(tx[0])]
		if !ok {
			return nil, errors.Errorf("invalid transaction type %v", tx[0])
		}
		transaction, ok := reflect.New(transactionType).Interface().(Transaction)
		if !ok {
			panic("This transaction type does not implement marshal/unmarshal functions")
		}
		if err := transaction.UnmarshalBinary(tx, scheme); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal transaction")
		}
		return transaction, nil
	}
}

func BytesToTransactions(count int, txs []byte, scheme Scheme) ([]Transaction, error) {
	res := make([]Transaction, count)
	for i := 0; i < count; i++ {
		n := int(binary.BigEndian.Uint32(txs[0:4]))
		if n+4 > len(txs) {
			return nil, errors.New("invalid tx size: exceeds bytes slice bounds")
		}
		txBytes := txs[4 : n+4]
		tx, err := BytesToTransaction(txBytes, scheme)
		if err != nil {
			return nil, err
		}
		res[i] = tx
		txs = txs[4+n:]
	}
	return res, nil
}

type TransactionTypeVersion struct {
	Type    TransactionType `json:"type"`
	Version byte            `json:"version,omitempty"`
}

// Guess transaction from type and version
func GuessTransactionType(t *TransactionTypeVersion) (Transaction, error) {
	var out Transaction
	switch t.Type {
	case GenesisTransaction: // 1
		out = &Genesis{}
	case PaymentTransaction: // 2
		out = &Payment{}
	case IssueTransaction: // 3
		if t.Version >= 2 {
			out = &IssueWithProofs{}
		} else {
			out = &IssueWithSig{}
		}
	case TransferTransaction: // 4
		if t.Version >= 2 {
			out = &TransferWithProofs{}
		} else {
			out = &TransferWithSig{}
		}
	case ReissueTransaction: // 5
		if t.Version >= 2 {
			out = &ReissueWithProofs{}
		} else {
			out = &ReissueWithSig{}
		}
	case BurnTransaction: // 6
		if t.Version >= 2 {
			out = &BurnWithProofs{}
		} else {
			out = &BurnWithSig{}
		}
	case ExchangeTransaction: // 7
		if t.Version >= 2 {
			out = &ExchangeWithProofs{}
		} else {
			out = &ExchangeWithSig{}
		}
	case LeaseTransaction: // 8
		if t.Version >= 2 {
			out = &LeaseWithProofs{}
		} else {
			out = &LeaseWithSig{}
		}
	case LeaseCancelTransaction: // 9
		if t.Version >= 2 {
			out = &LeaseCancelWithProofs{}
		} else {
			out = &LeaseCancelWithSig{}
		}
	case CreateAliasTransaction: // 10
		if t.Version >= 2 {
			out = &CreateAliasWithProofs{}
		} else {
			out = &CreateAliasWithSig{}
		}
	case MassTransferTransaction: // 11
		out = &MassTransferWithProofs{}
	case DataTransaction: // 12
		out = &DataWithProofs{}
	case SetScriptTransaction: // 13
		out = &SetScriptWithProofs{}
	case SponsorshipTransaction: // 14
		out = &SponsorshipWithProofs{}
	case SetAssetScriptTransaction: // 15
		out = &SetAssetScriptWithProofs{}
	case InvokeScriptTransaction: // 16
		out = &InvokeScriptWithProofs{}
	}
	if out == nil {
		return nil, errors.Errorf("unknown transaction type %d version %d", t.Type, t.Version)
	}
	return out, nil
}

func TxVersionFromJson(data []byte) (byte, error) {
	txVersion := struct {
		Version byte `json:"version"`
	}{}
	if err := json.Unmarshal(data, &txVersion); err != nil {
		return 0, err
	}
	return txVersion.Version, nil
}

type attachmentType struct {
	Type string `json:"type"`
}

func TxAttachmentFromJson(data []byte, txType TransactionType) (Attachment, error) {
	version, err := TxVersionFromJson(data)
	if err != nil {
		return nil, err
	}
	protobufVersion, ok := ProtobufTransactionsVersions[txType]
	if !ok {
		return nil, errors.Errorf("Type %v is absent in ProtobufTransactionsVersions", txType)
	}
	if version < protobufVersion {
		return &LegacyAttachment{}, nil
	}
	tx := struct {
		Attachment attachmentType `json:"attachment"`
	}{}
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, err
	}
	switch tx.Attachment.Type {
	case "integer":
		return &IntAttachment{}, nil
	case "boolean":
		return &BoolAttachment{}, nil
	case "binary":
		return &BinaryAttachment{}, nil
	case "string":
		return &StringAttachment{}, nil
	default:
		return nil, errors.New("unknown attachment type")
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

func (tx *Genesis) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Type      TransactionType   `json:"type"`
		Version   byte              `json:"version,omitempty"`
		ID        *crypto.Signature `json:"id,omitempty"`
		Signature *crypto.Signature `json:"signature,omitempty"`
		Timestamp uint64            `json:"timestamp"`
		Recipient Address           `json:"recipient"`
		Amount    uint64            `json:"amount"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	tx.Type = tmp.Type
	tx.Version = 1
	tx.ID = tmp.ID
	tx.Signature = tmp.Signature
	tx.Timestamp = tmp.Timestamp
	tx.Recipient = tmp.Recipient
	tx.Amount = tmp.Amount
	return nil
}

func (tx Genesis) BinarySize() int {
	return 1 + 8 + AddressSize + 8
}

func (tx Genesis) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx Genesis) GetVersion() byte {
	return tx.Version
}

func (tx *Genesis) GenerateID(scheme Scheme) error {
	return tx.generateID(scheme)
}

func (tx *Genesis) Sign(scheme Scheme, sk crypto.SecretKey) error {
	if err := tx.generateID(scheme); err != nil {
		return err
	}
	tx.Signature = tx.ID
	return nil
}

func (tx Genesis) GetSenderPK() crypto.PublicKey {
	return crypto.PublicKey{}
}

func (tx *Genesis) generateID(scheme Scheme) error {
	body, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return err
	}
	id := tx.generateBodyHash(body)
	tx.ID = &id
	return nil
}

func (tx Genesis) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx Genesis) GetFee() uint64 {
	return 0
}

func (tx Genesis) GetTimestamp() uint64 {
	return tx.Timestamp
}

//NewUnsignedGenesis returns a new unsigned Genesis transaction. Actually Genesis transaction could not be signed.
func NewUnsignedGenesis(recipient Address, amount, timestamp uint64) *Genesis {
	return &Genesis{Type: GenesisTransaction, Version: 1, Timestamp: timestamp, Recipient: recipient, Amount: amount}
}

//Valid checks the validity of transaction parameters and it's signature.
func (tx Genesis) Valid() (bool, error) {
	if tx.Version < 1 || tx.Version > MaxGenesisTransactionVersion {
		return false, errors.Errorf("bad version %d for Genesis transaction", tx.Version)
	}
	if tx.Amount == 0 {
		return false, errors.New("amount should be positive")
	}
	if !validJVMLong(tx.Amount) {
		return false, errors.New("amount is too big")
	}
	if ok, err := tx.Recipient.Valid(); !ok {
		return false, errors.Wrapf(err, "invalid recipient address '%s'", tx.Recipient.String())
	}
	return true, nil
}

func (tx *Genesis) BodyMarshalBinary() ([]byte, error) {
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

func (tx *Genesis) generateBodyHash(body []byte) crypto.Signature {
	d := make([]byte, len(body)+3)
	copy(d[3:], body)
	h := crypto.MustFastHash(d)
	var s crypto.Signature
	copy(s[0:], h[:])
	copy(s[crypto.DigestSize:], h[:])
	return s
}

func (tx *Genesis) GenerateSigID(scheme Scheme) error {
	if err := tx.GenerateSig(); err != nil {
		return err
	}
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *Genesis) GenerateSig() error {
	b, err := tx.BodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to generate signature of Genesis transaction")
	}
	s := tx.generateBodyHash(b)
	tx.Signature = &s
	return nil
}

//MarshalBinary writes transaction bytes to slice of bytes.
func (tx *Genesis) MarshalBinary() ([]byte, error) {
	b, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Genesis transaction to bytes")
	}
	return b, nil
}

//UnmarshalBinary reads transaction values from the slice of bytes.
func (tx *Genesis) UnmarshalBinary(data []byte, scheme Scheme) error {
	if l := len(data); l != genesisBodyLen {
		return errors.Errorf("incorrect data length for Genesis transaction, expected %d, received %d", genesisBodyLen, l)
	}
	if data[0] != byte(GenesisTransaction) {
		return errors.Errorf("incorrect transaction type %d for Genesis transaction", data[0])
	}
	err := tx.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Genesis transaction from bytes")
	}
	err = tx.GenerateSigID(scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Genesis transaction from bytes")
	}
	return nil
}

func (tx *Genesis) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *Genesis) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	genesisTx, ok := t.(*Genesis)
	if !ok {
		return errors.New("failed to convert result to Genesis")
	}
	*tx = *genesisTx
	return nil
}

func (tx *Genesis) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *Genesis) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	genesisTx, ok := t.(*Genesis)
	if !ok {
		return errors.New("failed to convert result to Genesis")
	}
	*tx = *genesisTx
	return nil
}

func (tx *Genesis) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	addrBody := tx.Recipient.Body()
	txData := &g.Transaction_Genesis{Genesis: &g.GenesisTransactionData{
		RecipientAddress: addrBody,
		Amount:           int64(tx.Amount),
	}}
	res := TransactionToProtobufCommon(scheme, tx)
	res.Data = txData
	return res, nil
}

func (tx *Genesis) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: unsigned,
		Proofs:      proofs.Bytes(),
	}, nil
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

func (tx *Payment) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Type      TransactionType   `json:"type"`
		Version   byte              `json:"version,omitempty"`
		ID        *crypto.Signature `json:"id,omitempty"`
		Signature *crypto.Signature `json:"signature,omitempty"`
		SenderPK  crypto.PublicKey  `json:"senderPublicKey"`
		Recipient Address           `json:"recipient"`
		Amount    uint64            `json:"amount"`
		Fee       uint64            `json:"fee"`
		Timestamp uint64            `json:"timestamp"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	tx.Type = tmp.Type
	tx.Version = 1
	tx.ID = tmp.ID
	tx.Signature = tmp.Signature
	tx.SenderPK = tmp.SenderPK
	tx.Recipient = tmp.Recipient
	tx.Amount = tmp.Amount
	tx.Fee = tmp.Fee
	tx.Timestamp = tmp.Timestamp
	return nil
}

func (tx Payment) BinarySize() int {
	return 1 + crypto.SignatureSize + crypto.PublicKeySize + AddressSize + 24
}

func (tx Payment) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Signature}
}

func (tx Payment) GetVersion() byte {
	return tx.Version
}

func (tx *Payment) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		tx.ID = tx.Signature
	}
	return nil
}

func (tx Payment) GetSenderPK() crypto.PublicKey {
	return tx.SenderPK
}

func (tx Payment) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx Payment) GetFee() uint64 {
	return tx.Fee
}

func (tx Payment) GetTimestamp() uint64 {
	return tx.Timestamp
}

//NewUnsignedPayment creates new Payment transaction with empty Signature and ID fields.
func NewUnsignedPayment(senderPK crypto.PublicKey, recipient Address, amount, fee, timestamp uint64) *Payment {
	return &Payment{Type: PaymentTransaction, Version: 1, SenderPK: senderPK, Recipient: recipient, Amount: amount, Fee: fee, Timestamp: timestamp}
}

func (tx Payment) Valid() (bool, error) {
	if tx.Version < 1 || tx.Version > MaxPaymentTransactionVersion {
		return false, errors.Errorf("bad version %d for Payment transaction", tx.Version)
	}
	if ok, err := tx.Recipient.Valid(); !ok {
		return false, errors.Wrapf(err, "invalid recipient address '%s'", tx.Recipient.String())
	}
	if tx.Amount == 0 {
		return false, errors.New("amount should be positive")
	}
	if !validJVMLong(tx.Amount) {
		return false, errors.New("amount is too big")
	}
	if tx.Fee == 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return false, errors.New("fee is too big")
	}
	if x := tx.Amount + tx.Fee; !validJVMLong(x) {
		return false, errors.New("sum of amount and fee overflows JVM long")
	}
	return true, nil
}

func (tx *Payment) bodyMarshalBinary(buf []byte) error {
	buf[0] = byte(tx.Type)
	binary.BigEndian.PutUint64(buf[1:], tx.Timestamp)
	copy(buf[9:], tx.SenderPK[:])
	copy(buf[9+crypto.PublicKeySize:], tx.Recipient[:])
	binary.BigEndian.PutUint64(buf[9+crypto.PublicKeySize+AddressSize:], tx.Amount)
	binary.BigEndian.PutUint64(buf[17+crypto.PublicKeySize+AddressSize:], tx.Fee)
	return nil
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
func (tx *Payment) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	if IsProtobufTx(tx) {
		b, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		s, err := crypto.Sign(secretKey, b)
		if err != nil {
			return err
		}
		tx.ID = &s
		tx.Signature = &s
		return nil
	}
	b := tx.bodyMarshalBinaryBuffer()
	err := tx.bodyMarshalBinary(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign Payment transaction")
	}
	d := make([]byte, len(b)+3)
	copy(d[3:], b)
	s, err := crypto.Sign(secretKey, d)
	if err != nil {
		return errors.Wrap(err, "failed to sign Payment transaction")
	}
	tx.ID = &s
	tx.Signature = &s
	return nil
}

func (tx *Payment) BodyMarshalBinary() ([]byte, error) {
	b := tx.bodyMarshalBinaryBuffer()
	err := tx.bodyMarshalBinary(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign Payment transaction")
	}
	d := make([]byte, len(b)+3)
	copy(d[3:], b)
	return d, nil
}

//Verify checks that the Signature is valid for given public key.
func (tx *Payment) Verify(scheme Scheme, publicKey crypto.PublicKey) (bool, error) {
	if tx.Signature == nil {
		return false, errors.New("empty signature")
	}
	if IsProtobufTx(tx) {
		b, err := tx.MarshalToProtobuf(scheme)
		if err != nil {
			return false, err
		}
		return crypto.Verify(publicKey, *tx.Signature, b), nil
	}
	b := tx.bodyMarshalBinaryBuffer()
	err := tx.bodyMarshalBinary(b)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify Payment transaction")
	}
	d := make([]byte, len(b)+3)
	copy(d[3:], b)
	return crypto.Verify(publicKey, *tx.Signature, d), nil
}

//MarshalBinary returns a bytes representation of Payment transaction.
func (tx *Payment) MarshalBinary() ([]byte, error) {
	b := tx.bodyMarshalBinaryBuffer()
	err := tx.bodyMarshalBinary(b)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, paymentBodyLen+crypto.SignatureSize)
	copy(buf, b)
	if tx.Signature == nil {
		return nil, errors.New("marshaling unsigned transaction")
	}
	copy(buf[paymentBodyLen:], tx.Signature[:])
	return buf, nil
}

func (tx *Payment) bodyMarshalBinaryBuffer() []byte {
	return make([]byte, paymentBodyLen)
}

//MarshalBinary returns a bytes representation of Payment transaction.
func (tx *Payment) MarshalBinary2(buf []byte) ([]byte, error) {
	b := tx.bodyMarshalBinaryBuffer()
	err := tx.bodyMarshalBinary(b)
	if err != nil {
		return nil, err
	}
	copy(buf, b)
	copy(buf[paymentBodyLen:], tx.Signature[:])
	return buf, nil
}

//UnmarshalBinary reads Payment transaction from its binary representation.
func (tx *Payment) UnmarshalBinary(data []byte, scheme Scheme) error {
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
	err = tx.GenerateID(scheme)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Payment transaction from bytes")
	}
	return nil
}

func (tx *Payment) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *Payment) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	paymentTx, ok := t.(*Payment)
	if !ok {
		return errors.New("failed to convert result to Payment")
	}
	*tx = *paymentTx
	return nil
}

func (tx *Payment) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *Payment) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	paymentTx, ok := t.(*Payment)
	if !ok {
		return errors.New("failed to convert result to Payment")
	}
	*tx = *paymentTx
	return nil
}

func (tx *Payment) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	addrBody := tx.Recipient.Body()
	txData := &g.Transaction_Payment{Payment: &g.PaymentTransactionData{
		RecipientAddress: addrBody,
		Amount:           int64(tx.Amount),
	}}
	fee := &g.Amount{AssetId: nil, Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}

func (tx *Payment) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	proofs := NewProofsFromSignature(tx.Signature)
	return &g.SignedTransaction{
		Transaction: unsigned,
		Proofs:      proofs.Bytes(),
	}, nil
}

type Issue struct {
	SenderPK    crypto.PublicKey `json:"senderPublicKey"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Quantity    uint64           `json:"quantity"`
	Decimals    byte             `json:"decimals"`
	Reissuable  bool             `json:"reissuable"`
	Timestamp   uint64           `json:"timestamp,omitempty"`
	Fee         uint64           `json:"fee"`
}

func (i Issue) BinarySize() int {
	return crypto.PublicKeySize + len(i.Name) + 2 + len(i.Description) + 2 + 8 + 1 + 1 + 16
}

func (i Issue) GetSenderPK() crypto.PublicKey {
	return i.SenderPK
}

func (i Issue) GetFee() uint64 {
	return i.Fee
}

func (i Issue) GetTimestamp() uint64 {
	return i.Timestamp
}

func (i Issue) Valid() (bool, error) {
	if i.Quantity == 0 {
		return false, errors.New("quantity should be positive")
	}
	if !validJVMLong(i.Quantity) {
		return false, errors.New("quantity is too big")
	}
	if i.Fee == 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(i.Fee) {
		return false, errors.New("fee is too big")
	}
	if l := len(i.Name); l < minAssetNameLen || l > maxAssetNameLen {
		return false, errors.New("incorrect number of bytes in the asset's name")
	}
	if l := len(i.Description); l > maxDescriptionLen {
		return false, errors.New("incorrect number of bytes in the asset's description")
	}
	if i.Decimals > maxDecimals {
		return false, errors.Errorf("incorrect decimals, should be no more then %d", maxDecimals)
	}
	return true, nil
}

func (i Issue) marshalBinary() ([]byte, error) {
	nl := len(i.Name)
	dl := len(i.Description)
	buf := make([]byte, issueLen+nl+dl)
	p := 0
	copy(buf[p:], i.SenderPK[:])
	p += crypto.PublicKeySize
	PutStringWithUInt16Len(buf[p:], i.Name)
	p += 2 + nl
	PutStringWithUInt16Len(buf[p:], i.Description)
	p += 2 + dl
	binary.BigEndian.PutUint64(buf[p:], i.Quantity)
	p += 8
	buf[p] = i.Decimals
	p++
	PutBool(buf[p:], i.Reissuable)
	p++
	binary.BigEndian.PutUint64(buf[p:], i.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], i.Timestamp)
	return buf, nil
}

func (i *Issue) UnmarshalBinary(data []byte) error {
	if l := len(data); l < issueLen {
		return errors.Errorf("%d is not enough bytes for Issue, expected not less then %d", l, issueLen)
	}
	copy(i.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	i.Name, err = StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Name")
	}
	data = data[2+len(i.Name):]
	i.Description, err = StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Description")
	}
	data = data[2+len(i.Description):]
	i.Quantity = binary.BigEndian.Uint64(data)
	data = data[8:]
	i.Decimals = data[0]
	data = data[1:]
	i.Reissuable, err = Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Reissuable")
	}
	data = data[1:]
	i.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	i.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

func (i *Issue) ToProtobuf() *g.Transaction_Issue {
	return &g.Transaction_Issue{Issue: &g.IssueTransactionData{
		Name:        i.Name,
		Description: i.Description,
		Amount:      int64(i.Quantity),
		Decimals:    int32(i.Decimals),
		Reissuable:  i.Reissuable,
	}}
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

func (tr Transfer) BinarySize() int {
	aaSize := tr.AmountAsset.BinarySize()
	faSize := tr.FeeAsset.BinarySize()
	return crypto.PublicKeySize + aaSize + faSize + 24 + tr.Recipient.BinarySize() + tr.attachmentSize() + 2
}

func (tr Transfer) GetSenderPK() crypto.PublicKey {
	return tr.SenderPK
}

func (tr Transfer) GetFee() uint64 {
	return tr.Fee
}

func (tr Transfer) GetTimestamp() uint64 {
	return tr.Timestamp
}

func (tr Transfer) Valid() (bool, error) {
	if tr.Amount == 0 {
		return false, errors.New("amount should be positive")
	}
	if !validJVMLong(tr.Amount) {
		return false, errors.New("amount is too big")
	}
	if tr.Fee == 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(tr.Fee) {
		return false, errors.New("fee is too big")
	}
	if x := tr.Amount + tr.Fee; !validJVMLong(x) {
		return false, errors.New("sum of amount and fee overflows JVM long")
	}
	if tr.attachmentSize() > maxAttachmentLengthBytes {
		return false, errors.New("attachment is too long")
	}
	if ok, err := tr.Recipient.Valid(); !ok {
		return false, errors.Wrapf(err, "invalid recipient '%s'", tr.Recipient.String())
	}
	return true, nil
}

func (tr *Transfer) attachmentSize() int {
	if tr.Attachment != nil {
		return tr.Attachment.Size()
	}
	return 0
}

func (tr *Transfer) marshalBinary() ([]byte, error) {
	p := 0
	aal := 0
	if tr.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	fal := 0
	if tr.FeeAsset.Present {
		fal += crypto.DigestSize
	}
	rb, err := tr.Recipient.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	rl := len(rb)
	att, ok := tr.Attachment.(*LegacyAttachment)
	if !ok {
		return nil, errors.New("binary format is only defined for untyped attachments")
	}
	atl := att.Size()
	buf := make([]byte, transferLen+aal+fal+atl+rl)
	copy(buf[p:], tr.SenderPK[:])
	p += crypto.PublicKeySize
	aab, err := tr.AmountAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	copy(buf[p:], aab)
	p += 1 + aal
	fab, err := tr.FeeAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	copy(buf[p:], fab)
	p += 1 + fal
	binary.BigEndian.PutUint64(buf[p:], tr.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tr.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], tr.Fee)
	p += 8
	copy(buf[p:], rb)
	p += rl
	attBytes, err := att.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Transfer body")
	}
	PutBytesWithUInt16Len(buf[p:], attBytes)
	return buf, nil
}

func (tr *Transfer) Serialize(s *serializer.Serializer) error {
	err := s.Bytes(tr.SenderPK[:])
	if err != nil {
		return err
	}
	err = tr.AmountAsset.Serialize(s)
	if err != nil {
		return err
	}
	err = tr.FeeAsset.Serialize(s)
	if err != nil {
		return err
	}
	err = s.Uint64(tr.Timestamp)
	if err != nil {
		return err
	}
	err = s.Uint64(tr.Amount)
	if err != nil {
		return err
	}
	err = s.Uint64(tr.Fee)
	if err != nil {
		return err
	}
	err = tr.Recipient.Serialize(s)
	if err != nil {
		return err
	}
	att, ok := tr.Attachment.(*LegacyAttachment)
	if !ok {
		return errors.New("binary format is only defined for untyped attachments")
	}
	attBytes, err := att.Bytes()
	if err != nil {
		return err
	}
	err = s.BytesWithUInt16Len(attBytes)
	if err != nil {
		return err
	}
	return nil
}

func (tr *Transfer) UnmarshalBinary(data []byte) error {
	if l := len(data); l < transferLen {
		return errors.Errorf("%d bytes is not enough for Transfer body, expected not less then %d bytes", l, transferLen)
	}
	copy(tr.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	err := tr.AmountAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Transfer body from bytes")
	}
	data = data[1:]
	if tr.AmountAsset.Present {
		data = data[crypto.DigestSize:]
	}
	err = tr.FeeAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Transfer body from bytes")
	}
	data = data[1:]
	if tr.FeeAsset.Present {
		data = data[crypto.DigestSize:]
	}
	tr.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	tr.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	tr.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	err = tr.Recipient.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Transfer body from bytes")
	}
	data = data[tr.Recipient.len:]
	a, err := BytesWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Transfer body from bytes")
	}
	tr.Attachment = &LegacyAttachment{Value: a}
	return nil
}

func (tr *Transfer) ToProtobuf() (*g.Transaction_Transfer, error) {
	rcpProto, err := tr.Recipient.ToProtobuf()
	if err != nil {
		return nil, err
	}
	var att *g.Attachment = nil
	if tr.Attachment != nil {
		att = tr.Attachment.ToProtobuf()
	}
	return &g.Transaction_Transfer{Transfer: &g.TransferTransactionData{
		Recipient:  rcpProto,
		Amount:     &g.Amount{AssetId: tr.AmountAsset.ToID(), Amount: int64(tr.Amount)},
		Attachment: att,
	}}, nil
}

type Reissue struct {
	SenderPK   crypto.PublicKey `json:"senderPublicKey"`
	AssetID    crypto.Digest    `json:"assetId"`
	Quantity   uint64           `json:"quantity"`
	Reissuable bool             `json:"reissuable"`
	Timestamp  uint64           `json:"timestamp,omitempty"`
	Fee        uint64           `json:"fee"`
}

func (r Reissue) BinarySize() int {
	return crypto.PublicKeySize + crypto.DigestSize + 24 + 1
}

func (r Reissue) ToProtobuf() *g.Transaction_Reissue {
	return &g.Transaction_Reissue{Reissue: &g.ReissueTransactionData{
		AssetAmount: &g.Amount{AssetId: r.AssetID.Bytes(), Amount: int64(r.Quantity)},
		Reissuable:  r.Reissuable,
	}}
}

func (r Reissue) GetSenderPK() crypto.PublicKey {
	return r.SenderPK
}

func (r Reissue) GetFee() uint64 {
	return r.Fee
}

func (r Reissue) GetTimestamp() uint64 {
	return r.Timestamp
}

func (r Reissue) Valid() (bool, error) {
	if r.Quantity == 0 {
		return false, errors.New("quantity should be positive")
	}
	if !validJVMLong(r.Quantity) {
		return false, errors.New("quantity is too big")
	}
	if r.Fee == 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(r.Fee) {
		return false, errors.New("fee is too big")
	}
	return true, nil
}

func (r *Reissue) marshalBinary() ([]byte, error) {
	p := 0
	buf := make([]byte, reissueLen)
	copy(buf[p:], r.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], r.AssetID[:])
	p += crypto.DigestSize
	binary.BigEndian.PutUint64(buf[p:], r.Quantity)
	p += 8
	PutBool(buf[p:], r.Reissuable)
	p++
	binary.BigEndian.PutUint64(buf[p:], r.Fee)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], r.Timestamp)
	return buf, nil
}

func (r *Reissue) UnmarshalBinary(data []byte) error {
	if l := len(data); l < reissueLen {
		return errors.Errorf("%d bytes is not enough for Reissue body, expected not less then %d bytes", l, reissueLen)
	}
	copy(r.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(r.AssetID[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	r.Quantity = binary.BigEndian.Uint64(data)
	data = data[8:]
	var err error
	r.Reissuable, err = Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Reissuable")
	}
	data = data[1:]
	r.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	r.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

type Burn struct {
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	AssetID   crypto.Digest    `json:"assetId"`
	Amount    uint64           `json:"amount"`
	Timestamp uint64           `json:"timestamp,omitempty"`
	Fee       uint64           `json:"fee"`
}

func (b Burn) BinarySize() int {
	return crypto.PublicKeySize + crypto.DigestSize + 24
}

func (b Burn) ToProtobuf() *g.Transaction_Burn {
	return &g.Transaction_Burn{Burn: &g.BurnTransactionData{
		AssetAmount: &g.Amount{AssetId: b.AssetID.Bytes(), Amount: int64(b.Amount)},
	}}
}

func (b Burn) GetSenderPK() crypto.PublicKey {
	return b.SenderPK
}

func (b Burn) GetFee() uint64 {
	return b.Fee
}

func (b Burn) GetTimestamp() uint64 {
	return b.Timestamp
}

func (b Burn) Valid() (bool, error) {
	if !validJVMLong(b.Amount) {
		return false, errors.New("amount is too big")
	}
	if b.Fee == 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(b.Fee) {
		return false, errors.New("fee is too big")
	}
	return true, nil
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

func (b *Burn) UnmarshalBinary(data []byte) error {
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

type Exchange interface {
	GetID(scheme Scheme) ([]byte, error)
	GetSenderPK() crypto.PublicKey
	GetBuyOrder() (Order, error)
	GetSellOrder() (Order, error)
	GetOrder1() Order
	GetOrder2() Order
	GetPrice() uint64
	GetAmount() uint64
	GetBuyMatcherFee() uint64
	GetSellMatcherFee() uint64
	GetFee() uint64
	GetTimestamp() uint64
}

type Lease struct {
	SenderPK  crypto.PublicKey `json:"senderPublicKey"`
	Recipient Recipient        `json:"recipient"`
	Amount    uint64           `json:"amount"`
	Fee       uint64           `json:"fee"`
	Timestamp uint64           `json:"timestamp,omitempty"`
}

func (l Lease) BinarySize() int {
	return crypto.PublicKeySize + l.Recipient.BinarySize() + 24
}

func (l Lease) ToProtobuf() (*g.Transaction_Lease, error) {
	rcpProto, err := l.Recipient.ToProtobuf()
	if err != nil {
		return nil, err
	}
	return &g.Transaction_Lease{Lease: &g.LeaseTransactionData{
		Recipient: rcpProto,
		Amount:    int64(l.Amount),
	}}, nil
}

func (l Lease) GetSenderPK() crypto.PublicKey {
	return l.SenderPK
}

func (l Lease) GetFee() uint64 {
	return l.Fee
}

func (l Lease) GetTimestamp() uint64 {
	return l.Timestamp
}

func (l Lease) Valid() (bool, error) {
	if ok, err := l.Recipient.Valid(); !ok {
		return false, errors.Wrap(err, "failed to create new unsigned Lease transaction")
	}
	if l.Amount == 0 {
		return false, errors.New("amount should be positive")
	}
	if !validJVMLong(l.Amount) {
		return false, errors.New("amount is too big")
	}
	if l.Fee == 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(l.Fee) {
		return false, errors.New("fee is too big")
	}
	if !validJVMLong(l.Amount + l.Fee) {
		return false, errors.New("sum of amount and fee overflows JVM long")
	}
	//TODO: check that sender and recipient is not the same
	return true, nil
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

func (l *Lease) UnmarshalBinary(data []byte) error {
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

func (lc LeaseCancel) BinarySize() int {
	return crypto.PublicKeySize + crypto.DigestSize + 16
}

func (lc LeaseCancel) ToProtobuf() *g.Transaction_LeaseCancel {
	return &g.Transaction_LeaseCancel{LeaseCancel: &g.LeaseCancelTransactionData{
		LeaseId: lc.LeaseID.Bytes(),
	}}
}

func (lc LeaseCancel) GetSenderPK() crypto.PublicKey {
	return lc.SenderPK
}

func (lc LeaseCancel) GetFee() uint64 {
	return lc.Fee
}

func (lc LeaseCancel) GetTimestamp() uint64 {
	return lc.Timestamp
}

func (lc LeaseCancel) Valid() (bool, error) {
	if lc.Fee == 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(lc.Fee) {
		return false, errors.New("fee is too big")
	}
	return true, nil
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

func (lc *LeaseCancel) UnmarshalBinary(data []byte) error {
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

func (ca CreateAlias) BinarySize() int {
	return crypto.PublicKeySize + 16 + 2 + ca.Alias.BinarySize()
}

func (ca CreateAlias) ToProtobuf() *g.Transaction_CreateAlias {
	return &g.Transaction_CreateAlias{CreateAlias: &g.CreateAliasTransactionData{
		Alias: ca.Alias.Alias,
	}}
}

func (ca CreateAlias) GetSenderPK() crypto.PublicKey {
	return ca.SenderPK
}

func (ca CreateAlias) GetFee() uint64 {
	return ca.Fee
}

func (ca CreateAlias) GetTimestamp() uint64 {
	return ca.Timestamp
}

func (ca CreateAlias) Valid() (bool, error) {
	if ca.Fee == 0 {
		return false, errors.New("fee should be positive")
	}
	if !validJVMLong(ca.Fee) {
		return false, errors.New("fee is too big")
	}
	ok, err := ca.Alias.Valid()
	if !ok {
		return false, err
	}
	return true, nil
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

func (ca *CreateAlias) UnmarshalBinary(data []byte) error {
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

func validJVMLong(x uint64) bool {
	return x <= maxLongValue
}
