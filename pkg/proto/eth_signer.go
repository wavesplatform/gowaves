package proto

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

var ErrInvalidChainId = errors.New("invalid chain id for signer")

const (
	EthereumPublicKeyLength = 64
	ethereumSignatureLength = 64 + 1 // 64 bytes ECDSA signature + 1 byte recovery id

	ethereumPublicKeyUncompressedPrefix byte = 0x4         // prefix which means this is an uncompressed point
	ethereumPublicKeyBytesUncompressed       = 1 + 32 + 32 // 0x4 prefix + x_coordinate bytes + y_coordinate bytes
	ethereumPublicKeyBytesCompressed         = 1 + 32      // y_bit (0x02 if y is even, 0x03 if y is odd) + x_coordinate bytes
)

// EthereumPrivateKey is an Ethereum ecdsa.PrivateKey.
type EthereumPrivateKey btcec.PrivateKey

// EthereumPublicKey returns *EthereumPublicKey from corresponding EthereumPrivateKey.
func (esk *EthereumPrivateKey) EthereumPublicKey() *EthereumPublicKey {
	return (*EthereumPublicKey)((*btcec.PrivateKey)(esk).PubKey())
}

// EthereumPublicKey is an Ethereum ecdsa.PublicKey.
type EthereumPublicKey btcec.PublicKey

// MarshalJSON marshal EthereumPublicKey in hex encoding.
func (epk *EthereumPublicKey) MarshalJSON() ([]byte, error) {
	data := epk.SerializeXYCoordinates()
	return HexBytes(data).MarshalJSON()
}

// UnmarshalJSON unmarshal EthereumPublicKey from hex encoding.
func (epk *EthereumPublicKey) UnmarshalJSON(bytes []byte) error {
	pkBytes := HexBytes{}
	err := pkBytes.UnmarshalJSON(bytes)
	if err != nil {
		return err
	}
	return epk.UnmarshalBinary(pkBytes)
}

func NewEthereumPublicKeyFromHexString(s string) (EthereumPublicKey, error) {
	b, err := DecodeFromHexString(s)
	if err != nil {
		return EthereumPublicKey{}, errors.Wrapf(err,
			"failed to decode marshaled EthereumPublicKey into bytes from hex string %q", s,
		)
	}
	return NewEthereumPublicKeyFromBytes(b)
}

// NewEthereumPublicKeyFromBase58String creates an EthereumPublicKey from its string representation.
func NewEthereumPublicKeyFromBase58String(s string) (EthereumPublicKey, error) {
	b, err := base58.Decode(s)
	if err != nil {
		return EthereumPublicKey{}, errors.Wrap(err, "invalid Base58 string")
	}
	return NewEthereumPublicKeyFromBytes(b)
}

func NewEthereumPublicKeyFromBytes(b []byte) (EthereumPublicKey, error) {
	var pubKey EthereumPublicKey
	if err := pubKey.UnmarshalBinary(b); err != nil {
		return EthereumPublicKey{}, err
	}
	return pubKey, nil
}

func (epk *EthereumPublicKey) String() string {
	data := epk.SerializeXYCoordinates()
	return EncodeToHexString(data)
}

func (epk *EthereumPublicKey) MarshalBinary() (data []byte, err error) {
	// nickeskov: right way is to use SerializeUncompressed
	// 	but for scala compatibility we use a 64 byte representation (scala node uses web3j library)
	return epk.SerializeXYCoordinates(), nil
}

