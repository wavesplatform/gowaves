package bn254

import (
	"errors"
	"math"
	"math/big"
)

// PointG2 is type for point in G2.
// PointG2 is both used for Affine and Jacobian point representation.
// If z is equal to one the point is accounted as in affine form.
type PointG2 [3]fe2

// Set copies valeus of one point to another.
func (p *PointG2) Set(p2 *PointG2) *PointG2 {
	p[0].set(&p2[0])
	p[1].set(&p2[1])
	p[2].set(&p2[2])
	return p
}

func (p *PointG2) Zero() *PointG2 {
	p[0].zero()
	p[1].one()
	p[2].zero()
	return p

}

type tempG2 struct {
	t [9]*fe2
}

// G2 is struct for G2 group.
type G2 struct {
	f *fp2
	tempG2
}

func (g *G2) FromCompressed(compressed []byte) (*PointG2, error) {
	if len(compressed) != 64 {
		return nil, errors.New("input string should be equal or larger than 96")
	}
	var in [64]byte
	copy(in[:], compressed[:])
	//if in[0]&(1<<7) == 0 {
	//	return nil, errors.New("bad compression")
	//}
	if in[0]&(1<<6) != 0 {
		// in[0] == (1 << 6) + (1 << 7)
		for i, v := range in {
			if (i == 0 && v != 0xc0) || (i != 0 && v != 0x00) {
				return nil, errors.New("input string should be zero when infinity flag is set")
			}
		}
		return g.Zero(), nil
	}
	a := in[0]&(1<<7) != 0
	in[0] &= 0x3f
	x, err := g.f.fromBytes(in[:])
	if err != nil {
		return nil, err
	}
	// solve curve equation
	y := &fe2{}
	g.f.square(y, x)
	g.f.mul(y, y, x)
	g.f.add(y, y, b2)
	if ok := g.f.sqrt(y, y); !ok {
		return nil, errors.New("point is not on curve")
	}
	if y.signBE() == a {
		g.f.neg(y, y)
	}
	z := new(fe2).one()
	p := &PointG2{*x, *y, *z}
	//if !g.InCorrectSubgroup(p) {
	//	return nil, errors.New("point is not on correct subgroup")
	//}
	return p, nil
}

// ToCompressed given a G2 point returns bytes in compressed form of the point.
func (g *G2) ToCompressed(p *PointG2) []byte {
	out := make([]byte, 64)
	g.Affine(p)
	if g.IsZero(p) {
		out[0] |= 1 << 6
		out[0] |= 1 << 7
	} else {
		copy(out[:], g.f.toBytes(&p[0]))
		if !p[1].signBE() {
			out[0] |= 1 << 7
		}
	}
	return out
}

// NewG2 constructs a new G2 instance.
func NewG2() *G2 {
	return newG2(nil)
}

func newG2(f *fp2) *G2 {
	if f == nil {
		f = newFp2()
	}
	t := newTempG2()
	return &G2{f, t}
}

func newTempG2() tempG2 {
	t := [9]*fe2{}
	for i := 0; i < 9; i++ {
		t[i] = &fe2{}
	}
	return tempG2{t}
}

// Q returns group order of BN254 in big.Int
func (g *G2) Q() *big.Int {
	return new(big.Int).Set(q)
}

func (g *G2) fromBytesUnchecked(in []byte) (*PointG2, error) {
	p0, err := g.f.fromBytes(in[:64])
	if err != nil {
		return nil, err
	}
	p1, err := g.f.fromBytes(in[64:])
	if err != nil {
		return nil, err
	}
	p2 := new(fe2).one()
	return &PointG2{*p0, *p1, *p2}, nil
}

