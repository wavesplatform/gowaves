package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
)

const (
	secP256r1UncompressedPubKeyPrefix = 0x04
	secP256r1RawPubKeySize            = 64                         // X(32) || Y(32)
	secP256r1UncompressedPubKeySize   = 1 + secP256r1RawPubKeySize // SEC1 0x04 || X(32) || Y(32)
	sec2562r1P1363SignatureSize       = 64
)

// SecP256Verify verifies ECDSA signature on NIST P-256 (aka secp256r1). X, Y, R, S are big-endian byte slices. Inputs:
//
// publicKey formats supported:
//   - 65 bytes: uncompressed SEC1 form 0x04 || X(32) || Y(32)
//   - 64 bytes: raw X(32) || Y(32)
//
// signature format supported:
//   - 64 bytes P1363: R(32) || S(32)
func SecP256Verify(digest, publicKey, signature []byte) (bool, error) {
	curve := elliptic.P256()

	// ---- Parse public key ----
	var x, y *big.Int
	switch len(publicKey) {
	case secP256r1UncompressedPubKeySize:
		if publicKey[0] != secP256r1UncompressedPubKeyPrefix {
			return false, errors.New("publicKey: expected uncompressed SEC1 prefix 0x04")
		}
		x = new(big.Int).SetBytes(publicKey[1:33])
		y = new(big.Int).SetBytes(publicKey[33:65])
	case secP256r1RawPubKeySize:
		x = new(big.Int).SetBytes(publicKey[0:32])
		y = new(big.Int).SetBytes(publicKey[32:64])
	default:
		return false, errors.New("publicKey: expected 64 or 65 bytes")
	}

	// Validate point is on curve (prevents invalid-curve / nonsense keys).
	// TODO: does we need these validations? all tests pass without them.
	if x.Sign() == 0 && y.Sign() == 0 {
		return false, errors.New("publicKey: point at infinity / zero not allowed")
	}
	if !curve.IsOnCurve(x, y) {
		return false, errors.New("publicKey: point is not on P-256 curve")
	}

	pub := ecdsa.PublicKey{Curve: curve, X: x, Y: y}

	r, s, err := parseECDSASignature(signature)
	if err != nil {
		return false, fmt.Errorf("failed to parse ECDSA signature: %w", err)
	}

	// Validate digest size.
	if len(digest) != DigestSize {
		return false, errors.New("digest: expected 32-byte digest")
	}

	// Verify signature.
	ok := ecdsa.Verify(&pub, digest, r, s)
	return ok, nil
}

// VerifyECDSASignature verifies ECDSA signature using the public key from the provided X.509 certificate.
// The certificate must contain an ECDSA public key on the NIST P-256 curve.
// The signature must be in P1363 format (R(32) || S(32)).
// The digest must be a 32-byte SHA-256 hash of the signed data.
func VerifyECDSASignature(
	cert *x509.Certificate, digest, signature []byte,
) (bool, error) {
	if cert == nil {
		return false, errors.New("no certificate provided")
	}
	pub := cert.PublicKey
	switch pk := pub.(type) {
	case *ecdsa.PublicKey:
		if len(digest) != DigestSize {
			return false, errors.New("invalid digest size, expected 32-byte digest")
		}
		r, s, err := parseECDSASignature(signature)
		if err != nil {
			return false, fmt.Errorf("failed to parse ECDSA signature: %w", err)
		}
		return ecdsa.Verify(pk, digest, r, s), nil
	default:
		return false, fmt.Errorf("unexpected public key type %T", pub)
	}
}

func parseECDSASignature(signature []byte) (*big.Int, *big.Int, error) {
	if len(signature) != sec2562r1P1363SignatureSize {
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
