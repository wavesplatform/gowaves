package crypto

import (
	"crypto/rand"
	"crypto/sha512"
	"hash"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal"
)

// This code is a port of the public domain, VXEdDSA C implementation by Trevor Perrin / Open Whisper Systems.
// Specification: https://whispersystems.org/docs/specifications/xeddsa/#vxeddsa

const (
	ProofSize = 32 + 32 + 32

	labelMaxLen    = 128
	labelSetMaxLen = 512
	maxMessageSize = 1048576

	protocol = "VEdDSA_25519_SHA512_Elligator2"
)

var (
	bBytes = []byte{
		0x58, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
	}
	zeroBytes       = make([]byte, 32)
	defaultLabelSet = newLabelSet(protocol)
	defaultLabel1   = addLabel(defaultLabelSet, "1")
	defaultLabel2   = addLabel(defaultLabelSet, "2")
	defaultLabel3   = addLabel(defaultLabelSet, "3")
	defaultLabel4   = addLabel(defaultLabelSet, "4")
)

func SignVRF(sk SecretKey, msg []byte) ([]byte, error) {
	r := make([]byte, 32)
	_, err := rand.Read(r)
	if err != nil {
		return nil, err
	}
	return generateVRFSignature(r, sk[:], msg)
}

func generateVRFSignature(random, sk, m []byte) ([]byte, error) {
	if len(random) != 32 {
		random = zeroBytes
	}
	if len(sk) != SecretKeySize {
		return nil, errors.Errorf("invalid secret key size")
	}
	if len(m) > maxMessageSize {
		return nil, errors.New("message is too big")
	}

	var a, aNeg, rB [32]byte
	var R internal.ExtendedGroupElement
	var edPubKey internal.ExtendedGroupElement
	copy(a[:], sk[:32]) // a - secret key bytes
	internal.GeScalarMultBase(&edPubKey, &a)
	var A [32]byte // A - public key bytes
	edPubKey.ToBytes(&A)

	// Force Edwards sign bit to zero
	signBit := (A[31] & 0x80) >> 7
	internal.ScNeg(&aNeg, &a)
	internal.ScCMove(&a, &aNeg, int32(signBit))

	A[31] &= 0x7F

	h := sha512.New()
	// Bv = hash(hash(labelset1 || K) || M)
	// Kv = k * Bv
	Bv, V := calculateBvAndV(h, defaultLabel1, a, A, m)
	BvBytes := new([32]byte)
	Bv.ToBytes(BvBytes)

	// R, r = commit(labelset2, (Bv || Kv), (K,k), Z, M)
	pl := 32 + len(defaultLabel2) + 32
	padLen1 := (128 - (pl % 128)) % 128
	pad1 := make([]byte, padLen1)
	pl += padLen1 + 32
	padLen2 := (128 - (pl % 128)) % 128
	pad2 := make([]byte, padLen2)

	var rH [64]byte
	h.Reset()
	_, _ = h.Write(bBytes)
	_, _ = h.Write(defaultLabel2)
	_, _ = h.Write(random)
	_, _ = h.Write(pad1)
	_, _ = h.Write(a[:])
	_, _ = h.Write(pad2)
	_, _ = h.Write(defaultLabel2)
	_, _ = h.Write(A[:])
	_, _ = h.Write(BvBytes[:])
	_, _ = h.Write(V[:])
	_, _ = h.Write(m)
	h.Sum(rH[:0])

	var Rv internal.ExtendedGroupElement
	internal.ScReduce(&rB, &rH) // rB == r_scalar
	internal.GeScalarMultBase(&R, &rB)
	internal.GeScalarMult(&Rv, &rB, Bv)

	/* h = SHA512(label(4) || A || V || R || Rv || M) */
	var Rb [32]byte
	R.ToBytes(&Rb)
	var RvB [32]byte
	Rv.ToBytes(&RvB)

	// h = challenge(labelset3, (Bv || Kv || Rv), R, K, M)
	var hB [64]byte
	h.Reset()
	_, _ = h.Write(bBytes)
	_, _ = h.Write(defaultLabel3)
	_, _ = h.Write(Rb[:])
	_, _ = h.Write(defaultLabel3)
	_, _ = h.Write(A[:])
	_, _ = h.Write(BvBytes[:])
	_, _ = h.Write(V[:])
	_, _ = h.Write(RvB[:])
	_, _ = h.Write(m)
	h.Sum(hB[:0])

	var rHB [32]byte
	internal.ScReduce(&rHB, &hB)
	var s [32]byte
	internal.ScMulAdd(&s, &rHB, &a, &rB)

	signature := make([]byte, ProofSize)
	copy(signature[:32], V[:])
	copy(signature[32:64], rHB[:])
	copy(signature[64:96], s[:])

	return signature, nil
}

