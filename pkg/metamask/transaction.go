package metamask

import (
	stderr "errors"
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"golang.org/x/crypto/sha3"
	"io"
	"math/big"
)

// Ethereum transaction types.
const (
	LegacyTxType byte = iota
	AccessListTxType
	DynamicFeeTxType
)

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

type TxData interface {
	txType() byte
	copy() TxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	accessList() AccessList
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *Address

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)

	fastRLPMarshaler
	fastRLPSignerHasher
}

type Transaction struct {
	inner TxData
}

func NewTx(inner TxData) Transaction {
	var tx Transaction
	tx.setDecoded(inner)
	return tx
}

// Type returns the transaction type.
func (tx *Transaction) Type() uint8 {
	return tx.inner.txType()
}

// ChainId returns the EIP155 chain ID of the transaction. The return value will always be
// non-nil. For legacy transactions which are not replay-protected, the return value is
// zero.
func (tx *Transaction) ChainId() *big.Int {
	return tx.inner.chainID()
}

// Data returns the input data of the transaction.
func (tx *Transaction) Data() []byte { return tx.inner.data() }

// AccessList returns the access list of the transaction.
func (tx *Transaction) AccessList() AccessList { return tx.inner.accessList() }

// Gas returns the gas limit of the transaction.
func (tx *Transaction) Gas() uint64 { return tx.inner.gas() }

// GasPrice returns the gas price of the transaction.
func (tx *Transaction) GasPrice() *big.Int { return copyBigInt(tx.inner.gasPrice()) }

// GasTipCap returns the gasTipCap per gas of the transaction.
func (tx *Transaction) GasTipCap() *big.Int { return copyBigInt(tx.inner.gasTipCap()) }

// GasFeeCap returns the fee cap per gas of the transaction.
func (tx *Transaction) GasFeeCap() *big.Int { return copyBigInt(tx.inner.gasFeeCap()) }

// Value returns the ether amount of the transaction.
func (tx *Transaction) Value() *big.Int { return copyBigInt(tx.inner.value()) }

// Nonce returns the sender account nonce of the transaction.
func (tx *Transaction) Nonce() uint64 { return tx.inner.nonce() }

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *Transaction) To() *Address { return tx.inner.to().copy() }

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.inner.rawSignatureValues()
}

func (tx *Transaction) Hash() Hash {
	// TODO(nickeskov): implement me
	panic("implement me")
}

func (tx *Transaction) DecodeRLP(rlpData []byte) error {
	parser := fastrlp.Parser{}
	rlpVal, err := parser.Parse(rlpData)
	if err != nil {
		return err
	}
	return tx.unmarshalFromFastRLP(rlpVal)
}

func (tx *Transaction) unmarshalFromFastRLP(value *fastrlp.Value) error {
	switch value.Type() {
	case fastrlp.TypeArray:
		// nickeskov: It's a legacy transaction.
		var inner LegacyTx
		err := inner.UnmarshalFromFastRLP(value)
		if err == nil {
			tx.setDecoded(&inner)
		}
		return err
	case fastrlp.TypeBytes:
		// nickeskov: It's an EIP-2718 typed TX envelope.
		typedTxBytes, err := value.Bytes()
		if err != nil {
			return err
		}
		inner, err := tx.decodeTyped(typedTxBytes)
		if err == nil {
			tx.setDecoded(inner)
		}
		return err
	default:
		return ErrTxTypeDecode
	}
}

func (tx Transaction) EncodeRLP(w io.Writer) error {
	arena := &fastrlp.Arena{}
	var fastrlpTx *fastrlp.Value
	// nickeskov: maybe use buffer pool?
	if tx.Type() == LegacyTxType {
		fastrlpTx = tx.inner.marshalToFastRLP(arena)
	} else {
		fastrlpTx = tx.encodeTyped(arena)
	}
	if _, err := w.Write(fastrlpTx.MarshalTo(nil)); err != nil {
		return err
	}
	return nil
}

func (tx *Transaction) setDecoded(inner TxData) {
	tx.inner = inner
}

// decodeTyped decodes a typed transaction from the canonical format.
func (tx *Transaction) decodeTyped(rlpData []byte) (TxData, error) {
	if len(rlpData) == 0 {
		return nil, errEmptyTypedTx
	}
	switch txType, rlpData := rlpData[0], rlpData[1:]; txType {
	case AccessListTxType:
		var inner AccessListTx
		if err := inner.DecodeRLP(rlpData); err != nil {
			return nil, err
		}
		return &inner, nil
	case DynamicFeeTxType:
		var inner DynamicFeeTx
		if err := inner.DecodeRLP(rlpData); err != nil {
			return nil, err
		}
		return &inner, nil
	default:
		return nil, ErrTxTypeNotSupported
	}
}

// encodeTyped writes the canonical encoding of a typed transaction to w.
func (tx *Transaction) encodeTyped(arena *fastrlp.Arena) *fastrlp.Value {
	rlpMarshaledTx := []byte{tx.Type()}
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
func (tx *Transaction) Protected() bool {
	switch tx := tx.inner.(type) {
	case *LegacyTx:
		return tx.V != nil && isProtectedV(tx.V)
	default:
		return true
	}
}

func (tx *Transaction) SignerHash(chainID *big.Int) Hash {
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(chainID, arena)

	var rlpData []byte

	switch tx.Type() {
	case LegacyTxType:
		rlpData = hashValues.MarshalTo(nil)
	case AccessListTxType, DynamicFeeTxType:
		rlpData = append(rlpData, tx.Type())
		rlpData = hashValues.MarshalTo(rlpData)
	default:
		// This _should_ not happen, but in case someone sends in a bad
		// json struct via RPC, it's probably more prudent to return an
		// empty hash instead of killing the node with a panic
		//panic("Unsupported transaction type: %d", tx.typ)
		return Hash{}
	}
	var h Hash
	sha := sha3.NewLegacyKeccak256().(KeccakState)
	// nickeskov: it always returns a nil error
	_, _ = sha.Write(rlpData)
	_, _ = sha.Read(h[:])
	return h
}
