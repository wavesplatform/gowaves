package crypto

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	btcECDSA "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/pkg/errors"
)

func ECDSARecoverPublicKey(digest, signature []byte) (*btcec.PublicKey, error) {
	const (
		signatureLen = 65
		legacyV27    = 27
		legacyV28    = legacyV27 + 1
	)
	if len(signature) != signatureLen {
		return nil, errors.Errorf("signature must be %d bytes long", signatureLen)
	}
	s := [signatureLen]byte{}
	v := signature[signatureLen-1]
	if v < legacyV27 {
		v += legacyV27
	}
	if v != legacyV27 && v != legacyV28 {
		return nil, errors.Errorf("invalid signature (v=%d is not %d or %d)", v, legacyV27, legacyV28)
	}
	s[0] = v
	copy(s[1:], signature)
	pub, _, err := btcECDSA.RecoverCompact(s[:], digest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to recover public key")
	}
	return pub, nil
}

// ECDSASign calculates an ECDSA signature.
//
// This function is susceptible to chosen plaintext attacks that can leak
// information about the private key that is used for signing. Callers must
// be aware that the given hash cannot be chosen by an adversery. Common
// solution is to hash any input before calculating the signature.
//
// The produced signature is in the [R || S || V] format where V is 0 or 1.
func ECDSASign(digest []byte, sk *btcec.PrivateKey) ([]byte, error) {
	if len(digest) != 32 {
		return nil, errors.Errorf("hash is required to be exactly 32 bytes (%d)", len(digest))
	}
	sig, err := btcECDSA.SignCompact(sk, digest, false)
	if err != nil {
		return nil, err
	}
	// Convert to Ethereum signature format with 'recovery id' v at the end.
	v := sig[0] - 27
	copy(sig, sig[1:])
	sig[64] = v
	return sig, nil
}

func ECDSAParsePublicKeyFromHex(hexString string) (*btcec.PublicKey, error) {
	data, err := hex.DecodeString(strings.TrimPrefix(hexString, "0x"))
	if err != nil {
		return nil, err
	}
	return ECDSAParsePublicKey(data)
}

func ECDSAParsePublicKey(data []byte) (*btcec.PublicKey, error) {
	return btcec.ParsePubKey(data)
}

func ECDSANewPrivateKey() (*btcec.PrivateKey, error) {
	return btcec.NewPrivateKey()
}

// ECDSAPrivateKeyFromHexString creates btcec.PrivateKey from hex string with appropriate checks.
func ECDSAPrivateKeyFromHexString(hexString string) (*btcec.PrivateKey, error) {
	d, err := hex.DecodeString(strings.TrimPrefix(hexString, "0x"))
	if err != nil {
		return nil, err
	}
	return ECDSAPrivateKeyFromBytes(d)
}

// ECDSAPrivateKeyFromBytes creates btcec.PrivateKey from 'd' PrivateKey parameter with appropriate checks.
func ECDSAPrivateKeyFromBytes(d []byte) (*btcec.PrivateKey, error) {
	sk := new(btcec.PrivateKey)
	if overflow := sk.Key.SetByteSlice(d); overflow || sk.Key.IsZero() {
		return nil, fmt.Errorf("invalid private key")
	}
	return sk, nil
}
