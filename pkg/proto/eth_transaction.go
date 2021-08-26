package proto

import (
	"bytes"
	stderr "errors"
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"io"
	"math/big"
)

// EthereumTxType is an ethereum transaction type.
type EthereumTxType byte

const (
	LegacyTxType EthereumTxType = iota
	AccessListTxType
	DynamicFeeTxType
)

func (e EthereumTxType) String() string {
	switch e {
	case LegacyTxType:
		return "LegacyTxType"
	case AccessListTxType:
		return "AccessListTxType"
	case DynamicFeeTxType:
		return "DynamicFeeTxType"
	default:
		return ""
	}
}

var (
	ErrInvalidSig         = errors.New("invalid transaction v, r, s values")
	ErrTxTypeDecode       = stderr.New("expected RLP list or RLP bytes")
	ErrTxTypeNotSupported = stderr.New("transaction type not supported")
	errEmptyTypedTx       = stderr.New("empty typed transaction bytes")
)

type fastRLPSignerHasher interface {
	signerHashFastRLP(chainID *big.Int, arena *fastrlp.Arena) *fastrlp.Value
}

type RLPDecoder interface {
	DecodeRLP([]byte) error
}

type RLPEncoder interface {
	EncodeRLP(io.Writer) error
}

//type fastRLPUnmarshaler interface {
//	unmarshalFromFastRLP(value *fastrlp.Value) error
//}

type fastRLPMarshaler interface {
	marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value
}

type EthereumTxData interface {
	ethereumTxType() EthereumTxType
	copy() EthereumTxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	accessList() EthereumAccessList
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *EthereumAddress

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)

	fastRLPMarshaler
	fastRLPSignerHasher
}

type EthereumTransaction struct {
	inner           EthereumTxData
	innerBinarySize int
	id              *crypto.Digest
	senderPK        *EthereumPublicKey
}

func (tx *EthereumTransaction) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{
		Type:         EthereumMetamaskTransaction,
		ProofVersion: Proof,
	}
}

func (tx *EthereumTransaction) GetVersion() byte {
	// TODO(nickeskov): Is that right?
	return byte(tx.EthereumTxType())
}

func (tx *EthereumTransaction) GetID(scheme Scheme) ([]byte, error) {
	if tx.id == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.id.Bytes(), nil
}

func (tx *EthereumTransaction) GetSender(scheme Scheme) (Address, error) {
	return tx.From()
}

func (tx *EthereumTransaction) GetFee() uint64 {
	// nickeskov: in scala node this is "gasLimit" field.
	return tx.Gas()
}

func (tx *EthereumTransaction) GetTimestamp() uint64 {
	return tx.Nonce()
}

func (tx *EthereumTransaction) Validate() (Transaction, error) {
	if tx.senderPK != nil {
		return tx, nil
	}
	signer := MakeEthereumSigner(tx.ChainId())
	senderPK, err := signer.SenderPK(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate EthereumTransaction")
	}
	tx.senderPK = senderPK
	return tx, nil
}

func (tx *EthereumTransaction) GenerateID(scheme Scheme) error {
	if tx.id != nil {
		return nil
	}
	body, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return err
	}
	id := crypto.MustFastHash(body)
	tx.id = &id
	return nil
}

func (tx *EthereumTransaction) Sign(_ Scheme, _ crypto.SecretKey) error {
	// TODO(nickeskov_: Do we need it?
	return errors.New("Sign method for EthereumTransaction isn't implemented")
}

func (tx *EthereumTransaction) MarshalBinary() ([]byte, error) {
	rlpData, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary ethereum transaction")
	}
	data := make([]byte, 1+len(rlpData))
	data[0] = byte(tx.GetTypeInfo().Type)
	copy(data[1:], rlpData)
	return data, nil
}

func (tx *EthereumTransaction) UnmarshalBinary(bytes []byte, scheme Scheme) error {
	if l := len(bytes); l < 1 {
		return errors.New("failed to UnmarshalBinary ethereum transaction, received empty data")
	}
	if bytes[0] != byte(tx.GetTypeInfo().Type) {
		return errors.Errorf("incorrect transaction type %d for EthereumTransaction transaction", bytes[0])
	}

	bytes = bytes[1:]
	if err := tx.DecodeRLP(bytes); err != nil {
		return errors.Wrap(err, "failed to UnmarshalBinary ethereum transaction from RLP")
	}
	if err := tx.GenerateID(scheme); err != nil {
		return err
	}
	return nil
}

