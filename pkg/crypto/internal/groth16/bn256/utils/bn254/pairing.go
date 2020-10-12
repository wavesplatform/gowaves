package bn254

type pair struct {
	g1 *PointG1
	g2 *PointG2
}

func newPair(g1 *PointG1, g2 *PointG2) pair {
	return pair{g1, g2}
}

// Engine is BN254 elliptic curve pairing engine
type Engine struct {
	G1   *G1
	G2   *G2
	fp12 *fp12
	fp6  *fp6
	fp2  *fp2
	pairingEngineTemp
	pairs []pair
}

// NewEngine creates new pairing engine insteace.
func NewEngine() *Engine {
	fp2 := newFp2()
	fp6 := newFp6(fp2)
	fp12 := newFp12(fp6)
	g1 := NewG1()
	g2 := newG2(fp2)
	return &Engine{
		fp2:               fp2,
		fp6:               fp6,
		fp12:              fp12,
		G1:                g1,
		G2:                g2,
		pairingEngineTemp: newEngineTemp(),
	}
}

type pairingEngineTemp struct {
	t2  [10]*fe2
	t12 [11]fe12
}

func newEngineTemp() pairingEngineTemp {
	t2 := [10]*fe2{}
	for i := 0; i < 10; i++ {
		t2[i] = &fe2{}
	}
	t12 := [11]fe12{}
	return pairingEngineTemp{t2, t12}
}

// AddPair adds a g1, g2 point pair to pairing engine
func (e *Engine) AddPair(g1 *PointG1, g2 *PointG2) *Engine {
	return e.addPair(e.G1.New().Set(g1), e.G2.New().Set(g2))
}

// AddPairInv adds a G1, G2 point pair to pairing engine. G1 point is negated.
func (e *Engine) AddPairInv(g1 *PointG1, g2 *PointG2) *Engine {
	ng1 := e.G1.New().Set(g1)
	e.G1.Neg(ng1, ng1)
	return e.addPair(ng1, e.G2.New().Set(g2))
}

func (e *Engine) addPair(g1 *PointG1, g2 *PointG2) *Engine {
	p := newPair(g1, g2)
	if !e.isZero(p) {
		e.affine(p)
		e.pairs = append(e.pairs, p)
	}
	return e
}

// Reset removes pair stack.
func (e *Engine) Reset() *Engine {
	e.pairs = []pair{}
	return e
}

func (e *Engine) isZero(p pair) bool {
	return e.G1.IsZero(p.g1) || e.G2.IsZero(p.g2)
}

func (e *Engine) affine(p pair) {
	e.G1.Affine(p.g1)
	e.G2.Affine(p.g2)
}

func (e *Engine) doublingStep(coeff *[3]fe2, r *PointG2) {
	// Adaptation of Formula 3 in https://eprint.iacr.org/2010/526.pdf
	fp2 := e.fp2
	t := e.t2
	fp2.mul(t[0], &r[0], &r[1])
	fp2.mulByFq(t[0], t[0], twoInv)
	fp2.square(t[1], &r[1])
	fp2.square(t[2], &r[2])
	fp2.double(t[7], t[2])
	fp2.addAssign(t[7], t[2])
	fp2.mul(t[3], t[7], b2)
	fp2.double(t[4], t[3])
	fp2.addAssign(t[4], t[3])
	fp2.add(t[5], t[1], t[4])
	fp2.mulByFq(t[5], t[5], twoInv)
	fp2.add(t[6], &r[1], &r[2])
	fp2.square(t[6], t[6])
	fp2.add(t[7], t[2], t[1])
	fp2.subAssign(t[6], t[7])
	fp2.sub(&coeff[2], t[3], t[1])
	fp2.square(t[7], &r[0])
	fp2.sub(t[4], t[1], t[4])
	fp2.mul(&r[0], t[4], t[0])
	fp2.square(t[2], t[3])
	fp2.double(t[3], t[2])
	fp2.addAssign(t[3], t[2])
	fp2.square(t[5], t[5])
	fp2.sub(&r[1], t[5], t[3])
	fp2.mul(&r[2], t[1], t[6])
	fp2.double(t[0], t[7])
	fp2.add(&coeff[1], t[0], t[7])
	fp2.neg(&coeff[0], t[6])
}