// FromBytes constructs a new point given uncompressed byte input.
// FromBytes does not take zcash flags into account.
// Byte input expected to be larger than 64 bytes.
// First 128 bytes should be concatenation of x and y values
// Point (0, 0) is considered as infinity.
func (g *G2) FromBytes(in []byte) (*PointG2, error) {
	if len(in) < 128 {
		return nil, errors.New("input string should be equal or larger than 128")
	}
	p0, err := g.f.fromBytes(in[:64])
	if err != nil {
		return nil, err
	}
	p1, err := g.f.fromBytes(in[64:])
	if err != nil {
		return nil, err
	}
	// check if given input points to infinity
	if p0.isZero() && p1.isZero() {
		return g.Zero(), nil
	}
	p2 := new(fe2).one()
	p := &PointG2{*p0, *p1, *p2}
	if !g.IsOnCurve(p) {
		return nil, errors.New("point is not on curve")
	}
	return p, nil
}

// ToBytes serializes a point into bytes in uncompressed form,
// does not take zcash flags into account,
// returns (0, 0) if point is infinity.
func (g *G2) ToBytes(p *PointG2) []byte {
	out := make([]byte, 128)
	if g.IsZero(p) {
		return out
	}
	g.Affine(p)
	copy(out[:64], g.f.toBytes(&p[0]))
	copy(out[64:], g.f.toBytes(&p[1]))
	return out
}

// New creates a new G2 Point which is equal to zero in other words point at infinity.
func (g *G2) New() *PointG2 {
	return new(PointG2).Zero()
}

// Zero returns a new G2 Point which is equal to point at infinity.
func (g *G2) Zero() *PointG2 {
	return new(PointG2).Zero()
}

// One returns a new G2 Point which is equal to generator point.
func (g *G2) One() *PointG2 {
	p := &PointG2{}
	return p.Set(&g2One)
}

// IsZero returns true if given point is equal to zero.
func (g *G2) IsZero(p *PointG2) bool {
	return p[2].isZero()
}

// Equal checks if given two G2 point is equal in their affine form.
func (g *G2) Equal(p1, p2 *PointG2) bool {
	if g.IsZero(p1) {
		return g.IsZero(p2)
	}
	if g.IsZero(p2) {
		return g.IsZero(p1)
	}
	t := g.t
	g.f.square(t[0], &p1[2])
	g.f.square(t[1], &p2[2])
	g.f.mul(t[2], t[0], &p2[0])
	g.f.mul(t[3], t[1], &p1[0])
	g.f.mul(t[0], t[0], &p1[2])
	g.f.mul(t[1], t[1], &p2[2])
	g.f.mul(t[1], t[1], &p1[1])
	g.f.mul(t[0], t[0], &p2[1])
	return t[0].equal(t[1]) && t[2].equal(t[3])
}

// InCorrectSubgroup checks whether given point is in correct subgroup.
func (g *G2) InCorrectSubgroup(p *PointG2) bool {
	tmp := &PointG2{}
	g.MulScalar(tmp, p, q)
	return g.IsZero(tmp)
}

// IsOnCurve checks a G2 point is on curve.
func (g *G2) IsOnCurve(p *PointG2) bool {
	if g.IsZero(p) {
		return true
	}
	t := g.t
	g.f.square(t[0], &p[1])
	g.f.square(t[1], &p[0])
	g.f.mul(t[1], t[1], &p[0])
	g.f.square(t[2], &p[2])
	g.f.square(t[3], t[2])
	g.f.mul(t[2], t[2], t[3])
	g.f.mul(t[2], b2, t[2])
	g.f.add(t[1], t[1], t[2])
	return t[0].equal(t[1])
}

// IsAffine checks a G2 point whether it is in affine form.
func (g *G2) IsAffine(p *PointG2) bool {
	return p[2].isOne()
}

// Affine calculates affine form of given G2 point.
func (g *G2) Affine(p *PointG2) *PointG2 {
	if g.IsZero(p) {
		return p
	}
	if !g.IsAffine(p) {
		t := g.t
		g.f.inverse(t[0], &p[2])
		g.f.square(t[1], t[0])
		g.f.mul(&p[0], &p[0], t[1])
		g.f.mul(t[0], t[0], t[1])
		g.f.mul(&p[1], &p[1], t[0])
		p[2].one()
	}
	return p
}

