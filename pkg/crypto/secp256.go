package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"
)

const (
	secp256r1UncompressedPubKeyPrefix = 0x04
	secp256r1RawPubKeySize            = 64                         // X(32) || Y(32)
	secp256r1UncompressedPubKeySize   = 1 + secp256r1RawPubKeySize // SEC1 0x04 || X(32) || Y(32)
	sec2562r1P1363SignatureSize       = 64
)

// Secp256Verify verifies ECDSA signature on NIST P-256 (aka secp256r1). X, Y, R, S are big-endian byte slices. Inputs:
//
// publicKey formats supported:
//   - 65 bytes: uncompressed SEC1 form 0x04 || X(32) || Y(32)
//   - 64 bytes: raw X(32) || Y(32)
//
// signature format supported:
//   - 64 bytes P1363: R(32) || S(32)
//
// TODO:
//   - what kind of signature formats to support? ASN.1 DER or P1363 (r||s)? should we support both?
//   - should we enforce low-S signatures to prevent malleability?
func Secp256Verify(digest, publicKey, signature []byte) (bool, error) {
	curve := elliptic.P256()

	// ---- Parse public key ----
	var x, y *big.Int
	switch len(publicKey) {
	case secp256r1UncompressedPubKeySize:
		if publicKey[0] != secp256r1UncompressedPubKeyPrefix {
			return false, errors.New("publicKey: expected uncompressed SEC1 prefix 0x04")
		}
		x = new(big.Int).SetBytes(publicKey[1:33])
		y = new(big.Int).SetBytes(publicKey[33:65])
	case secp256r1RawPubKeySize:
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

	// ---- Parse signature (P1363 r||s) ----
	if len(signature) != sec2562r1P1363SignatureSize {
		return false, errors.New("signature: expected 64-byte P1363 signature (r||s)")
	}
	r := new(big.Int).SetBytes(signature[0:32])
	s := new(big.Int).SetBytes(signature[32:64])

	// ---- Validate r,s range: 1 <= r,s <= N-1 ----
	// TODO: does we need these validations? all tests pass without them.
	basePoitOrderN := curve.Params().N
	if r.Sign() <= 0 || s.Sign() <= 0 {
		return false, errors.New("signature: r or s is zero/negative")
	}
	if r.Cmp(basePoitOrderN) >= 0 || s.Cmp(basePoitOrderN) >= 0 {
		return false, errors.New("signature: r or s >= curve order")
	}

	// ---- Validate digest size ----
	if len(digest) != DigestSize {
		return false, errors.New("digest: expected 32-byte digest")
	}

	// OPTIONAL (protocol-dependent): enforce low-S to prevent malleability.
	// Many systems (Bitcoin, Ethereum, JOSE, etc.) require this.
	// If you only want "pure ECDSA validity" (как в Wycheproof), comment out this block.
	// Note: Wycheproof test vectors include both high-S and low-S signatures, so if you
	// enable this check, some Wycheproof valid signatures will be rejected.
	/*
		halfN := new(big.Int).Rsh(new(big.Int).Set(N), 1)
		if s.Cmp(halfN) == 1 {
			return false, errors.New("signature: non-canonical (high-S)")
		}
	*/

	// ---- Verify ----
	ok := ecdsa.Verify(&pub, digest, r, s) // TODO: VerifyASN1 maybe? (most applications use ASN.1 DER signatures)
	return ok, nil
}