func VerifyVRF(pk PublicKey, msg, signature []byte) (bool, []byte, error) {
	return verifyVRFSignature(pk[:], msg, signature)
}

func verifyVRFSignature(publicKey, m, signature []byte) (bool, []byte, error) {
	if len(publicKey) != PublicKeySize {
		return false, nil, errors.New("invalid public key length")
	}
	if len(m) > maxMessageSize {
		return false, nil, errors.New("message is too big")
	}
	if len(signature) != ProofSize {
		return false, nil, errors.New("invalid signature length")
	}
	pk := new([32]byte)
	copy(pk[:], publicKey)
	var u internal.FieldElement
	internal.FeFromBytes(&u, pk)
	var strict [32]byte
	internal.FeToBytes(&strict, &u)
	if !(internal.FeCompare(&strict, pk) == 0) {
		return false, nil, nil
	}
	var y internal.FieldElement
	internal.FeMontgomeryXToEdwardsY(&y, &u)
	var edPubKey [32]byte
	internal.FeToBytes(&edPubKey, &y)

	h := sha512.New()

	Bv := calculateBv(h, defaultLabel1, edPubKey, m)
	BvBytes := new([32]byte)
	Bv.ToBytes(BvBytes)

	// Split signature into parts
	KvBytes := new([32]byte)
	copy(KvBytes[:], signature[:32])
	HvBytes := new([32]byte)
	copy(HvBytes[:], signature[32:64])
	SvBytes := new([32]byte)
	copy(SvBytes[:], signature[64:])

	if signature[63]&224 == 1 {
		return false, nil, nil
	}
	if signature[95]&224 == 1 {
		return false, nil, nil
	}

	// Load -A:
	var minusA internal.ExtendedGroupElement
	internal.FeFromBytes(&minusA.Y, &edPubKey)
	if !minusA.FromParityAndY((edPubKey[31]>>7)^0x01, &minusA.Y) {
		return false, nil, nil
	}

	// Load -V
	var minusV internal.ExtendedGroupElement
	var Vb [32]byte
	copy(Vb[:], signature[:32])
	internal.FeFromBytes(&minusV.Y, &Vb)
	if !minusV.FromParityAndY((Vb[31]>>7)^0x01, &minusV.Y) {
		return false, nil, nil
	}

	// Load h, s
	var hh, s [32]byte
	copy(hh[:], signature[32:64])
	copy(s[:], signature[64:96])
	if hh[31]&224 == 1 {
		return false, nil, nil
	} /* strict parsing of h */
	if s[31]&224 == 1 {
		return false, nil, nil
	} /* strict parsing of s */

	var A, cA, V, cV internal.ExtendedGroupElement
	internal.GeNeg(&A, minusA)
	internal.GeNeg(&V, minusV)

	internal.GeDouble(&cA, &A)
	internal.GeDouble(&cA, &cA)
	internal.GeDouble(&cA, &cA)

	internal.GeDouble(&cV, &V)
	internal.GeDouble(&cV, &cV)
	internal.GeDouble(&cV, &cV)

	if internal.GeIsNeutral(&cA) || internal.GeIsNeutral(&cV) || internal.GeIsNeutral(Bv) {
		return false, nil, nil
	}

	// R = (s*B) + (h * -A))
	var R internal.ProjectiveGroupElement
	internal.GeDoubleScalarMultVartime(&R, &hh, &minusA, &s)
	rB := new([32]byte)
	R.ToBytes(rB)

	// s * Bv
	var sBv internal.ExtendedGroupElement
	internal.GeScalarMult(&sBv, &s, Bv)

	// h * -V
	var hMinusV internal.ExtendedGroupElement
	internal.GeScalarMult(&hMinusV, &hh, &minusV)

	// Rv = (sc * Bv) + (hc * (-V))
	var Rv internal.ExtendedGroupElement
	internal.GeAdd(&Rv, &sBv, &hMinusV)
	RvBytes := new([32]byte)
	Rv.ToBytes(RvBytes)

	// Challenge
	var hB [64]byte
	h.Reset()
	_, _ = h.Write(bBytes)
	_, _ = h.Write(defaultLabel3)
	_, _ = h.Write(rB[:])
	_, _ = h.Write(defaultLabel3)
	_, _ = h.Write(edPubKey[:])
	_, _ = h.Write(BvBytes[:])
	_, _ = h.Write(KvBytes[:])
	_, _ = h.Write(RvBytes[:])
	_, _ = h.Write(m)
	h.Sum(hB[:0])
	var rHB [32]byte
	internal.ScReduce(&rHB, &hB)

	if internal.FeCompare(&rHB, HvBytes) == 0 {
		var cKvBytes [32]byte
		cV.ToBytes(&cKvBytes)
		var hV [64]byte
		h.Reset()
		_, _ = h.Write(bBytes)
		_, _ = h.Write(defaultLabel4)
		_, _ = h.Write(cKvBytes[:])
		h.Sum(hV[:0])

		return true, hV[:32], nil
	}
	return false, nil, nil
}