func (tx *EthereumTransaction) BodyMarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	b.Grow(tx.innerBinarySize)
	if err := tx.EncodeRLP(&b); err != nil {
		return nil, errors.Wrapf(err,
			"failed to marshal ethereum transaction to RLP, ehtTxType %q",
			tx.EthereumTxType().String(),
		)
	}
	return b.Bytes(), nil
}

func (tx *EthereumTransaction) bodyUnmarshalBinary(rlpData []byte) error {
	if err := tx.DecodeRLP(rlpData); err != nil {
		return errors.Wrapf(err, "failed to unmarshal ethereum transaction from RLP")
	}
	return nil
}

func (tx *EthereumTransaction) BinarySize() int {
	return tx.GetTypeInfo().Type.BinarySize() + tx.innerBinarySize
}

func (tx *EthereumTransaction) MarshalToProtobuf(_ Scheme) ([]byte, error) {
	return nil, errors.New("EthereumTransaction does not support 'MarshalToProtobuf' method.")
}

func (tx *EthereumTransaction) UnmarshalFromProtobuf(_ []byte) error {
	return errors.New("EthereumTransaction does not support 'UnmarshalFromProtobuf' method.")
}

func (tx *EthereumTransaction) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *EthereumTransaction) UnmarshalSignedFromProtobuf(bytes []byte) error {
	t, err := SignedTxFromProtobuf(bytes)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal from protobuf ethereum transaction")
	}
	ethTx, ok := t.(*EthereumTransaction)
	if !ok {
		return errors.Errorf(
			"failed to cast unmarshalled result '%T' to '*EthereumTransaction' type",
			t,
		)
	}
	*tx = *ethTx
	return nil
}

func (tx *EthereumTransaction) ToProtobuf(_ Scheme) (*g.Transaction, error) {
	return nil, errors.New("EthereumTransaction does not support 'ToProtobuf' method.")
}

func (tx *EthereumTransaction) ToProtobufSigned(_ Scheme) (*g.SignedTransaction, error) {
	// nickeskov: marshal body to rlp
	rlpData, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to marshal binary EthereumTransaction, type %q",
			tx.EthereumTxType().String(),
		)
	}
	wrapped := g.SignedTransaction{
		Transaction: &g.SignedTransaction_EthereumTransaction{
			EthereumTransaction: rlpData,
		},
	}
	return &wrapped, nil
}

// EthereumTxType returns the transaction type.
func (tx *EthereumTransaction) EthereumTxType() EthereumTxType {
	return tx.inner.ethereumTxType()
}

// ChainId returns the EIP155 chain ID of the transaction. The return value will always be
// non-nil. For legacy transactions which are not replay-protected, the return value is
// zero.
func (tx *EthereumTransaction) ChainId() *big.Int {
	return tx.inner.chainID()
}

// Data returns the input data of the transaction.
func (tx *EthereumTransaction) Data() []byte { return tx.inner.data() }

// AccessList returns the access list of the transaction.
func (tx *EthereumTransaction) AccessList() EthereumAccessList { return tx.inner.accessList() }

// Gas returns the gas limit of the transaction.
func (tx *EthereumTransaction) Gas() uint64 { return tx.inner.gas() }

// GasPrice returns the gas price of the transaction.
func (tx *EthereumTransaction) GasPrice() *big.Int { return copyBigInt(tx.inner.gasPrice()) }

// GasTipCap returns the gasTipCap per gas of the transaction.
func (tx *EthereumTransaction) GasTipCap() *big.Int { return copyBigInt(tx.inner.gasTipCap()) }

// GasFeeCap returns the fee cap per gas of the transaction.
func (tx *EthereumTransaction) GasFeeCap() *big.Int { return copyBigInt(tx.inner.gasFeeCap()) }

// Value returns the ether amount of the transaction.
func (tx *EthereumTransaction) Value() *big.Int { return copyBigInt(tx.inner.value()) }

// Nonce returns the sender account nonce of the transaction.
func (tx *EthereumTransaction) Nonce() uint64 { return tx.inner.nonce() }

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *EthereumTransaction) To() *EthereumAddress { return tx.inner.to().copy() }

// From returns the sender address of the transaction.
// Returns error if transaction doesn't pass validation.
func (tx *EthereumTransaction) From() (EthereumAddress, error) {
	if _, err := tx.Validate(); err != nil {
		return EthereumAddress{}, err
	}
	addr := tx.senderPK.EthereumAddress()
	return addr, nil
}

