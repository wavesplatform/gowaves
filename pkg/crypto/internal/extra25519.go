package internal

// FeMontgomeryXToEdwardsY compare to fe_montx_to_edy
func FeMontgomeryXToEdwardsY(out, x *FieldElement) {
	/*
	   	 y = (u - 1) / (u + 1)

	    	 NOTE: u=-1 is converted to y=0 since fe_invert is mod-exp
	*/
	var t, tt FieldElement
	FeOne(&t)
	FeAdd(&tt, x, &t)   // u+1
	FeInvert(&tt, &tt)  // 1/(u+1)
	FeSub(&t, x, &t)    // u-1
	FeMul(out, &tt, &t) // (u-1)/(u+1)
}

// HashToEdwards converts a 256-bit hash output into a point on the Edwards
// curve isomorphic to Curve25519 in a manner that preserves
// collision-resistance. The returned curve points are NOT indistinguishable
// from random even if the hash value is.
// Specifically, first one bit of the hash output is set aside for parity and
// the rest is truncated and fed into the elligator bijection (which covers half
// of the points on the elliptic curve).
func HashToEdwards(out *ExtendedGroupElement, h *[32]byte) {
	hh := *h
	bit := hh[31] >> 7
	hh[31] &= 127
	FeFromBytes(&out.Y, &hh)
	representativeToMontgomeryX(&out.X, &out.Y)
	FeMontgomeryXToEdwardsY(&out.Y, &out.X)
	if ok := out.FromParityAndY(bit, &out.Y); !ok {
		panic("HashToEdwards: point not on curve")
	}
}

var lMinus1 = [32]byte{0xec, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
	0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10}

// ScNeg computes:
// b = -a (mod l)
//
//	where l = 2^252 + 27742317777372353535851937790883648493.
func ScNeg(b, a *[32]byte) {
	var zero [32]byte
	ScMulAdd(b, &lMinus1, a, &zero)
}

// ScCMove is equivalent to FeCMove but operates directly on the [32]byte
// representation instead on the FieldElement. Can be used to spare a
// FieldElement.FromBytes operation.
func ScCMove(f, g *[32]byte, b int32) {
	var x [32]byte
	for i := range x {
		x[i] = f[i] ^ g[i]
	}
	b = -b
	for i := range x {
		x[i] &= byte(b)
	}
	for i := range f {
		f[i] ^= x[i]
	}
}

func ExtendedGroupElementCopy(t, u *ExtendedGroupElement) {
	FeCopy(&t.X, &u.X)
	FeCopy(&t.Y, &u.Y)
	FeCopy(&t.Z, &u.Z)
	FeCopy(&t.T, &u.T)
}

func ExtendedGroupElementCMove(t, u *ExtendedGroupElement, b int32) {
	FeCMove(&t.X, &u.X, b)
	FeCMove(&t.Y, &u.Y, b)
	FeCMove(&t.Z, &u.Z, b)
	FeCMove(&t.T, &u.T, b)
}

// GeAdd sets r = a+b. r may overlaop with a and b.
func GeAdd(r, a, b *ExtendedGroupElement) {
	var bca CachedGroupElement
	b.ToCached(&bca)
	var rc CompletedGroupElement
	geAdd(&rc, a, &bca)
	rc.ToExtended(r)
}

// GeScalarMult sets r = a*A
// where a = a[0]+256*a[1]+...+256^31 a[31].
func GeScalarMult(r *ExtendedGroupElement, a *[32]byte, A *ExtendedGroupElement) {
	var p, q ExtendedGroupElement
	q.Zero()
	ExtendedGroupElementCopy(&p, A)
	for i := uint(0); i < 256; i++ {
		bit := int32(a[i>>3]>>(i&7)) & 1
		var t ExtendedGroupElement
		GeAdd(&t, &q, &p)
		ExtendedGroupElementCMove(&q, &t, bit)
		GeDouble(&p, &p)
	}
	ExtendedGroupElementCopy(r, &q)
}

func FeIsequal(f, g FieldElement) int {
	var h FieldElement
	FeSub(&h, &f, &g)
	return 1 ^ (1 & (feIsNonzero(h) >> 8))
}

func feIsNonzero(f FieldElement) int {
	var s [32]byte
	FeToBytes(&s, &f)
	var zero [32]byte

	return FeCompare(&s, &zero)
}

