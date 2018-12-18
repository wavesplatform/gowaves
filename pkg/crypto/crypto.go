package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"strconv"
	"strings"

	"github.com/agl/ed25519"
	"github.com/agl/ed25519/edwards25519"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

const (
	DigestSize    = 32
	PublicKeySize = 32
	SecretKeySize = 32
	SignatureSize = 64
)

type Digest [DigestSize]byte

func (d Digest) String() string {
	return base58.Encode(d[:])
}

func (d Digest) Bytes() []byte {
	out := make([]byte, len(d))
	copy(out, d[:])
	return out
}

func (d Digest) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0, DigestSize)
	copy(b, d[:])
	return b, nil
}

func (d *Digest) UnmarshalBinary(data []byte) error {
	if l := len(data); l < DigestSize {
		return errors.Errorf("failed unmarshal Digest, required %d bytes, got %d", DigestSize, l)
	}
	copy(d[:], data[:DigestSize])
	return nil
}

func (d Digest) MarshalJSON() ([]byte, error) {
	return toBase58JSON(d[:]), nil
}

func (d *Digest) UnmarshalJSON(value []byte) error {
	b, err := fromBase58JSON(value, DigestSize, "Digest")
	if err != nil {
		return err
	}
	copy(d[:], b[:DigestSize])
	return nil
}

func NewDigestFromBase58(s string) (Digest, error) {
	return array32FromBase58(s, "Digest")
}

func NewDigestFromBytes(b []byte) (Digest, error) {
	if len(b) != 32 {
		return Digest{}, errors.New("invalid digest len")
	}
	var r [32]byte
	copy(r[:], b)
	return r, nil
}

type SecretKey [SecretKeySize]byte

func (k SecretKey) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0, SecretKeySize)
	copy(b, k[:])
	return b, nil
}

func (k *SecretKey) UnmarshalBinary(data []byte) error {
	if l := len(data); l < SecretKeySize {
		return errors.Errorf("failed unmarshal SecretKey, required %d bytes, got %d", SecretKeySize, l)
	}
	copy(k[:], data[:SecretKeySize])
	return nil
}

func (k SecretKey) MarshalJSON() ([]byte, error) {
	return toBase58JSON(k[:]), nil
}

func (k *SecretKey) UnmarshalJSON(value []byte) error {
	b, err := fromBase58JSON(value, SecretKeySize, "SecretKey")
	if err != nil {
		return err
	}
	copy(k[:], b[:SecretKeySize])
	return nil
}

func (k SecretKey) String() string {
	return base58.Encode(k[:])
}

func NewSecretKeyFromBase58(s string) (SecretKey, error) {
	return array32FromBase58(s, "SecretKey")
}

type PublicKey [PublicKeySize]byte

func (k PublicKey) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0, PublicKeySize)
	copy(b, k[:])
	return b, nil
}

func (k *PublicKey) UnmarshalBinary(data []byte) error {
	if l := len(data); l < PublicKeySize {
		return errors.Errorf("failed unmarshal PublicKey, required %d bytes, got %d", PublicKeySize, l)
	}
	copy(k[:], data[:PublicKeySize])
	return nil
}

func (k PublicKey) MarshalJSON() ([]byte, error) {
	return toBase58JSON(k[:]), nil
}

func (k *PublicKey) UnmarshalJSON(value []byte) error {
	b, err := fromBase58JSON(value, PublicKeySize, "PublicKey")
	if err != nil {
		return err
	}
	copy(k[:], b[:PublicKeySize])
	return nil
}

func (k *PublicKey) String() string {
	return base58.Encode(k[:])
}

func (k *PublicKey) Bytes() []byte {
	return k[:]
}

func NewPublicKeyFromBase58(s string) (PublicKey, error) {
	return array32FromBase58(s, "PublicKey")
}

func NewPublicKeyFromBytes(b []byte) (PublicKey, error) {
	if len(b) != PublicKeySize {
		return PublicKey{}, errors.New("invalid public key size")
	}
	pk := PublicKey{}
	copy(pk[:], b[:])
	return pk, nil
}

