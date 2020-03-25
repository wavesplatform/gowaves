package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"strings"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal"
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

type Digest [DigestSize]byte

func (d Digest) String() string {
	return base58.Encode(d[:])
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
	out := make([]byte, len(d))
	copy(out, d[:])
	return out
}

func (d Digest) MarshalBinary() ([]byte, error) {
	b := make([]byte, DigestSize)
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
	if len(b) != 32 {
		return Digest{}, errors.New("invalid digest len")
	}
	var r [32]byte
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

func Keccak256(data []byte) (Digest, error) {
	var d Digest
	h := sha3.NewLegacyKeccak256()
	if _, err := h.Write(data); err != nil {
		return d, err
	}
	h.Sum(d[:0])
	return d, nil
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
	var pk PublicKey
	s := [SecretKeySize]byte(sk)
	var ed internal.ExtendedGroupElement
	internal.GeScalarMultBase(&ed, &s)
	var edYPlusOne = new(internal.FieldElement)
	internal.FeAdd(edYPlusOne, &ed.Y, &ed.Z)
	var oneMinusEdY = new(internal.FieldElement)
	internal.FeSub(oneMinusEdY, &ed.Z, &ed.Y)
	var invOneMinusEdY = new(internal.FieldElement)
	internal.FeInvert(invOneMinusEdY, oneMinusEdY)
	var montX = new(internal.FieldElement)
	internal.FeMul(montX, edYPlusOne, invOneMinusEdY)
	p := new([PublicKeySize]byte)
	internal.FeToBytes(p, montX)
	copy(pk[:], p[:])
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
	var edPubKeyPoint internal.ExtendedGroupElement
	sk := [SecretKeySize]byte(secretKey)
	internal.GeScalarMultBase(&edPubKeyPoint, &sk)

	var edPubKey = new([PublicKeySize]byte)
	edPubKeyPoint.ToBytes(edPubKey)
	signBit := edPubKey[31] & 0x80
	s, err := sign(&sk, edPubKey, data)
	if err != nil {
		return sig, err
	}
	s[63] &= 0x7f
	s[63] |= signBit
	copy(sig[:], s[:SignatureSize])
	return sig, nil
}

func sign(curvePrivateKey, edPublicKey *[DigestSize]byte, data []byte) ([SignatureSize]byte, error) {
	var signature [64]byte
	var prefix = bytes.Repeat([]byte{0xff}, 32)
	prefix[0] = 0xfe

	random := make([]byte, 64)
	if _, err := rand.Read(random); err != nil {
		return signature, err
	}

	var messageDigest, hramDigest [64]byte
	h := sha512.New()
	if _, err := h.Write(prefix); err != nil {
		return signature, err
	}
	if _, err := h.Write(curvePrivateKey[:]); err != nil {
		return signature, err
	}
	if _, err := h.Write(data); err != nil {
		return signature, err
	}
	if _, err := h.Write(random); err != nil {
		return signature, err
	}
	h.Sum(messageDigest[:0])

	var messageDigestReduced [32]byte
	internal.ScReduce(&messageDigestReduced, &messageDigest)
	var R internal.ExtendedGroupElement
	internal.GeScalarMultBase(&R, &messageDigestReduced)

	var encodedR [32]byte
	R.ToBytes(&encodedR)

	h.Reset()
	if _, err := h.Write(encodedR[:]); err != nil {
		return signature, err
	}
	if _, err := h.Write(edPublicKey[:]); err != nil {
		return signature, err
	}
	if _, err := h.Write(data); err != nil {
		return signature, err
	}
	h.Sum(hramDigest[:0])
	var hramDigestReduced [32]byte
	internal.ScReduce(&hramDigestReduced, &hramDigest)

	var s [32]byte
	internal.ScMulAdd(&s, &hramDigestReduced, curvePrivateKey, &messageDigestReduced)

	copy(signature[:], encodedR[:])
	copy(signature[32:], s[:])
	return signature, nil
}

func Verify(publicKey PublicKey, signature Signature, data []byte) bool {
	pk := [DigestSize]byte(publicKey)
	var montX = new(internal.FieldElement)
	internal.FeFromBytes(montX, &pk)

	var one = new(internal.FieldElement)
	internal.FeOne(one)
	var montXMinusOne = new(internal.FieldElement)
	internal.FeSub(montXMinusOne, montX, one)
	var montXPlusOne = new(internal.FieldElement)
	internal.FeAdd(montXPlusOne, montX, one)
	var invMontXPlusOne = new(internal.FieldElement)
	internal.FeInvert(invMontXPlusOne, montXPlusOne)
	var edY = new(internal.FieldElement)
	internal.FeMul(edY, montXMinusOne, invMontXPlusOne)

	var edPubKey = new([PublicKeySize]byte)
	internal.FeToBytes(edPubKey, edY)

	edPubKey[31] &= 0x7F
	edPubKey[31] |= signature[63] & 0x80

	s := new([SignatureSize]byte)
	copy(s[:], signature[:])
	s[63] &= 0x7f

	return verify(edPubKey, data, s)
}

func verify(publicKey *[PublicKeySize]byte, message []byte, sig *[SignatureSize]byte) bool {
	if sig[63]&224 != 0 {
		return false
	}

	var A internal.ExtendedGroupElement
	if !A.FromBytes(publicKey) {
		return false
	}
	internal.FeNeg(&A.X, &A.X)
	internal.FeNeg(&A.T, &A.T)

	h := sha512.New()
	_, _ = h.Write(sig[:32])
	_, _ = h.Write(publicKey[:])
	_, _ = h.Write(message)
	var digest [64]byte
	h.Sum(digest[:0])

	var hReduced [32]byte
	internal.ScReduce(&hReduced, &digest)

	var R internal.ProjectiveGroupElement
	var s [32]byte
	copy(s[:], sig[32:])

	// https://tools.ietf.org/html/rfc8032#section-5.1.7 requires that s be in
	// the range [0, order) in order to prevent signature malleability.
	if !internal.ScMinimal(&s) {
		return false
	}

	internal.GeDoubleScalarMultVartime(&R, &hReduced, &A, &s)

	var checkR [32]byte
	R.ToBytes(&checkR)
	return bytes.Equal(sig[:32], checkR[:])
}

func array32FromBase58(s, name string) ([32]byte, error) {
	var r [32]byte
	b, err := base58.Decode(s)
	if err != nil {
		return r, err
	}
	if l := len(b); l != 32 {
		return r, fmt.Errorf("incorrect %s length %d, expected %d", name, l, 32)
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
		return r, fmt.Errorf("incorrect %s length %d, expected %d", name, l, 64)
	}
	copy(r[:], b[:64])
	return r, nil
}