func FeCompare(x, y *[32]byte) int {
	d := 0
	for i := 0; i < 32; i++ {
		d |= int(x[i]) ^ int(y[i])
	}
	return (1 & ((d - 1) >> 8)) - 1
}

// GeIsNeutral
// returns 1 if p is the neutral point
// returns 0 otherwise
func GeIsNeutral(p *ExtendedGroupElement) bool {
	var zero FieldElement
	FeZero(&zero)
	//  Check if p == neutral element == (0, 1)
	return FeIsequal(p.X, zero)&FeIsequal(p.Y, p.Z) == 1
}

// chi calculates out = z^((p-1)/2). The result is either 1, 0, or -1 depending
// on whether z is a non-zero square, zero, or a non-square.
func chi(out, z *FieldElement) {
	var t0, t1, t2, t3 FieldElement
	var i int

	FeSquare(&t0, z)        // 2^1
	FeMul(&t1, &t0, z)      // 2^1 + 2^0
	FeSquare(&t0, &t1)      // 2^2 + 2^1
	FeSquare(&t2, &t0)      // 2^3 + 2^2
	FeSquare(&t2, &t2)      // 4,3
	FeMul(&t2, &t2, &t0)    // 4,3,2,1
	FeMul(&t1, &t2, z)      // 4..0
	FeSquare(&t2, &t1)      // 5..1
	for i = 1; i < 5; i++ { // 9,8,7,6,5
		FeSquare(&t2, &t2)
	}
	FeMul(&t1, &t2, &t1)     // 9,8,7,6,5,4,3,2,1,0
	FeSquare(&t2, &t1)       // 10..1
	for i = 1; i < 10; i++ { // 19..10
		FeSquare(&t2, &t2)
	}
	FeMul(&t2, &t2, &t1)     // 19..0
	FeSquare(&t3, &t2)       // 20..1
	for i = 1; i < 20; i++ { // 39..20
		FeSquare(&t3, &t3)
	}
	FeMul(&t2, &t3, &t2)     // 39..0
	FeSquare(&t2, &t2)       // 40..1
	for i = 1; i < 10; i++ { // 49..10
		FeSquare(&t2, &t2)
	}
	FeMul(&t1, &t2, &t1)     // 49..0
	FeSquare(&t2, &t1)       // 50..1
	for i = 1; i < 50; i++ { // 99..50
		FeSquare(&t2, &t2)
	}
	FeMul(&t2, &t2, &t1)      // 99..0
	FeSquare(&t3, &t2)        // 100..1
	for i = 1; i < 100; i++ { // 199..100
		FeSquare(&t3, &t3)
	}
	FeMul(&t2, &t3, &t2)     // 199..0
	FeSquare(&t2, &t2)       // 200..1
	for i = 1; i < 50; i++ { // 249..50
		FeSquare(&t2, &t2)
	}
	FeMul(&t1, &t2, &t1)    // 249..0
	FeSquare(&t1, &t1)      // 250..1
	for i = 1; i < 4; i++ { // 253..4
		FeSquare(&t1, &t1)
	}
	FeMul(out, &t1, &t0) // 253..4,2,1
}

// representativeToMontgomeryX consumes the rr2 input
func representativeToMontgomeryX(v, rr2 *FieldElement) {
	var e FieldElement
	FeSquare2(rr2, rr2)
	rr2[0]++
	FeInvert(rr2, rr2)
	FeMul(v, &A, rr2)
	FeNeg(v, v)

	var v2, v3 FieldElement
	FeSquare(&v2, v)
	FeMul(&v3, v, &v2)
	FeAdd(&e, &v3, v)
	FeMul(&v2, &v2, &A)
	FeAdd(&e, &v2, &e)
	chi(&e, &e)
	var eBytes [32]byte
	FeToBytes(&eBytes, &e)
	// eBytes[1] is either 0 (for e = 1) or 0xff (for e = -1)
	eIsMinus1 := int32(eBytes[1]) & 1
	var negV FieldElement
	FeNeg(&negV, v)
	FeCMove(v, &negV, eIsMinus1)

	FeZero(&v2)
	FeCMove(&v2, &A, eIsMinus1)
	FeSub(v, v, &v2)
}