func (e *Engine) additionStep(coeff *[3]fe2, r, q *PointG2) {
	// Algorithm 12 in https://eprint.iacr.org/2010/526.pdf
	fp2 := e.fp2
	t := e.t2
	fp2.mul(t[0], &q[1], &r[2])
	fp2.neg(t[0], t[0])
	fp2.add(t[0], t[0], &r[1])
	fp2.mul(t[1], &q[0], &r[2])
	fp2.neg(t[1], t[1])
	fp2.add(t[1], t[1], &r[0])
	fp2.square(t[2], t[0])
	fp2.square(t[3], t[1])
	fp2.mul(t[4], t[1], t[3])
	fp2.mul(t[2], &r[2], t[2])
	fp2.mul(t[3], &r[0], t[3])
	fp2.double(t[5], t[3])
	fp2.sub(t[5], t[4], t[5])
	fp2.addAssign(t[5], t[2])
	fp2.mul(&r[0], t[1], t[5])
	fp2.sub(t[2], t[3], t[5])
	fp2.mulAssign(t[2], t[0])
	fp2.mul(t[3], &r[1], t[4])
	fp2.sub(&r[1], t[2], t[3])
	fp2.mul(&r[2], &r[2], t[4])
	fp2.mul(t[2], t[1], &q[1])
	fp2.mul(t[3], t[0], &q[0])
	fp2.sub(&coeff[2], t[3], t[2])
	fp2.neg(&coeff[1], t[0])
	coeff[0].set(t[1])
}

func (e *Engine) prepare(ellCoeffs *[102][3]fe2, twistPoint *PointG2) {
	// Algorithm 5 in  https://eprint.iacr.org/2019/077.pdf
	if e.G2.IsZero(twistPoint) {
		return
	}
	r := new(PointG2).Set(twistPoint)
	j := 0
	for i := int(sixUPlus2.BitLen() - 2); i >= 0; i-- {
		e.doublingStep(&ellCoeffs[j], r)
		j++
		if sixUPlus2.Bit(i) != 0 {
			ellCoeffs[j] = fe6{}
			e.additionStep(&ellCoeffs[j], r, twistPoint)
			j++
		}
	}
	j = len(ellCoeffs) - 2
	Q1 := new(PointG2)
	e.fp2.conjugate(&Q1[0], &twistPoint[0])
	e.fp2.conjugate(&Q1[1], &twistPoint[1])
	e.fp2.mulAssign(&Q1[0], &frobeniusCoeffs61[1])
	e.fp2.mulAssign(&Q1[1], &nonResidueInPMinusOver2)
	e.additionStep(&ellCoeffs[j], r, Q1)
	j++
	Q2 := new(PointG2)
	Q2.Set(twistPoint)
	e.fp2.mulAssign(&Q2[0], &frobeniusCoeffs61[2])
	e.additionStep(&ellCoeffs[j], r, Q2)
}

func (e *Engine) millerLoop(f *fe12) {
	pairs := e.pairs
	ellCoeffs := make([][102][3]fe2, len(pairs))
	for i := 0; i < len(pairs); i++ {
		e.prepare(&ellCoeffs[i], pairs[i].g2)
	}

	fp12, fp2 := e.fp12, e.fp2
	t := e.t2
	f.one()

	j := 0
	for i := sixUPlus2.BitLen() - 2; i >= 0; i-- {
		if j > 0 {
			fp12.square(f, f)
		}
		for i := 0; i <= len(pairs)-1; i++ {
			fp2.mulByFq(t[0], &ellCoeffs[i][j][0], &pairs[i].g1[1])
			fp2.mulByFq(t[1], &ellCoeffs[i][j][1], &pairs[i].g1[0])
			fp12.mulBy034Assign(f, t[0], t[1], &ellCoeffs[i][j][2])
		}
		j++
		if sixUPlus2.Bit(i) != 0 {
			for i := 0; i <= len(pairs)-1; i++ {
				fp2.mulByFq(t[0], &ellCoeffs[i][j][0], &pairs[i].g1[1])
				fp2.mulByFq(t[1], &ellCoeffs[i][j][1], &pairs[i].g1[0])
				fp12.mulBy034Assign(f, t[0], t[1], &ellCoeffs[i][j][2])
			}
			j++
		}
	}
	j = 100
	for i := 0; i <= len(pairs)-1; i++ {
		fp2.mulByFq(t[0], &ellCoeffs[i][j][0], &pairs[i].g1[1])
		fp2.mulByFq(t[1], &ellCoeffs[i][j][1], &pairs[i].g1[0])
		fp12.mulBy034Assign(f, t[0], t[1], &ellCoeffs[i][j][2])
	}
	j++
	for i := 0; i <= len(pairs)-1; i++ {
		fp2.mulByFq(t[0], &ellCoeffs[i][j][0], &pairs[i].g1[1])
		fp2.mulByFq(t[1], &ellCoeffs[i][j][1], &pairs[i].g1[0])
		fp12.mulBy034Assign(f, t[0], t[1], &ellCoeffs[i][j][2])
	}
}

