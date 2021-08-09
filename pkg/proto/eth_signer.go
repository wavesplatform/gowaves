package proto

import (
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

var ErrInvalidChainId = errors.New("invalid chain id for signer")

const ethereumSignatureLength = 64 + 1 // 64 bytes ECDSA signature + 1 byte recovery id

type EthereumSigner interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *EthereumTransaction) (EthereumAddress, error)

	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *EthereumTransaction, sig []byte) (R, S, V *big.Int, err error)
	ChainID() *big.Int

	// Hash returns 'signature hash', i.e. the transaction hash that is signed by the
	// private key. This hash does not uniquely identify the transaction.
	Hash(tx *EthereumTransaction) EthereumHash

	// Equal returns true if the given signer is the same as the receiver.
	Equal(EthereumSigner) bool
}

// londonSigner is a main signer after the London hardfork (hardfork date - 05.08.2021)
type londonSigner struct{ eip2930Signer }

// NewLondonEthereumSigner returns a signer that accepts
// - EIP-1559 dynamic fee transactions
// - EIP-2930 access list transactions,
// - EIP-155 replay protected transactions, and
// - legacy Homestead transactions.
func NewLondonEthereumSigner(chainId *big.Int) EthereumSigner {
	return londonSigner{newEIP2930Signer(chainId)}
}

func (s londonSigner) Sender(tx *EthereumTransaction) (EthereumAddress, error) {
	if tx.EthereumTxType() != DynamicFeeTxType {
		return s.eip2930Signer.Sender(tx)
	}
	V, R, S := tx.RawSignatureValues()
	// DynamicFee txs are defined to use 0 and 1 as their recovery
	// id, add 27 to become equivalent to unprotected Homestead signatures.
	V = new(big.Int).Add(V, big.NewInt(27))
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return EthereumAddress{}, ErrInvalidChainId
	}
	return recoverEthereumAddress(s.Hash(tx), R, S, V, true)
}

func (s londonSigner) Equal(s2 EthereumSigner) bool {
	x, ok := s2.(londonSigner)
	return ok && x.chainId.Cmp(s.chainId) == 0
}

func (s londonSigner) SignatureValues(tx *EthereumTransaction, sig []byte) (R, S, V *big.Int, err error) {
	txdata, ok := tx.inner.(*EthereumDynamicFeeTx)
	if !ok {
		return s.eip2930Signer.SignatureValues(tx, sig)
	}
	// Check that chain ID of tx matches the signer. We also accept ID zero here,
	// because it indicates that the chain ID was not specified in the tx.
	if txdata.ChainID.Sign() != 0 && txdata.ChainID.Cmp(s.chainId) != 0 {
		return nil, nil, nil, ErrInvalidChainId
	}
	R, S, _ = decodeSignature(sig)
	V = big.NewInt(int64(sig[64]))
	return R, S, V, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s londonSigner) Hash(tx *EthereumTransaction) EthereumHash {
	if tx.EthereumTxType() != DynamicFeeTxType {
		return s.eip2930Signer.Hash(tx)
	}
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(s.chainId, arena)

	rlpData := []byte{byte(tx.EthereumTxType())}
	rlpData = hashValues.MarshalTo(rlpData)

	return Keccak256EthereumHash(rlpData)
}

// BERLIN signer
type eip2930Signer struct{ eip155Signer }

// newEIP2930Signer returns a signer that accepts EIP-2930 access list transactions,
// EIP-155 replay protected transactions, and legacy Homestead transactions.
func newEIP2930Signer(chainId *big.Int) eip2930Signer {
	return eip2930Signer{newEIP155Signer(chainId)}
}

func (s eip2930Signer) Equal(s2 EthereumSigner) bool {
	x, ok := s2.(eip2930Signer)
	return ok && x.chainId.Cmp(s.chainId) == 0
}

func (s eip2930Signer) Sender(tx *EthereumTransaction) (EthereumAddress, error) {
	if tx.EthereumTxType() != AccessListTxType {
		return s.eip155Signer.Sender(tx)
	}
	V, R, S := tx.RawSignatureValues()
	// AL txs are defined to use 0 and 1 as their recovery
	// id, add 27 to become equivalent to unprotected Homestead signatures.
	V = new(big.Int).Add(V, big.NewInt(27))
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return EthereumAddress{}, ErrInvalidChainId
	}
	return recoverEthereumAddress(s.Hash(tx), R, S, V, true)
}