// Add adds two G2 points p1, p2 and assigns the result to point at first argument.
func (g *G2) Add(r, p1, p2 *PointG2) *PointG2 {
	// http://www.hyperelliptic.org/EFD/gp/auto-shortw-jacobian-0.html#addition-add-2007-bl
	if g.IsZero(p1) {
		return r.Set(p2)
	}
	if g.IsZero(p2) {
		return r.Set(p1)
	}
	t := g.t
	g.f.square(t[7], &p1[2])
	g.f.mul(t[1], &p2[0], t[7])
	g.f.mul(t[2], &p1[2], t[7])
	g.f.mul(t[0], &p2[1], t[2])
	g.f.square(t[8], &p2[2])
	g.f.mul(t[3], &p1[0], t[8])
	g.f.mul(t[4], &p2[2], t[8])
	g.f.mul(t[2], &p1[1], t[4])
	if t[1].equal(t[3]) {
		if t[0].equal(t[2]) {
			return g.Double(r, p1)
		} else {
			return r.Zero()
		}
	}
	g.f.sub(t[1], t[1], t[3])
	g.f.double(t[4], t[1])
	g.f.square(t[4], t[4])
	g.f.mul(t[5], t[1], t[4])
	g.f.sub(t[0], t[0], t[2])
	g.f.double(t[0], t[0])
	g.f.square(t[6], t[0])
	g.f.sub(t[6], t[6], t[5])
	g.f.mul(t[3], t[3], t[4])
	g.f.double(t[4], t[3])
	g.f.sub(&r[0], t[6], t[4])
	g.f.sub(t[4], t[3], &r[0])
	g.f.mul(t[6], t[2], t[5])
	g.f.double(t[6], t[6])
	g.f.mul(t[0], t[0], t[4])
	g.f.sub(&r[1], t[0], t[6])
	g.f.add(t[0], &p1[2], &p2[2])
	g.f.square(t[0], t[0])
	g.f.sub(t[0], t[0], t[7])
	g.f.sub(t[0], t[0], t[8])
	g.f.mul(&r[2], t[0], t[1])
	return r
}

// Double doubles a G2 point p and assigns the result to the point at first argument.
func (g *G2) Double(r, p *PointG2) *PointG2 {
	// http://www.hyperelliptic.org/EFD/gp/auto-shortw-jacobian-0.html#doubling-dbl-2009-l
	if g.IsZero(p) {
		return r.Set(p)
	}
	t := g.t
	g.f.square(t[0], &p[0])
	g.f.square(t[1], &p[1])
	g.f.square(t[2], t[1])
	g.f.add(t[1], &p[0], t[1])
	g.f.square(t[1], t[1])
	g.f.sub(t[1], t[1], t[0])
	g.f.sub(t[1], t[1], t[2])
	g.f.double(t[1], t[1])
	g.f.double(t[3], t[0])
	g.f.add(t[0], t[3], t[0])
	g.f.square(t[4], t[0])
	g.f.double(t[3], t[1])
	g.f.sub(&r[0], t[4], t[3])
	g.f.sub(t[1], t[1], &r[0])
	g.f.double(t[2], t[2])
	g.f.double(t[2], t[2])
	g.f.double(t[2], t[2])
	g.f.mul(t[0], t[0], t[1])
	g.f.sub(t[1], t[0], t[2])
	g.f.mul(t[0], &p[1], &p[2])
	r[1].set(t[1])
	g.f.double(&r[2], t[0])
	return r
}

// Neg negates a G2 point p and assigns the result to the point at first argument.
func (g *G2) Neg(r, p *PointG2) *PointG2 {
	r[0].set(&p[0])
	g.f.neg(&r[1], &p[1])
	r[2].set(&p[2])
	return r
}