// H(n) = (f(h(n))^8)
func hashToPoint(digest *[64]byte) *internal.ExtendedGroupElement {
	var hmb [32]byte
	copy(hmb[:], digest[:32])
	var hm internal.ExtendedGroupElement
	internal.HashToEdwards(&hm, &hmb)
	internal.GeDouble(&hm, &hm)
	internal.GeDouble(&hm, &hm)
	internal.GeDouble(&hm, &hm)
	return &hm
}

func calculateBv(h hash.Hash, label []byte, A [32]byte, msg []byte) *internal.ExtendedGroupElement {
	/* Calculate SHA512(label || A || msg) */
	h64 := new([64]byte)
	h.Reset()
	_, _ = h.Write(bBytes)
	_, _ = h.Write(label)
	_, _ = h.Write(A[:])
	_, _ = h.Write(msg)
	h.Sum(h64[:0])
	return hashToPoint(h64)
}

func calculateBvAndV(h hash.Hash, label []byte, a, Abytes [32]byte, msg []byte) (*internal.ExtendedGroupElement, *[32]byte) {
	Bv := calculateBv(h, label, Abytes, msg)
	var p3 internal.ExtendedGroupElement
	internal.GeScalarMult(&p3, &a, Bv)
	V := new([32]byte)
	p3.ToBytes(V)
	return Bv, V
}

func newLabelSet(protocol string) []byte {
	pl := len(protocol)
	if pl > labelMaxLen {
		panic("invalid label length")
	}
	r := make([]byte, 3+pl)
	r[0] = 2
	r[1] = byte(pl)
	copy(r[2:], protocol)
	r[2+pl] = 0
	return r
}

func addLabel(set []byte, label string) []byte {
	sl := len(set)
	ll := len(label)
	if ll > labelMaxLen {
		panic("invalid label length")
	}
	if sl+1+ll > labelSetMaxLen {
		panic("invalid label set length")
	}
	count := set[0]
	r := make([]byte, sl+1+ll)
	copy(r, set)
	r[0] = count + 1
	r[sl] = byte(ll)
	copy(r[sl+1:], label)
	return r
}

// ComputeVRF generates the VRF value for the byte slice msg using given private key sk.
func ComputeVRF(sk SecretKey, msg []byte) []byte {
	var a, aNeg, A [32]byte
	copy(a[:], sk[:SecretKeySize])

	var x [32]byte
	copy(x[:], sk[:])
	var edPubKey internal.ExtendedGroupElement
	internal.GeScalarMultBase(&edPubKey, &x)
	edPubKey.ToBytes(&A)

	signBit := (A[31] & 0x80) >> 7
	internal.ScNeg(&aNeg, &a)
	internal.ScCMove(&a, &aNeg, int32(signBit))
	A[31] &= 0x7F

	h := sha512.New()
	_, vb := calculateBvAndV(h, defaultLabel1, a, A, msg)
	return computeVrfFromV(h, vb)
}

func computeVrfFromV(h hash.Hash, vb *[32]byte) []byte {
	var v, cv internal.ExtendedGroupElement
	v.FromBytes(vb)

	internal.GeDouble(&cv, &v)
	internal.GeDouble(&cv, &cv)
	internal.GeDouble(&cv, &cv)

	var cvb [32]byte
	cv.ToBytes(&cvb)

	h64 := new([64]byte)
	h.Reset()
	_, _ = h.Write(bBytes)
	_, _ = h.Write(defaultLabel4)
	_, _ = h.Write(cvb[:])
	h.Sum(h64[:0])

	vrf := make([]byte, DigestSize)
	copy(vrf, h64[:DigestSize])
	return vrf
}