type Signature [SignatureSize]byte

func (s Signature) String() string {
	return base58.Encode(s[:])
}

func (s Signature) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0, SignatureSize)
	copy(b, s[:])
	return b, nil
}

func (s *Signature) UnmarshalBinary(data []byte) error {
	if l := len(data); l < SignatureSize {
		return errors.Errorf("failed unmarshal Signature, required %d bytes, got %d", SignatureSize, l)
	}
	copy(s[:], data[:SignatureSize])
	return nil
}

func (s Signature) MarshalJSON() ([]byte, error) {
	return toBase58JSON(s[:]), nil
}

func (s *Signature) UnmarshalJSON(value []byte) error {
	b, err := fromBase58JSON(value, SignatureSize, "Signature")
	if err != nil {
		return err
	}
	copy(s[:], b[:SignatureSize])
	return nil
}

func (s Signature) Bytes() []byte {
	return s[:]
}

func NewSignatureFromBase58(s string) (Signature, error) {
	return array64FromBase58(s, "Signature")
}

func NewSignatureFromBytes(b []byte) (Signature, error) {
	if len(b) != SignatureSize {
		return Signature{}, errors.New("invalid signature size")
	}
	s := Signature{}
	copy(s[:], b[:])
	return s, nil
}

func Keccak256(data []byte) Digest {
	var d Digest
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	h.Sum(d[:0])
	return d
}

func FastHash(data []byte) (Digest, error) {
	var d Digest
	h, err := blake2b.New256(nil)
	if err != nil {
		return d, err
	}
	h.Write(data)
	h.Sum(d[:0])
	return d, nil
}

func SecureHash(data []byte) (Digest, error) {
	var d Digest
	fh, err := blake2b.New256(nil)
	if err != nil {
		return d, err
	}
	fh.Write(data)
	fh.Sum(d[:0])
	h := sha3.NewLegacyKeccak256()
	h.Write(d[:DigestSize])
	h.Sum(d[:0])
	return d, nil
}

func GenerateSecretKey(seed []byte) SecretKey {
	var sk SecretKey
	copy(sk[:], seed[:SecretKeySize])
	sk[0] &= 248
	sk[31] &= 127
	sk[31] |= 64
	return sk
}

func GeneratePublicKey(sk SecretKey) PublicKey {
	var pk PublicKey
	s := [SecretKeySize]byte(sk)
	var ed edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&ed, &s)
	var edYPlusOne = new(edwards25519.FieldElement)
	edwards25519.FeAdd(edYPlusOne, &ed.Y, &ed.Z)
	var oneMinusEdY = new(edwards25519.FieldElement)
	edwards25519.FeSub(oneMinusEdY, &ed.Z, &ed.Y)
	var invOneMinusEdY = new(edwards25519.FieldElement)
	edwards25519.FeInvert(invOneMinusEdY, oneMinusEdY)
	var montX = new(edwards25519.FieldElement)
	edwards25519.FeMul(montX, edYPlusOne, invOneMinusEdY)
	p := new([PublicKeySize]byte)
	edwards25519.FeToBytes(p, montX)
	copy(pk[:], p[:])
	return pk
}

func GenerateKeyPair(seed []byte) (SecretKey, PublicKey) {
	h := sha256.New()
	h.Write(seed)
	digest := h.Sum(nil)
	var sk SecretKey
	var pk PublicKey
	sk = GenerateSecretKey(digest)
	pk = GeneratePublicKey(sk)
	return sk, pk
}

func Sign(secretKey SecretKey, data []byte) Signature {
	var sig Signature
	var edPubKeyPoint edwards25519.ExtendedGroupElement
	sk := [SecretKeySize]byte(secretKey)
	edwards25519.GeScalarMultBase(&edPubKeyPoint, &sk)

	var edPubKey = new([PublicKeySize]byte)
	edPubKeyPoint.ToBytes(edPubKey)
	signBit := edPubKey[31] & 0x80
	s := sign(&sk, edPubKey, data)
	s[63] &= 0x7f
	s[63] |= signBit
	copy(sig[:], s[:SignatureSize])
	return sig
}