// Sub subtracts two G2 points p1, p2 and assigns the result to point at first argument.
func (g *G2) Sub(c, a, b *PointG2) *PointG2 {
	d := &PointG2{}
	g.Neg(d, b)
	g.Add(c, a, d)
	return c
}

// MulScalar multiplies a point by given scalar value in big.Int and assigns the result to point at first argument.
func (g *G2) MulScalar(c, p *PointG2, e *big.Int) *PointG2 {
	q, n := &PointG2{}, &PointG2{}
	n.Set(p)
	l := e.BitLen()
	for i := 0; i < l; i++ {
		if e.Bit(i) == 1 {
			g.Add(q, q, n)
		}
		g.Double(n, n)
	}
	return c.Set(q)
}

// MultiExp calculates multi exponentiation. Given pairs of G2 point and scalar values
// (P_0, e_0), (P_1, e_1), ... (P_n, e_n) calculates r = e_0 * P_0 + e_1 * P_1 + ... + e_n * P_n
// Length of points and scalars are expected to be equal, otherwise an error is returned.
// Result is assigned to point at first argument.
func (g *G2) MultiExp(r *PointG2, points []*PointG2, powers []*big.Int) (*PointG2, error) {
	if len(points) != len(powers) {
		return nil, errors.New("point and scalar vectors should be in same length")
	}
	var c uint32 = 3
	if len(powers) >= 32 {
		c = uint32(math.Ceil(math.Log10(float64(len(powers)))))
	}
	bucketSize, numBits := (1<<c)-1, uint32(g.Q().BitLen())
	windows := make([]*PointG2, numBits/c+1)
	bucket := make([]*PointG2, bucketSize)
	acc, sum := g.New(), g.New()
	for i := 0; i < bucketSize; i++ {
		bucket[i] = g.New()
	}
	mask := (uint64(1) << c) - 1
	j := 0
	var cur uint32
	for cur <= numBits {
		acc.Zero()
		bucket = make([]*PointG2, (1<<c)-1)
		for i := 0; i < len(bucket); i++ {
			bucket[i] = g.New()
		}
		for i := 0; i < len(powers); i++ {
			s0 := powers[i].Uint64()
			index := uint(s0 & mask)
			if index != 0 {
				g.Add(bucket[index-1], bucket[index-1], points[i])
			}
			powers[i] = new(big.Int).Rsh(powers[i], uint(c))
		}
		sum.Zero()
		for i := len(bucket) - 1; i >= 0; i-- {
			g.Add(sum, sum, bucket[i])
			g.Add(acc, acc, sum)
		}
		windows[j] = g.New()
		windows[j].Set(acc)
		j++
		cur += c
	}
	acc.Zero()
	for i := len(windows) - 1; i >= 0; i-- {
		for j := uint32(0); j < c; j++ {
			g.Double(acc, acc)
		}
		g.Add(acc, acc, windows[i])
	}
	return r.Set(acc), nil
}

// MapToPointTI maps given 64 bytes into G2 point
func (g *G2) MapToPointTI(in []byte) (*PointG2, error) {
	fp2 := g.f
	y := &fe2{}
	x, err := fp2.fromBytesUnchecked(in)
	if err != nil {
		return nil, err
	}
	one := fp2.one()
	for {
		fp2.square(y, x)
		fp2.mul(y, y, x)
		fp2.add(y, y, b2)
		if ok := fp2.sqrt(y, y); ok {
			if !y.sign() {
				fp2.neg(y, y)
			}
			p := &PointG2{*x, *y, *one}
			return p, nil
		}
		fp2.add(x, x, one)
	}
}

