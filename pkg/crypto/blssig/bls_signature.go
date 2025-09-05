package blssig

import (
	"errors"
	"fmt"

	cbls "github.com/cloudflare/circl/sign/bls"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	G1PubKeyCompressedLen = 48 // bytes
	G2SigCompressedLen    = 96 // bytes
)

var (
	ErrNoKeys             = errors.New("no keys")
	ErrNoSignatures       = errors.New("no signatures")
	ErrDuplicatePublicKey = errors.New("duplicate public key")
	ErrDuplicateSignature = errors.New("duplicate signature")
)

func SecretKeyFromWaves(wavesSK crypto.SecretKey) (*cbls.PrivateKey[cbls.G1], error) {
	return cbls.KeyGen[cbls.G1](wavesSK[:], nil, nil)
}

// PublicKeyBytes 48-byte compressed G1 pub key.
func PublicKeyBytes(sk *cbls.PrivateKey[cbls.G1]) ([]byte, error) {
	if sk == nil {
		return nil, errors.New("nil secret key")
	}
	return sk.PublicKey().MarshalBinary()
}

// Sign 96-byte compressed G2 signature over msg.
// Default separation tag is "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_".
func Sign(sk *cbls.PrivateKey[cbls.G1], msg []byte) []byte {
	return cbls.Sign[cbls.G1](sk, msg)
}

// AggregateSignatures Default separation tag is "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_".
func AggregateSignatures(sigs []cbls.Signature) (cbls.Signature, error) {
	if len(sigs) == 0 {
		return nil, ErrNoSignatures
	}
	if err := checkNoDuplicateSignatures(sigs); err != nil {
		return nil, err
	}
	// min-pk => keys in G1, so aggregate in G2 with tag G1{}
	return cbls.Aggregate(cbls.G1{}, sigs)
}

// AggregateFromWavesSecrets Default separation tag is "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_".
func AggregateFromWavesSecrets(
	wavesSKs []crypto.SecretKey,
	msg []byte,
) (cbls.Signature, []*cbls.PublicKey[cbls.G1], error) {
	if len(wavesSKs) == 0 {
		return nil, nil, ErrNoKeys
	}
	sigs := make([]cbls.Signature, 0, len(wavesSKs))
	pubs := make([]*cbls.PublicKey[cbls.G1], 0, len(wavesSKs))
	for _, w := range wavesSKs {
		sk, err := SecretKeyFromWaves(w)
		if err != nil {
			return nil, nil, err
		}
		sigs = append(sigs, cbls.Sign[cbls.G1](sk, msg))
		pubs = append(pubs, sk.PublicKey())
	}
	if err := checkNoDuplicatePubKeys(pubs); err != nil {
		return nil, nil, err
	}
	if err := checkNoDuplicateSignatures(sigs); err != nil {
		return nil, nil, err
	}
	agg, err := cbls.Aggregate(cbls.G1{}, sigs)
	if err != nil {
		return nil, nil, err
	}
	return agg, pubs, nil
}

func VerifyAggregate(
	pubs []*cbls.PublicKey[cbls.G1],
	msg []byte,
	aggSig cbls.Signature,
) bool {
	if len(pubs) == 0 {
		return false
	}
	if err := checkNoDuplicatePubKeys(pubs); err != nil {
		return false
	}
	msgs := make([][]byte, len(pubs))
	for i := range msgs {
		msgs[i] = msg
	}
	return cbls.VerifyAggregate[cbls.G1](pubs, msgs, aggSig)
}

func checkNoDuplicatePubKeys(pubs []*cbls.PublicKey[cbls.G1]) error {
	seen := make(map[[G1PubKeyCompressedLen]byte]struct{}, len(pubs))
	for i, pk := range pubs {
		if pk == nil {
			return fmt.Errorf("nil public key at index %d", i)
		}
		b, err := pk.MarshalBinary()
		if err != nil {
			return fmt.Errorf("marshal pubkey %d: %w", i, err)
		}
		if len(b) != G1PubKeyCompressedLen {
			return fmt.Errorf("pubkey %d length %d != %d", i, len(b), G1PubKeyCompressedLen)
		}
		var k [G1PubKeyCompressedLen]byte
		copy(k[:], b)
		if _, ok := seen[k]; ok {
			return ErrDuplicatePublicKey
		}
		seen[k] = struct{}{}
	}
	return nil
}

func checkNoDuplicateSignatures(sigs []cbls.Signature) error {
	seen := make(map[[G2SigCompressedLen]byte]struct{}, len(sigs))
	for i, s := range sigs {
		if s == nil {
			return fmt.Errorf("nil signature at index %d", i)
		}
		if len(s) != G2SigCompressedLen {
			return fmt.Errorf("signature %d length %d != %d", i, len(s), G2SigCompressedLen)
		}
		var k [G2SigCompressedLen]byte
		copy(k[:], s)
		if _, ok := seen[k]; ok {
			return ErrDuplicateSignature
		}
		seen[k] = struct{}{}
	}
	return nil
}
