package proto

import (
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big2)
)

// ValidateEthereumSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func ValidateEthereumSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if r.Cmp(big1) < 0 || s.Cmp(big1) < 0 {
		return false
	}
	// reject upper range of s values (ECDSA malleability)
	// see discussion in secp256k1/libsecp256k1/include/secp256k1.h
	if homestead && s.Cmp(secp256k1halfN) > 0 {
		return false
	}
	// Frontier: allow s to be in full N range
	return r.Cmp(secp256k1N) < 0 && s.Cmp(secp256k1N) < 0 && (v == 0 || v == 1)
}

// VerifyEthereumSignature checks that the given public key created signature over hash.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
func VerifyEthereumSignature(pubKey *EthereumPublicKey, r, s *big.Int, hash []byte) bool {
	sig := btcec.Signature{R: r, S: s}
	// Reject malleable signatures. libsecp256k1 does this check but btcec doesn't.
	if sig.S.Cmp(secp256k1halfN) > 0 {
		return false
	}
	return sig.Verify(hash, (*btcec.PublicKey)(pubKey))
}

// EthereumSignature represents ethereum signature (v, r, s signature values).
type EthereumSignature struct {
	sig [ethereumSignatureLength]byte
}

func NewEthereumSignatureFromHexString(hexString string) (ethSig EthereumSignature, err error) {
	b, err := DecodeFromHexString(hexString)
	if err != nil {
		return ethSig, errors.Wrap(err, "failed parse hex string to bytes to create EthereumSignature")
	}
	return NewEthereumSignatureFromBytes(b)
}

func NewEthereumSignatureFromBytes(b []byte) (ethSig EthereumSignature, err error) {
	err = ethSig.UnmarshalBinary(b)
	if err != nil {
		return EthereumSignature{}, err
	}
	return ethSig, nil
}

func (es *EthereumSignature) Bytes() []byte {
	return es.sig[:]
}

func (es *EthereumSignature) String() string {
	return EncodeToHexString(es.Bytes())
}

// AsVRS return ethereum signature as V, R, S signature values.
// Note that V can be 27/28 for legacy reasons, but real V value is 0/1.
func (es *EthereumSignature) AsVRS() (v, r, s *big.Int) {
	r = new(big.Int).SetBytes(es.sig[:32])
	s = new(big.Int).SetBytes(es.sig[32:64])
	v = new(big.Int).SetBytes([]byte{es.sig[64]})
	return v, r, s
}

func (es *EthereumSignature) MarshalBinary() (data []byte, err error) {
	return es.Bytes(), nil
}

func (es *EthereumSignature) UnmarshalBinary(data []byte) error {
	sigLen := len(data)
	if sigLen != ethereumSignatureLength {
		return errors.Errorf("eip712Signature should be of length %d", ethereumSignatureLength)
	}
	copy(es.sig[:], data)
	return nil
}

func (es EthereumSignature) MarshalJSON() ([]byte, error) {
	// nickeskov: can't fail
	data, _ := es.MarshalBinary()
	return HexBytes(data).MarshalJSON()
}

func (es *EthereumSignature) UnmarshalJSON(bytes []byte) error {
	sigBytes := HexBytes{}
	err := sigBytes.UnmarshalJSON(bytes)
	if err != nil {
		return err
	}
	return es.UnmarshalBinary(sigBytes)
}

func (es *EthereumSignature) RecoverEthereumPublicKey(digest []byte) (*EthereumPublicKey, error) {
	pk, err := crypto.ECDSARecoverPublicKey(digest, es.Bytes())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to recover public from signature %s with digest %q",
			es.String(), EncodeToHexString(digest),
		)
	}
	return (*EthereumPublicKey)(pk), nil
}
