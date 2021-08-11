package proto

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
	"math/big"
)

// Common big integers often used
var (
	Big1           = big.NewInt(1)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
)

// ValidateEthereumSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func ValidateEthereumSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if r.Cmp(Big1) < 0 || s.Cmp(Big1) < 0 {
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

const (
	// TODO(nickeskov): this is weird case. ethSig can't have size == 129, but in scala node it can...
	weirdEthereumSignatureLength = 129
)

// EthereumSignature represents ethereum signature (v, r, s signature values).
type EthereumSignature struct {
	sig []byte
}

func NewEthereumSignatureFromBytes(b []byte) (EthereumSignature, error) {
	sigLen := len(b)
	if sigLen != ethereumSignatureLength && sigLen != weirdEthereumSignatureLength {
		return EthereumSignature{},
			errors.Errorf("ethSignature should be of length %d or %d",
				ethereumSignatureLength, weirdEthereumSignatureLength)
	}
	sig := make([]byte, sigLen)
	copy(sig, b)
	return EthereumSignature{sig: sig}, nil
}

func (es EthereumSignature) Bytes() []byte {
	return es.sig
}

func (es EthereumSignature) MarshalBinary() ([]byte, error) {
	b := make([]byte, len(es.sig))
	copy(b, es.sig[:])
	return b, nil
}

func (es *EthereumSignature) UnmarshalBinary(data []byte) error {
	newEthSig, err := NewEthereumSignatureFromBytes(data)
	if err != nil {
		return errors.Wrap(err, "failed unmarshal binary EthereumSignature")
	}
	*es = newEthSig
	return nil
}

func (es EthereumSignature) MarshalJSON() ([]byte, error) {
	// TODO(nickeskov): Should it be hex or base58?
	return []byte(EncodeToHexString(es.sig)), nil
}

func (es *EthereumSignature) UnmarshalJSON(value []byte) error {
	sig, err := DecodeFromHexString(string(value))
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal JSON EthereumSignature")
	}
	*es = EthereumSignature{sig: sig}
	return nil
}

// AsVRS return ethereum signature as V, R, S signature values.
// Note that V can be 27/28 for legacy reasons, but real V value is 0/1.
func (es EthereumSignature) AsVRS() (v, r, s *big.Int) {
	switch len(es.sig) {
	case ethereumSignatureLength:
		r = new(big.Int).SetBytes(es.sig[:32])
		s = new(big.Int).SetBytes(es.sig[32:64])
		v = new(big.Int).SetBytes([]byte{es.sig[64]})
	case weirdEthereumSignatureLength:
		r = new(big.Int).SetBytes(es.sig[:64])
		s = new(big.Int).SetBytes(es.sig[64:128])
		v = new(big.Int).SetBytes([]byte{es.sig[128]})
	}
	return v, r, s
}
