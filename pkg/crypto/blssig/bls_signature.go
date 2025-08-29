package blssig

import (
	"errors"

	cbls "github.com/cloudflare/circl/sign/bls"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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

func AggregateSignatures(sigs []cbls.Signature) (cbls.Signature, error) {
	if len(sigs) == 0 {
		return nil, errors.New("no signatures")
	}
	// min-pk => keys in G1, so aggregate in G2 with tag G1{}
	return cbls.Aggregate(cbls.G1{}, sigs)
}

func AggregateFromWavesSecrets(
	wavesSKs []crypto.SecretKey,
	msg []byte,
) (cbls.Signature, []*cbls.PublicKey[cbls.G1], error) {
	if len(wavesSKs) == 0 {
		return nil, nil, errors.New("no keys")
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
	msgs := make([][]byte, len(pubs))
	for i := range msgs {
		msgs[i] = msg
	}
	return cbls.VerifyAggregate[cbls.G1](pubs, msgs, aggSig)
}