func (epk *EthereumPublicKey) UnmarshalBinary(data []byte) error {
	if len(data) == ethereumPublicKeyBytesUncompressed-1 {
		// nickeskov: special case for web3j (scala node)
		//	in this library public key len == 64 bytes (uncompressed key without prefix)
		uncompressed := make([]byte, ethereumPublicKeyBytesUncompressed)
		uncompressed[0] = ethereumPublicKeyUncompressedPrefix
		copy(uncompressed[1:], data)
		data = uncompressed
	}
	pubKeyLen := len(data)
	if pubKeyLen != ethereumPublicKeyBytesUncompressed && pubKeyLen != ethereumPublicKeyBytesCompressed {
		return errors.Errorf(
			"wrong size for marshaled ethereum public key: got %d, want %d (uncompressed without prefix) or %d (uncompressed) or %d (compressed)",
			pubKeyLen,
			ethereumPublicKeyBytesUncompressed-1,
			ethereumPublicKeyBytesUncompressed,
			ethereumPublicKeyBytesCompressed,
		)
	}
	pubKey, err := crypto.ECDSAParsePublicKey(data)
	if err != nil {
		return errors.Wrapf(err, "failed to parse EthereumPublicKey from bytes %q", EncodeToHexString(data))
	}
	*epk = EthereumPublicKey(*pubKey)
	return nil
}

// ToECDSA returns the public key as a *ecdsa.PublicKey.
func (epk *EthereumPublicKey) ToECDSA() *ecdsa.PublicKey {
	return (*btcec.PublicKey)(epk).ToECDSA()
}

// SerializeUncompressed serializes a public key in a 65-byte uncompressed format.
func (epk *EthereumPublicKey) SerializeUncompressed() []byte {
	return (*btcec.PublicKey)(epk).SerializeUncompressed()
}

// SerializeCompressed serializes a public key in a 33-byte compressed format.
func (epk *EthereumPublicKey) SerializeCompressed() []byte {
	return (*btcec.PublicKey)(epk).SerializeCompressed()
}

// SerializeXYCoordinates serializes a public key in a 64-byte uncompressed format without 0x4 byte prefix.
func (epk *EthereumPublicKey) SerializeXYCoordinates() []byte {
	return epk.SerializeUncompressed()[1:]
}

func (epk *EthereumPublicKey) EthereumAddress() EthereumAddress {
	xy := epk.SerializeXYCoordinates()
	hash := crypto.MustKeccak256(xy)
	var addr EthereumAddress
	addr.setBytes(hash[12:])
	return addr
}

func (epk *EthereumPublicKey) copy() *EthereumPublicKey {
	cpy := btcec.PublicKey(*epk)
	return (*EthereumPublicKey)(&cpy)
}

type EthereumSigner interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *EthereumTransaction) (EthereumAddress, error)

	// SenderPK returns the sender public key of the transaction.
	SenderPK(tx *EthereumTransaction) (*EthereumPublicKey, error)

	// SignatureValues returns the raw R, S, V values corresponding to the given signature.
	SignatureValues(tx *EthereumTransaction, sig []byte) (r, s, v *big.Int, err error)

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

func (ls londonSigner) SenderPK(tx *EthereumTransaction) (*EthereumPublicKey, error) {
	if tx.EthereumTxType() != EthereumDynamicFeeTxType {
		return ls.eip2930Signer.SenderPK(tx)
	}
	v, r, s := tx.RawSignatureValues()
	// DynamicFee txs are defined to use 0 and 1 as their recovery
	// id, add 27 to become equivalent to unprotected Homestead signatures.
	v = new(big.Int).Add(v, big.NewInt(27))
	if tx.ChainId().Cmp(ls.chainId) != 0 {
		return nil, ErrInvalidChainId
	}
	return recoverEthereumPubKey(ls.Hash(tx), r, s, v)
}

func (ls londonSigner) Sender(tx *EthereumTransaction) (EthereumAddress, error) {
	pk, err := ls.SenderPK(tx)
	if err != nil {
		return EthereumAddress{}, err
	}
	return pk.EthereumAddress(), nil
}

func (ls londonSigner) Equal(s2 EthereumSigner) bool {
	x, ok := s2.(londonSigner)
	return ok && x.chainId.Cmp(ls.chainId) == 0
}

