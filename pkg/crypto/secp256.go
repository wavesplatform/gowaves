package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"
)

const (
	P256RawPubKeySize  = 64 // X(32) || Y(32)
	P1363SignatureSize = 64
)

// SecP256Verify verifies ECDSA signature on NIST P-256.
// Public key must be in raw (64-byte) format.
// Signature must be in 64-byte P1363 format (R(32) || S(32)).
func SecP256Verify(digest, publicKey, signature []byte) (bool, error) {
	if l := len(digest); l != DigestSize { // Validate digest size.
		return false, fmt.Errorf("unexpected digest size %d, expected 32-byte digest", l)
	}
	pk, err := parseECDSAPublicKey(publicKey)
	if err != nil {
		return false, fmt.Errorf("failed to verify P-256 signature: %w", err)
	}
	r, s, err := parseECDSASignature(signature)
	if err != nil {
		return false, fmt.Errorf("failed to parse ECDSA signature: %w", err)
	}
	return ecdsa.Verify(pk, digest, r, s), nil
}

func parseECDSAPublicKey(data []byte) (*ecdsa.PublicKey, error) {
	if len(data) != P256RawPubKeySize {
		return nil, errors.New("invalid public key size, expected 64-byte raw format (X||Y)")
	}
	x := new(big.Int).SetBytes(data[0:32])
	y := new(big.Int).SetBytes(data[32:64])

	curve := elliptic.P256()
	if x.Sign() == 0 && y.Sign() == 0 {
		return nil, errors.New("invalid public key, point at infinity / zero not allowed")
	}
	if !curve.IsOnCurve(x, y) {
		return nil, errors.New("invalid public key, point is not on P-256 curve")
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func parseECDSASignature(signature []byte) (*big.Int, *big.Int, error) {
	if len(signature) != P1363SignatureSize {
		return nil, nil, errors.New("invalid signature size, expected 64-byte P1363 signature (r||s)")
	}
	r := new(big.Int).SetBytes(signature[0:32])
	s := new(big.Int).SetBytes(signature[32:64])

	p256N := elliptic.P256().Params().N
	if r.Sign() == 0 || s.Sign() == 0 {
		return nil, nil, errors.New("invalid signature, r or s is zero")
	}
	if r.Cmp(p256N) >= 0 || s.Cmp(p256N) >= 0 {
		return nil, nil, errors.New("invalid signature, R or S is out of range")
	}
	return r, s, nil
}
