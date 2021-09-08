package crypto

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
)

var (
	curve         = btcec.S256()
	secp256k1N, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
)

func ECDSARecoverPublicKey(hash, signature []byte) (*btcec.PublicKey, error) {
	s := [65]byte{}
	if signature[64] < 27 {
		signature[64] += 27
	}
	s[0] = signature[64]
	copy(s[1:], signature)
	pub, _, err := btcec.RecoverCompact(curve, s[:], hash)
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
func ECDSASign(hash []byte, prv *btcec.PrivateKey) ([]byte, error) {
	if len(hash) != 32 {
		return nil, errors.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}
	if prv.Curve != curve {
		return nil, errors.Errorf("private key curve is not secp256k1")
	}
	sig, err := btcec.SignCompact(curve, prv, hash, false)
	if err != nil {
		return nil, err
	}
	// Convert to Ethereum signature format with 'recovery id' v at the end.
	v := sig[0] - 27
	copy(sig, sig[1:])
	sig[64] = v
	return sig, nil
}

func ECDSAParsePublicKey(data []byte) (*btcec.PublicKey, error) {
	return btcec.ParsePubKey(data, curve)
}

func ECDSANewPrivateKey() (*btcec.PrivateKey, error) {
	return btcec.NewPrivateKey(curve)
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
	priv := new(btcec.PrivateKey)
	priv.PublicKey.Curve = curve

	// strictly checking bit size
	if 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}
