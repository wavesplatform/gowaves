package bls

import (
	"crypto/rand"
	"errors"
	"io"
	"math/big"

	"github.com/kilic/bn254"
)

var Order = bn254.Order

type PointG1 = bn254.PointG1 // 32 * 2 bytes -> signature
type PointG2 = bn254.PointG2 // 32 * 4 bytes -> pubkey

type PublicKey struct {
	point *PointG2
}

type SecretKey [32]byte

type Signature struct {
	point *PointG1
}

type AggregatedKey = PublicKey

type AggregatedSignature = Signature

type KeyPair struct {
	secret *SecretKey
	Public *PublicKey
}

type Message = []byte
type Domain = []byte

type BLSSigner struct {
	Domain  []byte
	Account *KeyPair
}

type BLSVerifier struct {
	Domain []byte
}

func PublicKeyFromBytes(in []byte) (*PublicKey, error) {
	g := bn254.NewG2()
	publicKey, err := g.FromBytes(in)
	if err != nil {
		return nil, err
	}
	return &PublicKey{publicKey}, nil
}

func (p *PublicKey) ToBytes() []byte {
	g := bn254.NewG2()
	return g.ToBytes(p.point)
}

func SignatureKeyFromBytes(in []byte) (*Signature, error) {
	g := bn254.NewG1()
	signature, err := g.FromBytes(in)
	if err != nil {
		return nil, err
	}
	return &Signature{signature}, nil
}

func (p *Signature) ToBytes() []byte {
	g := bn254.NewG1()
	return g.ToBytes(p.point)
}

func NewBLSSigner(domain Domain, account *KeyPair) *BLSSigner {
	return &BLSSigner{domain, account}
}

func NewBLSVerifier(domain Domain) *BLSVerifier {
	return &BLSVerifier{domain}
}

func NewKeyPair(r io.Reader) (*KeyPair, error) {
	s, err := rand.Int(r, Order)
	if err != nil {
		return nil, err
	}
	secret := &SecretKey{}
	copy(secret[32-len(s.Bytes()):], s.Bytes()[:])
	g2 := bn254.NewG2()
	public := g2.New()
	g2.MulScalar(public, g2.One(), s)
	return &KeyPair{secret, &PublicKey{public}}, nil
}

func NewKeyPairFromBytes(in []byte) (*KeyPair, error) {
	if len(in) != 128+32 {
		return nil, errors.New("160 byte input is required to recover")
	}
	g2 := bn254.NewG2()
	publicKey, err := g2.FromBytes(in[:128])
	if err != nil {
		return nil, err
	}
	secretKey := &SecretKey{}
	copy(secretKey[:], in[128:])
	return &KeyPair{secretKey, &PublicKey{publicKey}}, nil
}

func NewKeyPairFromSecret(in []byte) (*KeyPair, error) {
	if len(in) != 32 {
		return nil, errors.New("32 byte input is required to make new key pair")
	}
	g2 := bn254.NewG2()
	secretKey := &SecretKey{}
	copy(secretKey[:], in[:])
	publicKey := g2.New()
	g2.MulScalar(publicKey, g2.One(), new(big.Int).SetBytes(in))
	return &KeyPair{secretKey, &PublicKey{publicKey}}, nil
}

func (e *KeyPair) ToBytes() []byte {
	out := make([]byte, 128+32)
	copy(out[:128], e.Public.ToBytes())
	copy(out[128:], e.secret[:])
	return out
}

func (signer *BLSSigner) Sign(message Message) (*Signature, error) {
	g := bn254.NewG1()
	signature, err := g.HashToCurveFT(message, signer.Domain)
	if err != nil {
		return nil, err
	}
	g.MulScalar(signature, signature, new(big.Int).SetBytes(signer.Account.secret[:]))
	return &Signature{signature}, nil
}

func (verifier *BLSVerifier) AggregatePublicKeys(keys []*PublicKey) *AggregatedKey {
	g := bn254.NewG2()
	if len(keys) == 0 {
		return &AggregatedKey{g.Zero()}
	}
	aggregated := new(PointG2).Set(keys[0].point)
	for i := 1; i < len(keys); i++ {
		g.Add(aggregated, aggregated, keys[i].point)
	}
	return &AggregatedKey{aggregated}
}

func (verifier *BLSVerifier) AggregateSignatures(signatures []*Signature) *AggregatedSignature {
	g := bn254.NewG1()
	if len(signatures) == 0 {
		return &AggregatedSignature{g.Zero()}
	}
	aggregated := new(PointG1).Set(signatures[0].point)
	for i := 1; i < len(signatures); i++ {
		g.Add(aggregated, aggregated, signatures[i].point)
	}
	return &AggregatedSignature{aggregated}
}

func (verifier *BLSVerifier) Verify(message Message, signature *Signature, publicKey *PublicKey) (bool, error) {
	e := bn254.NewEngine()
	g1, g2 := e.G1, e.G2
	M, err := g1.HashToCurveFT(message, verifier.Domain)
	if err != nil {
		return false, err
	}
	e.AddPair(M, publicKey.point)
	e.AddPairInv(signature.point, g2.One())
	return e.Check(), nil
}

func (verifier *BLSVerifier) VerifyAggregateCommon(message Message, publicKeys []*PublicKey, signature *AggregatedSignature) (bool, error) {
	if len(publicKeys) == 0 {
		return false, errors.New("public key size is zero")
	}
	e := bn254.NewEngine()
	g1, g2 := e.G1, e.G2
	M, err := g1.HashToCurveFT(message, verifier.Domain)
	if err != nil {
		return false, err
	}
	aggregatedPublicKeys := verifier.AggregatePublicKeys(publicKeys)
	e.AddPair(M, aggregatedPublicKeys.point)
	e.AddPairInv(signature.point, g2.One())
	return e.Check(), nil
}

func (verifier *BLSVerifier) VerifyAggregate(messages []Message, publicKeys []*PublicKey, signature *AggregatedSignature) (bool, error) {
	if len(publicKeys) == 0 {
		return false, errors.New("public key size is zero")
	}
	if len(messages) != len(publicKeys) {
		return false, errors.New("message and key sizes must be equal")
	}
	e := bn254.NewEngine()
	g1, g2 := e.G1, e.G2
	e.AddPairInv(signature.point, g2.One())
	for i := 0; i < len(messages); i++ {
		M, err := g1.HashToCurveFT(messages[i], verifier.Domain)
		if err != nil {
			return false, err
		}
		e.AddPair(M, publicKeys[i].point)
	}
	return e.Check(), nil
}
