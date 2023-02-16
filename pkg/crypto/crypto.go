package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	edwards "filippo.io/edwards25519"
	"filippo.io/edwards25519/field"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

const (
	DigestSize    = 32
	KeySize       = 32
	PublicKeySize = KeySize
	SecretKeySize = KeySize
	SignatureSize = 64
)

var (
	prefix = []byte{
		0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
	one = new(field.Element).One()
)

type Digest [DigestSize]byte

func (d Digest) String() string {
	return base58.Encode(d[:])
}

func (d Digest) Hex() string {
	return hex.EncodeToString(d[:])
}

func (d Digest) ShortString() string {
	str := base58.Encode(d[:])
	sb := new(strings.Builder)
	sb.WriteString(str[:6])
	sb.WriteRune(0x2026) //22ef
	sb.WriteString(str[len(str)-6:])
	return sb.String()
}

func (d Digest) Bytes() []byte {
	return d[:]
}

func (d Digest) MarshalBinary() ([]byte, error) {
	return d[:], nil
}

func (d *Digest) UnmarshalBinary(data []byte) error {
	if l := len(data); l < DigestSize {
		return errors.Errorf("failed unmarshal Digest, required %d bytes, got %d", DigestSize, l)
	}
	copy(d[:], data[:DigestSize])
	return nil
}

func (d Digest) MarshalJSON() ([]byte, error) {
	return common.ToBase58JSON(d[:]), nil
}

func (d *Digest) UnmarshalJSON(value []byte) error {
	b, err := common.FromBase58JSON(value, DigestSize, "Digest")
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
	if len(b) != DigestSize {
		return Digest{}, errors.New("invalid digest len")
	}
	var r Digest
	copy(r[:], b)
	return r, nil
}

func MustDigestFromBase58(s string) Digest {
	rs, err := array32FromBase58(s, "Digest")
	if err != nil {
		panic(err.Error())
	}
	return rs
}

func MustBytesFromBase58(s string) []byte {
	b, err := base58.Decode(s)
	if err != nil {
		panic(err)
	}
	return b
}

type SecretKey [SecretKeySize]byte

func (k SecretKey) MarshalBinary() ([]byte, error) {
	b := make([]byte, SecretKeySize)
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
	return common.ToBase58JSON(k[:]), nil
}

func (k *SecretKey) UnmarshalJSON(value []byte) error {
	b, err := common.FromBase58JSON(value, SecretKeySize, "SecretKey")
	if err != nil {
		return err
	}
	copy(k[:], b[:SecretKeySize])
	return nil
}

func (k SecretKey) String() string {
	return base58.Encode(k[:])
}

func (k SecretKey) Bytes() []byte {
	out := make([]byte, len(k))
	copy(out, k[:])
	return out
}

func NewSecretKeyFromBase58(s string) (SecretKey, error) {
	return array32FromBase58(s, "SecretKey")
}

func MustSecretKeyFromBase58(s string) SecretKey {
	rs, err := NewSecretKeyFromBase58(s)
	if err != nil {
		panic(err)
	}
	return rs
}

func NewSecretKeyFromBytes(b []byte) (SecretKey, error) {
	var sk SecretKey
	if l := len(b); l != SecretKeySize {
		return sk, fmt.Errorf("invalid array length %d, expected exact %d bytes", l, SecretKeySize)
	}
	copy(sk[:], b)
	return sk, nil
}

type PublicKey [PublicKeySize]byte

func (k PublicKey) MarshalBinary() ([]byte, error) {
	b := make([]byte, PublicKeySize)
	copy(b, k[:])
	return b, nil
}

func (k PublicKey) WriteTo(buf []byte) error {
	if len(buf) < PublicKeySize {
		return errors.New("buffer is too small")
	}
	copy(buf, k[:])
	return nil
}

func (k *PublicKey) UnmarshalBinary(data []byte) error {
	if l := len(data); l < PublicKeySize {
		return errors.Errorf("failed unmarshal PublicKey, required %d bytes, got %d", PublicKeySize, l)
	}
	copy(k[:], data[:PublicKeySize])
	return nil
}

func (k PublicKey) MarshalJSON() ([]byte, error) {
	return common.ToBase58JSON(k[:]), nil
}

func (k *PublicKey) UnmarshalJSON(value []byte) error {
	b, err := common.FromBase58JSON(value, PublicKeySize, "PublicKey")
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

func NewPublicKeyFromBase58(s string) (PublicKey, error) {
	return array32FromBase58(s, "PublicKey")
}

func MustPublicKeyFromBase58(s string) PublicKey {
	rs, err := NewPublicKeyFromBase58(s)
	if err != nil {
		panic(err)
	}
	return rs
}

func NewPublicKeyFromBytes(b []byte) (PublicKey, error) {
	var pk PublicKey
	if l := len(b); l < PublicKeySize {
		return pk, fmt.Errorf("insufficient array length %d, expected atleast %d", l, PublicKeySize)
	}
	copy(pk[:], b[:PublicKeySize])
	return pk, nil
}

type Signature [SignatureSize]byte

func (s Signature) String() string {
	return base58.Encode(s[:])
}

func (s Signature) ShortString() string {
	str := base58.Encode(s[:])
	sb := new(strings.Builder)
	sb.WriteString(str[:6])
	sb.WriteRune(0x2026) //22ef
	sb.WriteString(str[len(str)-6:])
	return sb.String()
}

func (s Signature) MarshalBinary() ([]byte, error) {
	b := make([]byte, SignatureSize)
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
	return common.ToBase58JSON(s[:]), nil
}

func (s *Signature) UnmarshalJSON(value []byte) error {
	b, err := common.FromBase58JSON(value, SignatureSize, "Signature")
	if err != nil {
		return err
	}
	copy(s[:], b[:SignatureSize])
	return nil
}

func (s Signature) Bytes() []byte {
	out := make([]byte, len(s))
	copy(out, s[:])
	return out
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

func MustSignatureFromBase58(s string) Signature {
	rs, err := array64FromBase58(s, "Signature")
	if err != nil {
		panic(err.Error())
	}
	return rs
}

func MustKeccak256(data []byte) Digest {
	d, err := Keccak256(data)
	if err != nil {
		panic(errors.Errorf("BUG, CREATE REPORT: failed to calculate Keccak256 hash: %v", err))
	}
	return d
}

func Keccak256(data []byte) (Digest, error) {
	var d Digest
	h := sha3.NewLegacyKeccak256()
	if _, err := h.Write(data); err != nil {
		return d, err
	}
	h.Sum(d[:0])
	return d, nil
}

func NewFastHash() (hash.Hash, error) {
	return blake2b.New256(nil)
}

func FastHash(data []byte) (Digest, error) {
	var d Digest
	h, err := blake2b.New256(nil)
	if err != nil {
		return d, err
	}
	if _, err := h.Write(data); err != nil {
		return d, err
	}
	h.Sum(d[:0])
	return d, nil
}

func MustFastHash(data []byte) Digest {
	d, err := FastHash(data)
	if err != nil {
		panic(err.Error())
	}
	return d
}

func SecureHash(data []byte) (Digest, error) {
	var d Digest
	fh, err := blake2b.New256(nil)
	if err != nil {
		return d, err
	}
	if _, err := fh.Write(data); err != nil {
		return d, err
	}
	fh.Sum(d[:0])
	h := sha3.NewLegacyKeccak256()
	if _, err := h.Write(d[:DigestSize]); err != nil {
		return d, err
	}
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
	s, err := new(edwards.Scalar).SetBytesWithClamping(sk[:])
	if err != nil { // The only possible error is on size check
		panic(err)
	}
	p := new(edwards.Point).ScalarBaseMult(s)
	var pk PublicKey
	copy(pk[:], p.BytesMontgomery())
	return pk
}

func GenerateKeyPair(seed []byte) (SecretKey, PublicKey, error) {
	var sk SecretKey
	var pk PublicKey
	h := sha256.New()
	if _, err := h.Write(seed); err != nil {
		return sk, pk, err
	}
	digest := h.Sum(nil)
	sk = GenerateSecretKey(digest)
	pk = GeneratePublicKey(sk)
	return sk, pk, nil
}

func Sign(secretKey SecretKey, data []byte) (Signature, error) {
	var sig Signature
	sks, err := edwards.NewScalar().SetBytesWithClamping(secretKey[:])
	if err != nil {
		return sig, err
	}
	pkp := new(edwards.Point).ScalarBaseMult(sks)
	pkb := pkp.Bytes()
	sf := pkb[31] & 0x80

	random := make([]byte, sha512.Size)
	if _, err := rand.Read(random); err != nil {
		return sig, err
	}

	md := make([]byte, 0, sha512.Size)
	h := sha512.New()
	if _, err := h.Write(prefix); err != nil {
		return sig, err
	}
	if _, err := h.Write(sks.Bytes()); err != nil {
		return sig, err
	}
	if _, err := h.Write(data); err != nil {
		return sig, err
	}
	if _, err := h.Write(random); err != nil {
		return sig, err
	}
	md = h.Sum(md)

	rs, err := edwards.NewScalar().SetUniformBytes(md)
	if err != nil {
		return sig, err
	}

	rp := new(edwards.Point).ScalarBaseMult(rs)

	hd := make([]byte, 0, sha512.Size)
	h.Reset()
	if _, err := h.Write(rp.Bytes()); err != nil {
		return sig, err
	}
	if _, err := h.Write(pkb); err != nil {
		return sig, err
	}
	if _, err := h.Write(data); err != nil {
		return sig, err
	}
	hd = h.Sum(hd)

	ks, err := edwards.NewScalar().SetUniformBytes(hd)
	if err != nil {
		return sig, err
	}

	ss := edwards.NewScalar().MultiplyAdd(ks, sks, rs)

	copy(sig[:DigestSize], rp.Bytes())
	copy(sig[DigestSize:], ss.Bytes())

	sig[63] &= 0x7f
	sig[63] |= sf
	return sig, nil
}

func Verify(publicKey PublicKey, sig Signature, data []byte) bool {
	pk := publicKeyFromMontgomery(publicKey, sig[63])
	sig[63] &= 0x7f
	if sig[63]&224 != 0 {
		return false
	}
	ap, err := new(edwards.Point).SetBytes(pk)
	if err != nil {
		return false
	}
	h := sha512.New()
	if _, err := h.Write(sig[:32]); err != nil {
		return false
	}
	if _, err := h.Write(pk); err != nil {
		return false
	}
	if _, err := h.Write(data); err != nil {
		return false
	}
	hd := make([]byte, 0, sha512.Size)
	hd = h.Sum(hd)
	ks, err := edwards.NewScalar().SetUniformBytes(hd)
	if err != nil {
		return false
	}
	ss, err := edwards.NewScalar().SetCanonicalBytes(sig[32:])
	if err != nil {
		return false
	}
	nap := new(edwards.Point).Negate(ap)
	rp := new(edwards.Point).VarTimeDoubleScalarBaseMult(ks, nap, ss)
	return bytes.Equal(sig[:32], rp.Bytes())
}

func publicKeyFromMontgomery(publicKey PublicKey, sb byte) []byte {
	x, err := new(field.Element).SetBytes(publicKey[:])
	if err != nil {
		panic(err)
	}
	xMinusOne := new(field.Element).Subtract(x, one)
	xPlusOne := new(field.Element).Add(x, one)
	invXPlusOne := new(field.Element).Invert(xPlusOne)
	y := new(field.Element).Multiply(xMinusOne, invXPlusOne)

	pk := y.Bytes()
	pk[31] &= 0x7F
	pk[31] |= sb & 0x80
	return pk
}

func array32FromBase58(s, name string) ([32]byte, error) {
	var r [32]byte
	b, err := base58.Decode(s)
	if err != nil {
		return r, err
	}
	if l := len(b); l != 32 {
		return r, NewIncorrectLengthError(name, l, 32)
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
		return r, NewIncorrectLengthError(name, l, 64)
	}
	copy(r[:], b[:64])
	return r, nil
}