func (ls londonSigner) SignatureValues(tx *EthereumTransaction, sig []byte) (r, s, v *big.Int, err error) {
	txdata, ok := tx.inner.(*EthereumDynamicFeeTx)
	if !ok {
		return ls.eip2930Signer.SignatureValues(tx, sig)
	}
	// Check that chain ID of tx matches the signer. We also accept ID zero here,
	// because it indicates that the chain ID was not specified in the tx.
	if txdata.ChainID.Sign() != 0 && txdata.ChainID.Cmp(ls.chainId) != 0 {
		return nil, nil, nil, ErrInvalidChainId
	}
	r, s, _, err = decodeSignature(sig, true)
	if err != nil {
		return nil, nil, nil, err
	}
	v = big.NewInt(int64(sig[64]))
	return r, s, v, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (ls londonSigner) Hash(tx *EthereumTransaction) EthereumHash {
	if tx.EthereumTxType() != EthereumDynamicFeeTxType {
		return ls.eip2930Signer.Hash(tx)
	}
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(ls.chainId, arena)

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

func (es eip2930Signer) Equal(s2 EthereumSigner) bool {
	x, ok := s2.(eip2930Signer)
	return ok && x.chainId.Cmp(es.chainId) == 0
}

func (es eip2930Signer) Sender(tx *EthereumTransaction) (EthereumAddress, error) {
	pk, err := es.SenderPK(tx)
	if err != nil {
		return EthereumAddress{}, err
	}
	return pk.EthereumAddress(), nil
}

func (es eip2930Signer) SenderPK(tx *EthereumTransaction) (*EthereumPublicKey, error) {
	if tx.EthereumTxType() != EthereumAccessListTxType {
		return es.eip155Signer.SenderPK(tx)
	}
	v, r, s := tx.RawSignatureValues()
	// AL txs are defined to use 0 and 1 as their recovery
	// id, add 27 to become equivalent to unprotected Homestead signatures.
	v = new(big.Int).Add(v, big.NewInt(27))
	if tx.ChainId().Cmp(es.chainId) != 0 {
		return nil, ErrInvalidChainId
	}
	return recoverEthereumPubKey(es.Hash(tx), r, s, v)
}

func (es eip2930Signer) SignatureValues(tx *EthereumTransaction, sig []byte) (r, s, v *big.Int, err error) {
	switch txdata := tx.inner.(type) {
	case *EthereumLegacyTx:
		return es.eip155Signer.SignatureValues(tx, sig)
	case *EthereumAccessListTx:
		// Check that chain ID of tx matches the signer. We also accept ID zero here,
		// because it indicates that the chain ID was not specified in the tx.
		if txdata.ChainID.Sign() != 0 && txdata.ChainID.Cmp(es.chainId) != 0 {
			return nil, nil, nil, ErrInvalidChainId
		}
		r, s, _, err = decodeSignature(sig, true)
		if err != nil {
			return nil, nil, nil, err
		}
		v = big.NewInt(int64(sig[64]))
	default:
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	return r, s, v, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (es eip2930Signer) Hash(tx *EthereumTransaction) EthereumHash {
	if tx.EthereumTxType() != EthereumAccessListTxType {
		return es.eip155Signer.Hash(tx)
	}
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(es.chainId, arena)

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

func (es eip155Signer) ChainID() *big.Int {
	return es.chainId
}

func (es eip155Signer) Equal(s2 EthereumSigner) bool {
	eip155, ok := s2.(eip155Signer)
	return ok && eip155.chainId.Cmp(es.chainId) == 0
}

func (es eip155Signer) Sender(tx *EthereumTransaction) (EthereumAddress, error) {
	pk, err := es.SenderPK(tx)
	if err != nil {
		return EthereumAddress{}, err
	}
	return pk.EthereumAddress(), nil
}

func (es eip155Signer) SenderPK(tx *EthereumTransaction) (*EthereumPublicKey, error) {
	if tx.EthereumTxType() != EthereumLegacyTxType {
		return nil, ErrTxTypeNotSupported
	}
	if !tx.Protected() {
		return HomesteadSigner{}.SenderPK(tx)
	}
	if tx.ChainId().Cmp(es.chainId) != 0 {
		return nil, ErrInvalidChainId
	}
	v, r, s := tx.RawSignatureValues()
	v = new(big.Int).Sub(v, es.chainIdMul)
	v.Sub(v, big.NewInt(8))
	return recoverEthereumPubKey(es.Hash(tx), r, s, v)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (es eip155Signer) SignatureValues(tx *EthereumTransaction, sig []byte) (r, s, v *big.Int, err error) {
	if tx.EthereumTxType() != EthereumLegacyTxType {
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	r, s, v, err = decodeSignature(sig, true)
	if err != nil {
		return nil, nil, nil, err
	}
	if es.chainId.Sign() != 0 {
		v = big.NewInt(int64(sig[64] + 35))
		v.Add(v, es.chainIdMul)
	}
	return r, s, v, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (es eip155Signer) Hash(tx *EthereumTransaction) EthereumHash {
	if tx.EthereumTxType() != EthereumLegacyTxType {
		// This _should_ not happen, but in case someone sends in a bad
		// json struct via RPC, it's probably more prudent to return an
		// empty hash instead of killing the node with a panic
		//panic("Unsupported transaction type: %d", tx.typ)
		return EthereumHash{}
	}
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(es.chainId, arena)

	rlpData := hashValues.MarshalTo(nil)

	return Keccak256EthereumHash(rlpData)
}

// HomesteadSigner implements EthereumSigner using the homestead rules.
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
	pk, err := hs.SenderPK(tx)
	if err != nil {
		return EthereumAddress{}, err
	}
	return pk.EthereumAddress(), nil
}

func (hs HomesteadSigner) SenderPK(tx *EthereumTransaction) (*EthereumPublicKey, error) {
	if tx.EthereumTxType() != EthereumLegacyTxType {
		return nil, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverEthereumPubKey(hs.Hash(tx), r, s, v)
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
	pk, err := fs.SenderPK(tx)
	if err != nil {
		return EthereumAddress{}, err
	}
	return pk.EthereumAddress(), nil
}

func (fs FrontierSigner) SenderPK(tx *EthereumTransaction) (*EthereumPublicKey, error) {
	if tx.EthereumTxType() != EthereumLegacyTxType {
		return nil, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverEthereumPubKey(fs.Hash(tx), r, s, v)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (fs FrontierSigner) SignatureValues(tx *EthereumTransaction, sig []byte) (r, s, v *big.Int, err error) {
	if tx.EthereumTxType() != EthereumLegacyTxType {
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	r, s, v, err = decodeSignature(sig, true)
	if err != nil {
		return nil, nil, nil, err
	}
	return r, s, v, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (fs FrontierSigner) Hash(tx *EthereumTransaction) EthereumHash {
	arena := &fastrlp.Arena{}
	hashValues := tx.inner.signerHashFastRLP(fs.ChainID(), arena)

	var rlpData []byte

	switch tx.EthereumTxType() {
	case EthereumLegacyTxType:
		rlpData = hashValues.MarshalTo(nil)
	case EthereumAccessListTxType, EthereumDynamicFeeTxType:
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

// decodeSignature decodes r, s, v signature values from bytes.
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons, if legacyV==true.
func decodeSignature(sig []byte, legacyV bool) (r, s, v *big.Int, err error) {
	if len(sig) != ethereumSignatureLength {
		return nil, nil, nil,
			errors.Errorf("wrong size for signature: got %d, want %d", len(sig), ethereumSignatureLength)
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	vByte := sig[64]
	if legacyV {
		vByte += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	v = new(big.Int).SetBytes([]byte{vByte})
	return r, s, v, nil
}

func recoverEthereumPubKey(sighash EthereumHash, r, s, v *big.Int) (*EthereumPublicKey, error) {
	if v.BitLen() > 8 {
		return nil, ErrInvalidSig
	}
	legacyV := v.Uint64()
	if legacyV < 27 {
		return nil, ErrInvalidSig
	}
	vByte := byte(legacyV - 27)
	sig, err := NewEthereumSignatureFromVRS(vByte, r, s)
	if err != nil {
		return nil, err
	}
	return sig.RecoverEthereumPublicKey(sighash[:])
}
