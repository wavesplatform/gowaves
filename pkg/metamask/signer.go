package metamask

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"math/big"
)

const SignatureLength = 64 + 1 // 64 bytes ECDSA signature + 1 byte recovery id
var ErrInvalidChainId = errors.New("invalid chain id for signer")

type Signer interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *Transaction) (Address, error)

	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
	ChainID() *big.Int

	// Hash returns 'signature hash', i.e. the transaction hash that is signed by the
	// private key. This hash does not uniquely identify the transaction.
	Hash(tx *Transaction) Hash

	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}

// BERLIN signer
type eip2930Signer struct{ EIP155Signer }

// NewEIP2930Signer returns a signer that accepts EIP-2930 access list transactions,
// EIP-155 replay protected transactions, and legacy Homestead transactions.
func NewEIP2930Signer(chainId *big.Int) Signer {
	return eip2930Signer{NewEIP155Signer(chainId)}
}

func (s eip2930Signer) ChainID() *big.Int {
	return s.chainId
}

func (s eip2930Signer) Equal(s2 Signer) bool {
	x, ok := s2.(eip2930Signer)
	return ok && x.chainId.Cmp(s.chainId) == 0
}

func (s eip2930Signer) Sender(tx *Transaction) (Address, error) {
	V, R, S := tx.RawSignatureValues()
	switch tx.Type() {
	case LegacyTxType:
		if !tx.Protected() {
			return HomesteadSigner{}.Sender(tx)
		}
		V = new(big.Int).Sub(V, s.chainIdMul)
		V.Sub(V, big8)
	case AccessListTxType:
		// AL txs are defined to use 0 and 1 as their recovery
		// id, add 27 to become equivalent to unprotected Homestead signatures.
		V = new(big.Int).Add(V, big.NewInt(27))
	default:
		return Address{}, ErrTxTypeNotSupported
	}
	//if tx.ChainId().Cmp(s.chainId) != 0 {
	//	return Address{}, ErrInvalidChainId
	//}
	return recoverPlain(tx.SignerHash(s.chainId), R, S, V, true)
}

func (s eip2930Signer) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	switch txdata := tx.inner.(type) {
	case *LegacyTx:
		return s.EIP155Signer.SignatureValues(tx, sig)
	case *AccessListTx:
		// Check that chain ID of tx matches the signer. We also accept ID zero here,
		// because it indicates that the chain ID was not specified in the tx.
		if txdata.ChainID.Sign() != 0 && txdata.ChainID.Cmp(s.chainId) != 0 {
			return nil, nil, nil, ErrInvalidChainId
		}
		R, S, _ = decodeSignature(sig)
		V = big.NewInt(int64(sig[64]))
	default:
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	return R, S, V, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s eip2930Signer) Hash(tx *Transaction) Hash {
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(s.chainId, arena)

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
	return Keccak256Hash(rlpData)
}

type EIP155Signer struct {
	chainId, chainIdMul *big.Int
}

func NewEIP155Signer(chainId *big.Int) EIP155Signer {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return EIP155Signer{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

func (s EIP155Signer) ChainID() *big.Int {
	return s.chainId
}

func (s EIP155Signer) Equal(s2 Signer) bool {
	eip155, ok := s2.(EIP155Signer)
	return ok && eip155.chainId.Cmp(s.chainId) == 0
}

var big8 = big.NewInt(8)

func (s EIP155Signer) Sender(tx *Transaction) (Address, error) {
	if tx.Type() != LegacyTxType {
		return Address{}, ErrTxTypeNotSupported
	}
	if !tx.Protected() {
		return HomesteadSigner{}.Sender(tx)
	}
	// TODO
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return Address{}, ErrInvalidChainId
	}
	V, R, S := tx.RawSignatureValues()
	V = new(big.Int).Sub(V, s.chainIdMul)
	V.Sub(V, big8)
	return recoverPlain(tx.SignerHash(s.chainId), R, S, V, true)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (s EIP155Signer) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	if tx.Type() != LegacyTxType {
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	R, S, V = decodeSignature(sig)
	if s.chainId.Sign() != 0 {
		V = big.NewInt(int64(sig[64] + 35))
		V.Add(V, s.chainIdMul)
	}
	return R, S, V, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s EIP155Signer) Hash(tx *Transaction) Hash {
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(s.chainId, arena)

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
	return Keccak256Hash(rlpData)
}

// HomesteadTransaction implements TransactionInterface using the
// homestead rules.
type HomesteadSigner struct{ FrontierSigner }

func (s HomesteadSigner) ChainID() *big.Int {
	return nil
}

func (s HomesteadSigner) Equal(s2 Signer) bool {
	_, ok := s2.(HomesteadSigner)
	return ok
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (hs HomesteadSigner) SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	return hs.FrontierSigner.SignatureValues(tx, sig)
}

func (hs HomesteadSigner) Sender(tx *Transaction) (Address, error) {
	if tx.Type() != LegacyTxType {
		return Address{}, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverPlain(tx.SignerHash(hs.ChainID()), r, s, v, true)
}

type FrontierSigner struct{}

func (s FrontierSigner) ChainID() *big.Int {
	return nil
}

func (s FrontierSigner) Equal(s2 Signer) bool {
	_, ok := s2.(FrontierSigner)
	return ok
}

func (fs FrontierSigner) Sender(tx *Transaction) (Address, error) {
	if tx.Type() != LegacyTxType {
		return Address{}, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverPlain(tx.SignerHash(fs.ChainID()), r, s, v, true)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (fs FrontierSigner) SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	if tx.Type() != LegacyTxType {
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	r, s, v = decodeSignature(sig)
	return r, s, v, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (fs FrontierSigner) Hash(tx *Transaction) Hash {
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(fs.ChainID(), arena)

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
	return Keccak256Hash(rlpData)
}

func decodeSignature(sig []byte) (r, s, v *big.Int) {
	if len(sig) != SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), SignatureLength))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}

func recoverPlain(sighash Hash, R, S, Vb *big.Int, homestead bool) (Address, error) {
	if Vb.BitLen() > 8 {
		return Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !ValidateSignatureValues(V, R, S, homestead) {
		return Address{}, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pubKey, err := crypto.ECDSARecoverPublicKey(sighash[:], sig)
	if err != nil {
		return Address{}, err
	}

	var addrKey Address
	res := pubKey.SerializeUncompressed()[1:]
	l, err := crypto.Keccak256(res)
	if err != nil {
		return Address{}, err
	}
	copy(addrKey[:], l[12:])
	return addrKey, nil
}