func (s eip2930Signer) SignatureValues(tx *EthereumTransaction, sig []byte) (R, S, V *big.Int, err error) {
	switch txdata := tx.inner.(type) {
	case *EthereumLegacyTx:
		return s.eip155Signer.SignatureValues(tx, sig)
	case *EthereumAccessListTx:
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
func (s eip2930Signer) Hash(tx *EthereumTransaction) EthereumHash {
	if tx.EthereumTxType() != AccessListTxType {
		return s.eip155Signer.Hash(tx)
	}
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(s.chainId, arena)

	rlpData := []byte{byte(tx.EthereumTxType())}
	rlpData = hashValues.MarshalTo(rlpData)

	return Keccak256EthereumHash(rlpData)
}

type eip155Signer struct {
	chainId, chainIdMul *big.Int
}

func newEIP155Signer(chainId *big.Int) eip155Signer {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return eip155Signer{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

func (s eip155Signer) ChainID() *big.Int {
	return s.chainId
}

func (s eip155Signer) Equal(s2 EthereumSigner) bool {
	eip155, ok := s2.(eip155Signer)
	return ok && eip155.chainId.Cmp(s.chainId) == 0
}

var big8 = big.NewInt(8)

func (s eip155Signer) Sender(tx *EthereumTransaction) (EthereumAddress, error) {
	if tx.EthereumTxType() != LegacyTxType {
		return EthereumAddress{}, ErrTxTypeNotSupported
	}
	if !tx.Protected() {
		return HomesteadSigner{}.Sender(tx)
	}
	// TODO
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return EthereumAddress{}, ErrInvalidChainId
	}
	V, R, S := tx.RawSignatureValues()
	V = new(big.Int).Sub(V, s.chainIdMul)
	V.Sub(V, big8)
	return recoverEthereumAddress(s.Hash(tx), R, S, V, true)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (s eip155Signer) SignatureValues(tx *EthereumTransaction, sig []byte) (R, S, V *big.Int, err error) {
	if tx.EthereumTxType() != LegacyTxType {
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
func (s eip155Signer) Hash(tx *EthereumTransaction) EthereumHash {
	if tx.EthereumTxType() != LegacyTxType {
		// This _should_ not happen, but in case someone sends in a bad
		// json struct via RPC, it's probably more prudent to return an
		// empty hash instead of killing the node with a panic
		//panic("Unsupported transaction type: %d", tx.typ)
		return EthereumHash{}
	}
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(s.chainId, arena)

	rlpData := hashValues.MarshalTo(nil)

	return Keccak256EthereumHash(rlpData)
}

// HomesteadTransaction implements TransactionInterface using the
// homestead rules.
type HomesteadSigner struct{ FrontierSigner }

func (hs HomesteadSigner) ChainID() *big.Int {
	return nil
}

func (hs HomesteadSigner) Equal(s2 EthereumSigner) bool {
	_, ok := s2.(HomesteadSigner)
	return ok
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (hs HomesteadSigner) SignatureValues(tx *EthereumTransaction, sig []byte) (r, s, v *big.Int, err error) {
	return hs.FrontierSigner.SignatureValues(tx, sig)
}

func (hs HomesteadSigner) Sender(tx *EthereumTransaction) (EthereumAddress, error) {
	if tx.EthereumTxType() != LegacyTxType {
		return EthereumAddress{}, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverEthereumAddress(hs.Hash(tx), r, s, v, true)
}

type FrontierSigner struct{}

func (fs FrontierSigner) ChainID() *big.Int {
	return nil
}

func (fs FrontierSigner) Equal(s2 EthereumSigner) bool {
	_, ok := s2.(FrontierSigner)
	return ok
}

func (fs FrontierSigner) Sender(tx *EthereumTransaction) (EthereumAddress, error) {
	if tx.EthereumTxType() != LegacyTxType {
		return EthereumAddress{}, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverEthereumAddress(fs.Hash(tx), r, s, v, true)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (fs FrontierSigner) SignatureValues(tx *EthereumTransaction, sig []byte) (r, s, v *big.Int, err error) {
	if tx.EthereumTxType() != LegacyTxType {
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	r, s, v = decodeSignature(sig)
	return r, s, v, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (fs FrontierSigner) Hash(tx *EthereumTransaction) EthereumHash {
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(fs.ChainID(), arena)

	var rlpData []byte

	switch tx.EthereumTxType() {
	case LegacyTxType:
		rlpData = hashValues.MarshalTo(nil)
	case AccessListTxType, DynamicFeeTxType:
		rlpData = append(rlpData, byte(tx.EthereumTxType()))
		rlpData = hashValues.MarshalTo(rlpData)
	default:
		// This _should_ not happen, but in case someone sends in a bad
		// json struct via RPC, it's probably more prudent to return an
		// empty hash instead of killing the node with a panic
		//panic("Unsupported transaction type: %d", tx.typ)
		return EthereumHash{}
	}
	return Keccak256EthereumHash(rlpData)
}

func decodeSignature(sig []byte) (r, s, v *big.Int) {
	if len(sig) != ethereumSignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), ethereumSignatureLength))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}

func recoverEthereumAddress(sighash EthereumHash, R, S, Vb *big.Int, homestead bool) (EthereumAddress, error) {
	// recover the public key from the signature
	pubKey, err := recoverEthereumPubKey(sighash, R, S, Vb, homestead)
	if err != nil {
		return EthereumAddress{}, err
	}
	return ecdsaPublicKeyToAddress(pubKey), nil
}

func recoverEthereumPubKey(sighash EthereumHash, R, S, Vb *big.Int, homestead bool) (*btcec.PublicKey, error) {
	if Vb.BitLen() > 8 {
		return nil, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !ValidateSignatureValues(V, R, S, homestead) {
		return nil, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, ethereumSignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pubKey, err := crypto.ECDSARecoverPublicKey(sighash[:], sig)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func ecdsaPublicKeyToAddress(p *btcec.PublicKey) EthereumAddress {
	pubBytes := p.SerializeUncompressed()
	// nickeskov: can't fail
	hash, _ := crypto.Keccak256(pubBytes[1:])
	var addr EthereumAddress
	addr.setBytes(hash[12:])
	return addr
}
