package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"github.com/agl/ed25519"
	"github.com/agl/ed25519/edwards25519"
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

type SecretKey [SecretKeySize]byte

type PublicKey [PublicKeySize]byte

type Signature [SignatureSize]byte

func Keccak256(data []byte) (digest Digest) {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	h.Sum(digest[:0])
	return
}

func FastHash(data []byte) (digest Digest, err error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return
	}
	h.Write(data)
	h.Sum(digest[:0])
	return
}

func SecureHash(data []byte) (digest Digest, err error) {
	fh, err := blake2b.New256(nil)
	if err != nil {
		return
	}
	fh.Write(data)
	fh.Sum(digest[:0])
	h := sha3.NewLegacyKeccak256()
	h.Write(digest[:DigestSize])
	h.Sum(digest[:0])
	return
}

func GenerateSecretKey(seed []byte) (sk SecretKey) {
	copy(sk[:], seed[:SecretKeySize])
	sk[0] &= 248
	sk[31] &= 127
	sk[31] |= 64
	return sk
}

func GeneratePublicKey(sk SecretKey) (pk PublicKey) {
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

func GenerateKeyPair(seed []byte) (sk SecretKey, pk PublicKey) {
	h := sha256.New()
	h.Write(seed)
	digest := h.Sum(nil)
	sk = GenerateSecretKey(digest)
	pk = GeneratePublicKey(sk)
	return
}

func Sign(secretKey SecretKey, data []byte) (sig Signature) {
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
	return
}

func sign(curvePrivateKey, edPublicKey *[DigestSize]byte, data []byte) (signature *[SignatureSize]byte) {
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

	signature = new([64]byte)
	copy(signature[:], encodedR[:])
	copy(signature[32:], s[:])
	return
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