func (e *Engine) exp(c, a *fe12) {
	fp12 := e.fp12
	fp12.cyclotomicExp(c, a, u)
}

func (e *Engine) finalExpX(f *fe12) {
	fp12 := e.fp12
	t := e.t12
	// easy part
	fp12.frobeniusMap(&t[0], f, 6)
	fp12.inverse(&t[1], f)
	fp12.mul(&t[2], &t[0], &t[1])
	t[1].set(&t[2])
	fp12.frobeniusMapAssign(&t[2], 2)
	fp12.mulAssign(&t[2], &t[1])
	fp12.cyclotomicSquare(&t[1], &t[2])
	fp12.conjugate(&t[1], &t[1])
	// hard part
	e.exp(&t[3], &t[2])
	fp12.cyclotomicSquare(&t[4], &t[3])
	fp12.mul(&t[5], &t[1], &t[3])
	e.exp(&t[1], &t[5])
	e.exp(&t[0], &t[1])
	e.exp(&t[6], &t[0])
	fp12.mulAssign(&t[6], &t[4])
	e.exp(&t[4], &t[6])
	fp12.conjugate(&t[5], &t[5])
	fp12.mulAssign(&t[4], &t[5])
	fp12.mulAssign(&t[4], &t[2])
	fp12.conjugate(&t[5], &t[2])
	fp12.mulAssign(&t[1], &t[2])
	fp12.frobeniusMapAssign(&t[1], 3)
	fp12.mulAssign(&t[6], &t[5])
	fp12.frobeniusMapAssign(&t[6], 1)
	fp12.mulAssign(&t[3], &t[0])
	fp12.frobeniusMapAssign(&t[3], 2)
	fp12.mulAssign(&t[3], &t[1])
	fp12.mulAssign(&t[3], &t[6])
	fp12.mul(f, &t[3], &t[4])
}

func (e *Engine) finalExp(f *fe12) {
	fp12 := e.fp12
	t := e.t12
	fp12.frobeniusMap(&t[0], f, 6)
	fp12.inverse(&t[1], f)
	fp12.mulAssign(&t[1], &t[0])
	fp12.frobeniusMap(&t[0], &t[1], 2)
	fp12.mulAssign(&t[1], &t[0])
	fp12.frobeniusMap(&t[0], &t[1], 1)
	fp12.frobeniusMap(&t[3], &t[1], 2)
	fp12.frobeniusMap(&t[4], &t[3], 1)
	e.exp(&t[2], &t[1])
	e.exp(&t[5], &t[2])
	e.exp(&t[6], &t[5])
	fp12.frobeniusMap(&t[7], &t[2], 1)
	fp12.frobeniusMap(&t[8], &t[5], 1)
	fp12.frobeniusMap(&t[9], &t[6], 1)
	fp12.frobeniusMap(&t[10], &t[5], 2)
	fp12.mulAssign(&t[3], &t[0])
	fp12.mulAssign(&t[3], &t[4])
	fp12.conjugate(&t[1], &t[1])
	fp12.conjugate(&t[4], &t[5])
	fp12.conjugate(&t[7], &t[7])
	fp12.mulAssign(&t[2], &t[8])
	fp12.conjugate(&t[2], &t[2])
	fp12.mulAssign(&t[9], &t[6])
	fp12.conjugate(&t[9], &t[9])
	fp12.cyclotomicSquare(&t[9], &t[9])
	fp12.mulAssign(&t[9], &t[2])
	fp12.mulAssign(&t[9], &t[4])
	fp12.mulAssign(&t[7], &t[4])
	fp12.mulAssign(&t[7], &t[9])
	fp12.mulAssign(&t[9], &t[10])
	fp12.cyclotomicSquare(&t[7], &t[7])
	fp12.mulAssign(&t[7], &t[9])
	fp12.square(&t[7], &t[7])
	fp12.mulAssign(&t[1], &t[7])
	fp12.mulAssign(&t[7], &t[3])
	fp12.cyclotomicSquare(&t[1], &t[1])
	fp12.mulAssign(&t[1], &t[7])
	f.set(&t[1])
}

func (e *Engine) calculate() *fe12 {
	f := e.fp12.one()
	if len(e.pairs) == 0 {
		return f
	}
	e.millerLoop(f)
	e.finalExp(f)
	return f
}

// Check computes pairing and checks if result is equal to one
func (e *Engine) Check() bool {
	return e.calculate().isOne()
}

// Result computes pairing and returns target group element as result.
func (e *Engine) Result() *E {
	r := e.calculate()
	e.Reset()
	return r
}

// GT returns target group instance.
func (e *Engine) GT() *GT {
	return NewGT()
}