// FromPK returns the sender public key of the transaction.
// Returns error if transaction doesn't pass validation.
func (tx *EthereumTransaction) FromPK() (*EthereumPublicKey, error) {
	if _, err := tx.Validate(); err != nil {
		return nil, err
	}
	return tx.senderPK.copy(), nil
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *EthereumTransaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.inner.rawSignatureValues()
}

//func (tx *EthereumTransaction) Hash() EthereumHash {
//	// TODO(nickeskov): implement me
//	panic("implement me")
//}

func (tx *EthereumTransaction) DecodeRLP(rlpData []byte) error {
	parser := fastrlp.Parser{}
	rlpVal, err := parser.Parse(rlpData)
	if err != nil {
		return err
	}
	return tx.unmarshalFromFastRLP(rlpVal)
}

func (tx *EthereumTransaction) unmarshalFromFastRLP(value *fastrlp.Value) error {
	switch value.Type() {
	case fastrlp.TypeArray:
		// nickeskov: It's a legacy transaction.
		var inner EthereumLegacyTx
		if err := inner.unmarshalFromFastRLP(value); err != nil {
			return errors.Wrapf(err,
				"failed to unmarshal from RLP ethereum legacy transaction, ethereumTxType %q",
				LegacyTxType.String(),
			)
		}
		tx.inner = &inner
	case fastrlp.TypeBytes:
		// nickeskov: It's an EIP-2718 typed TX envelope.
		typedTxBytes, err := value.Bytes()
		if err != nil {
			return errors.Wrap(err, "failed to represent RLP value as bytes")
		}
		inner, err := tx.decodeTyped(typedTxBytes)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal from RLP ethereum typed transaction")
		}
		tx.inner = inner
	default:
		return ErrTxTypeDecode
	}
	tx.innerBinarySize = int(value.Len())
	return nil
}

func (tx EthereumTransaction) EncodeRLP(w io.Writer) error {
	arena := &fastrlp.Arena{}
	var fastrlpTx *fastrlp.Value
	// nickeskov: maybe use buffer pool?
	if tx.EthereumTxType() == LegacyTxType {
		fastrlpTx = tx.inner.marshalToFastRLP(arena)
	} else {
		fastrlpTx = tx.encodeTyped(arena)
	}
	if _, err := w.Write(fastrlpTx.MarshalTo(nil)); err != nil {
		return err
	}
	return nil
}

// decodeTyped decodes a typed transaction from the canonical format.
func (tx *EthereumTransaction) decodeTyped(rlpData []byte) (EthereumTxData, error) {
	if len(rlpData) == 0 {
		return nil, errEmptyTypedTx
	}
	switch txType, rlpData := rlpData[0], rlpData[1:]; EthereumTxType(txType) {
	case AccessListTxType:
		var inner EthereumAccessListTx
		if err := inner.DecodeRLP(rlpData); err != nil {
			return nil, errors.Wrapf(err,
				"failed to unmarshal ethereum tx from RLP, ethereumTxType %q",
				AccessListTxType.String(),
			)
		}
		return &inner, nil
	case DynamicFeeTxType:
		var inner EthereumDynamicFeeTx
		if err := inner.DecodeRLP(rlpData); err != nil {
			return nil, errors.Wrapf(err,
				"failed to unmarshal ethereum tx from RLP, ethereumTxType %q",
				DynamicFeeTxType.String(),
			)
		}
		return &inner, nil
	default:
		return nil, ErrTxTypeNotSupported
	}
}

// encodeTyped writes the canonical encoding of a typed transaction to w.
func (tx *EthereumTransaction) encodeTyped(arena *fastrlp.Arena) *fastrlp.Value {
	rlpMarshaledTx := []byte{byte(tx.EthereumTxType())}
	typedTxVal := tx.inner.marshalToFastRLP(arena)
	rlpMarshaledTx = typedTxVal.MarshalTo(rlpMarshaledTx)
	return arena.NewBytes(rlpMarshaledTx)
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// Protected says whether the transaction is replay-protected.
func (tx *EthereumTransaction) Protected() bool {
	switch tx := tx.inner.(type) {
	case *EthereumLegacyTx:
		return tx.V != nil && isProtectedV(tx.V)
	default:
		return true
	}
}