// func (g *G2) wnafMul(c, p *PointG2, e *big.Int) *PointG2 {
// 	windowSize := uint(6)
// 	precompTable := make([]*PointG2, (1 << (windowSize - 1)))
// 	for i := 0; i < len(precompTable); i++ {
// 		precompTable[i] = g.New()
// 	}
// 	var indexForPositive uint64 = (1 << (windowSize - 2))
// 	precompTable[indexForPositive].Set(p)
// 	g.Neg(precompTable[indexForPositive-1], p)
// 	doubled, precomp := g.New(), g.New()
// 	g.Double(doubled, p)
// 	precomp.Set(p)
// 	for i := uint64(1); i < indexForPositive; i++ {
// 		g.Add(precomp, precomp, doubled)
// 		precompTable[indexForPositive+i].Set(precomp)
// 		g.Neg(precompTable[indexForPositive-1-i], precomp)
// 	}
// 	wnaf := wnaf(e, windowSize)
// 	q := g.Zero()
// 	found := false
// 	var idx uint64
// 	for i := len(wnaf) - 1; i >= 0; i-- {
// 		if found {
// 			g.Double(q, q)
// 		}
// 		if wnaf[i] != 0 {
// 			found = true
// 			if wnaf[i] > 0 {
// 				idx = uint64(wnaf[i] >> 1)
// 				g.Add(q, q, precompTable[indexForPositive+idx])
// 			} else {
// 				idx = uint64(((0 - wnaf[i]) >> 1))
// 				g.Add(q, q, precompTable[indexForPositive-1-idx])
// 			}
// 		}
// 	}
// 	return c.Set(q)
// }

// // MapToCurve given a byte slice returns a valid G2 point.
// // This mapping function implements the Simplified Shallue-van de Woestijne-Ulas method.
// // https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-05#section-6.6.2
// // Input byte slice should be a valid field element, otherwise an error is returned.
// func (g *G2) MapToCurve(in []byte) (*PointG2, error) {
// 	fp2 := g.f
// 	u, err := fp2.fromBytes(in)
// 	if err != nil {
// 		return nil, err
// 	}
// 	x, y := swuMapG2(fp2, u)
// 	isogenyMapG2(fp2, x, y)
// 	z := new(fe2).one()
// 	q := &PointG2{*x, *y, *z}
// 	g.ClearCofactor(q)
// 	return g.Affine(q), nil
// }

// // EncodeToCurve given a message and domain seperator tag returns the hash result
// // which is a valid curve point.
// // Implementation follows BLS12381G1_XMD:SHA-256_SSWU_NU_ suite at
// // https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
// func (g *G2) EncodeToCurve(msg, domain []byte) (*PointG2, error) {
// 	hashRes, err := hashToFpXMDSHA256(msg, domain, 2)
// 	if err != nil {
// 		return nil, err
// 	}
// 	fp2 := g.f
// 	u := &fe2{*hashRes[0], *hashRes[1]}
// 	x, y := swuMapG2(fp2, u)
// 	isogenyMapG2(fp2, x, y)
// 	z := new(fe2).one()
// 	q := &PointG2{*x, *y, *z}
// 	g.ClearCofactor(q)
// 	return g.Affine(q), nil
// }

// // HashToCurve given a message and domain seperator tag returns the hash result
// // which is a valid curve point.
// // Implementation follows BLS12381G1_XMD:SHA-256_SSWU_RO_ suite at
// // https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
// func (g *G2) HashToCurve(msg, domain []byte) (*PointG2, error) {
// 	hashRes, err := hashToFpXMDSHA256(msg, domain, 4)
// 	if err != nil {
// 		return nil, err
// 	}
// 	fp2 := g.f
// 	u0, u1 := &fe2{*hashRes[0], *hashRes[1]}, &fe2{*hashRes[2], *hashRes[3]}
// 	x0, y0 := swuMapG2(fp2, u0)
// 	x1, y1 := swuMapG2(fp2, u1)
// 	z0 := new(fe2).one()
// 	z1 := new(fe2).one()
// 	p0, p1 := &PointG2{*x0, *y0, *z0}, &PointG2{*x1, *y1, *z1}
// 	g.Add(p0, p0, p1)
// 	g.Affine(p0)
// 	isogenyMapG2(fp2, &p0[0], &p0[1])
// 	g.ClearCofactor(p0)
// 	return g.Affine(p0), nil
// }