func sign(curvePrivateKey, edPublicKey *[DigestSize]byte, data []byte) [SignatureSize]byte {
	var prefix = bytes.Repeat([]byte{0xff}, 32)
	prefix[0] = 0xfe

	random := make([]byte, 64)
	rand.Read(random)

	var messageDigest, hramDigest [64]byte
	h := sha512.New()
	h.Write(prefix)
	h.Write(curvePrivateKey[:])
	h.Write(data)
	h.Write(random)
	h.Sum(messageDigest[:0])

	var messageDigestReduced [32]byte
	edwards25519.ScReduce(&messageDigestReduced, &messageDigest)
	var R edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&R, &messageDigestReduced)

	var encodedR [32]byte
	R.ToBytes(&encodedR)

	h.Reset()
	h.Write(encodedR[:])
	h.Write(edPublicKey[:])
	h.Write(data)
	h.Sum(hramDigest[:0])
	var hramDigestReduced [32]byte
	edwards25519.ScReduce(&hramDigestReduced, &hramDigest)

	var s [32]byte
	edwards25519.ScMulAdd(&s, &hramDigestReduced, curvePrivateKey, &messageDigestReduced)

	var signature [64]byte
	copy(signature[:], encodedR[:])
	copy(signature[32:], s[:])
	return signature
}

func Verify(publicKey PublicKey, signature Signature, data []byte) bool {
	pk := [DigestSize]byte(publicKey)
	var montX = new(edwards25519.FieldElement)
	edwards25519.FeFromBytes(montX, &pk)

	var one = new(edwards25519.FieldElement)
	edwards25519.FeOne(one)
	var montXMinusOne = new(edwards25519.FieldElement)
	edwards25519.FeSub(montXMinusOne, montX, one)
	var montXPlusOne = new(edwards25519.FieldElement)
	edwards25519.FeAdd(montXPlusOne, montX, one)
	var invMontXPlusOne = new(edwards25519.FieldElement)
	edwards25519.FeInvert(invMontXPlusOne, montXPlusOne)
	var edY = new(edwards25519.FieldElement)
	edwards25519.FeMul(edY, montXMinusOne, invMontXPlusOne)

	var edPubKey = new([PublicKeySize]byte)
	edwards25519.FeToBytes(edPubKey, edY)

	edPubKey[31] &= 0x7F
	edPubKey[31] |= signature[63] & 0x80

	s := new([SignatureSize]byte)
	copy(s[:], signature[:])
	s[63] &= 0x7f

	return ed25519.Verify(edPubKey, data, s)
}

func toBase58JSON(b []byte) []byte {
	s := base58.Encode(b)
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return []byte(sb.String())
}

func fromBase58JSON(value []byte, size int, name string) ([]byte, error) {
	s := string(value)
	if s == "null" {
		return nil, nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s from JSON", name)
	}
	v, err := base58.Decode(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s from Base58 string", name)
	}
	if l := len(v); l != size {
		return nil, errors.Errorf("incorrect length %d of %s value, expected %d", l, name, DigestSize)
	}
	return v[:size], nil
}

func array32FromBase58(s, name string) ([32]byte, error) {
	var r [32]byte
	b, err := base58.Decode(s)
	if err != nil {
		return r, err
	}
	if l := len(b); l != 32 {
		return r, fmt.Errorf("incorrect %s lenght %d, expected %d", name, l, 32)
	}
	copy(r[:], b[:32])
	return r, nil
}

func array64FromBase58(s, name string) ([64]byte, error) {
	var r [64]byte
	b, err := base58.Decode(s)
	if err != nil {
		return r, err
	}
	if l := len(b); l != 64 {
		return r, fmt.Errorf("incorrect %s lenght %d, expected %d", name, l, 64)
	}
	copy(r[:], b[:64])
	return r, nil
}
