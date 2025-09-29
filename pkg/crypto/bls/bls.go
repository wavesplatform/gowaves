package bls

import (
	"errors"
	"fmt"
	"strings"

	cbls "github.com/cloudflare/circl/sign/bls"
	"github.com/mr-tron/base58"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const (
	SecretKeySize = 32
	PublicKeySize = 48
	SignatureSize = 96
)

var (
	ErrNoSignatures       = errors.New("no signatures")
	ErrDuplicateSignature = errors.New("duplicate signature")
)

// SecretKey is 32-byte BLS secret key.
type SecretKey [SecretKeySize]byte

func (k *SecretKey) Bytes() []byte {
	return k[:]
}

func (k *SecretKey) String() string {
	return base58.Encode(k[:])
}

func (k *SecretKey) toCIRCLSecretKey() (*cbls.PrivateKey[cbls.G1], error) {
	sk := new(cbls.PrivateKey[cbls.G1])
	if err := sk.UnmarshalBinary(k[:]); err != nil {
		return nil, fmt.Errorf("failed to get CIRCL secret key: %w", err)
	}
	return sk, nil
}

func (k *SecretKey) PublicKey() (PublicKey, error) {
	sk, err := k.toCIRCLSecretKey()
	if err != nil {
		return PublicKey{}, fmt.Errorf("failed to get public key: %w", err)
	}
	pkb, err := sk.PublicKey().MarshalBinary()
	if err != nil {
		return PublicKey{}, fmt.Errorf("failed to get public key: %w", err)
	}
	var pk PublicKey
	copy(pk[:], pkb[:PublicKeySize])
	return pk, nil
}

// NewSecretKeyFromBytes creates BLS secret key from given slice of bytes.
func NewSecretKeyFromBytes(b []byte) (SecretKey, error) {
	if l := len(b); l != SecretKeySize {
		return SecretKey{}, crypto.NewIncorrectLengthError("BLS SecretKey", l, SecretKeySize)
	}
	var sk SecretKey
	copy(sk[:], b[:SecretKeySize])
	return sk, nil
}

func NewSecretKeyFromBase58(s string) (SecretKey, error) {
	var sk SecretKey
	b, err := base58.Decode(s)
	if err != nil {
		return sk, err
	}
	if l := len(b); l != SecretKeySize {
		return sk, crypto.NewIncorrectLengthError("BLS SecretKey", l, SecretKeySize)
	}
	copy(sk[:], b[:SecretKeySize])
	return sk, nil
}

// PublicKey is 48-byte compressed BLS public key.
type PublicKey [PublicKeySize]byte

func (k PublicKey) MarshalJSON() ([]byte, error) {
	return common.ToBase58JSON(k[:]), nil
}

func (k *PublicKey) UnmarshalJSON(value []byte) error {
	b, err := common.FromBase58JSON(value, PublicKeySize, "publicKey")
	if err != nil {
		return err
	}
	copy(k[:], b[:PublicKeySize])
	return nil
}

func (k PublicKey) String() string {
	return base58.Encode(k[:])
}

func (k *PublicKey) Bytes() []byte {
	return k[:]
}

func (k *PublicKey) toCIRCLPublicKey() (*cbls.PublicKey[cbls.G1], error) {
	pk := new(cbls.PublicKey[cbls.G1])
	if err := pk.UnmarshalBinary(k[:]); err != nil {
		return nil, fmt.Errorf("failed to get CIRCL public key: %w", err)
	}
	return pk, nil
}

// NewPublicKeyFromBase58 creates PublicKey from base58-encoded string.
func NewPublicKeyFromBase58(s string) (PublicKey, error) {
	var pk PublicKey
	b, err := base58.Decode(s)
	if err != nil {
		return pk, err
	}
	if l := len(b); l != PublicKeySize {
		return pk, crypto.NewIncorrectLengthError("BLS PublicKey", l, PublicKeySize)
	}
	copy(pk[:], b[:PublicKeySize])
	return pk, nil
}

// NewPublicKeyFromBytes creates PublicKey from byte slice.
func NewPublicKeyFromBytes(b []byte) (PublicKey, error) {
	var pk PublicKey
	if l := len(b); l != PublicKeySize {
		return pk, crypto.NewIncorrectLengthError("BLS PublicKey", l, PublicKeySize)
	}
	copy(pk[:], b[:PublicKeySize])
	return pk, nil
}

// Signature is 96-byte compressed BLS signature.
type Signature [SignatureSize]byte

func (s Signature) MarshalJSON() ([]byte, error) {
	return common.ToBase58JSON(s[:]), nil
}

func (s *Signature) UnmarshalJSON(value []byte) error {
	b, err := common.FromBase58JSON(value, PublicKeySize, "publicKey")
	if err != nil {
		return err
	}
	copy(s[:], b[:PublicKeySize])
	return nil
}

func (s Signature) String() string {
	return base58.Encode(s[:])
}

func (s Signature) ShortString() string {
	const ellipsis = 0x2026 // Ellipsis symbol like '...'.
	str := base58.Encode(s[:])
	sb := new(strings.Builder)
	sb.WriteString(str[:6])
	sb.WriteRune(ellipsis)
	sb.WriteString(str[len(str)-6:])
	return sb.String()
}

func (s *Signature) Bytes() []byte {
	return s[:]
}

func NewSignatureFromBytes(b []byte) (Signature, error) {
	var s Signature
	if l := len(b); l != SignatureSize {
		return s, crypto.NewIncorrectLengthError("BLS Signature", l, SignatureSize)
	}
	copy(s[:], b[:SignatureSize])
	return s, nil
}

func NewSignatureFromBase58(s string) (Signature, error) {
	var sig Signature
	b, err := base58.Decode(s)
	if err != nil {
		return sig, err
	}
	if l := len(b); l != SignatureSize {
		return sig, crypto.NewIncorrectLengthError("BLS Signature", l, SignatureSize)
	}
	copy(sig[:], b[:SignatureSize])
	return sig, nil
}

// Sign calculates 96-byte compressed BLS signature over msg.
// Default separation tag "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_" is used.
func Sign(sk SecretKey, msg []byte) (Signature, error) {
	csk, err := sk.toCIRCLSecretKey()
	if err != nil {
		return Signature{}, fmt.Errorf("failed to sign: %w", err)
	}
	s := cbls.Sign[cbls.G1](csk, msg)
	return NewSignatureFromBytes(s)
}

func Verify(pk PublicKey, msg []byte, sig Signature) (bool, error) {
	cpk, err := pk.toCIRCLPublicKey()
	if err != nil {
		return false, fmt.Errorf("failed to verify signature: %w", err)
	}
	return cbls.Verify[cbls.G1](cpk, msg, sig.Bytes()), nil
}

func AggregateSignatures(sigs []Signature) (cbls.Signature, error) {
	if len(sigs) == 0 {
		return nil, ErrNoSignatures
	}
	if !isUnique(sigs) {
		return nil, ErrDuplicateSignature
	}
	// min-pk => keys in G1, so aggregate in G2 with tag G1{}
	ss := make([]cbls.Signature, len(sigs))
	for i := range sigs {
		ss[i] = sigs[i].Bytes()
	}
	return cbls.Aggregate(cbls.G1{}, ss)
}

// VerifyAggregate verifies aggregated signature over the same message.
func VerifyAggregate(pks []PublicKey, msg []byte, sig cbls.Signature) bool {
	if len(pks) == 0 {
		return false
	}
	if !isUnique(pks) {
		return false
	}
	ks := make([]*cbls.PublicKey[cbls.G1], len(pks))
	ms := make([][]byte, len(pks))
	for i := range pks {
		k := new(cbls.PublicKey[cbls.G1])
		if err := k.UnmarshalBinary(pks[i].Bytes()); err != nil {
			return false
		}
		ks[i] = k
		ms[i] = msg
	}
	return cbls.VerifyAggregate[cbls.G1](ks, ms, sig)
}

func isUnique[T comparable](in []T) bool {
	seen := make(map[T]struct{}, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return true
}
