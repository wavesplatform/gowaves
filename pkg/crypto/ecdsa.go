package crypto

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
)

var curve = btcec.S256()

func ECDSARecoverPublicKey(digest, signature []byte) (*btcec.PublicKey, error) {
	s := make([]byte, 65)
	if signature[64] < 27 {
		signature[64] += 27
	}
	s[0] = signature[64]
	copy(s[1:], signature)
	pub, _, err := btcec.RecoverCompact(curve, s, digest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to recover public key")
	}
	return pub, nil
}
