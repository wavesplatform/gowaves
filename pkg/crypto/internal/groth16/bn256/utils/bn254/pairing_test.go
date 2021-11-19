//go:build groth16
// +build groth16

package bn254

import (
	"bytes"
	"math/big"
	"testing"
)

func TestPairingExpected(t *testing.T) {
	expected := fromHex(-1, "0x108c19d15f9446f744d0f110405d3856d6cc3bda6c4d537663729f52576284170dc26f240656bbe2029bd441d77c221f0ba4c70c94b29b5f17f0f6d08745a069279db296f9d479292532c7c493d8e0722b6efae42158387564889c79fc038ee31ad9db1937fd72f4ac462173d31d3d6117411fa48dba8d499d762b47edb3b54a27ed208e7a0b55ae6e710bbfbd2fd922669c026360e37cc5b2ab8624115361042c53748bcd21a7c038fb30ddc8ac3bf0af25d7859cfbc12c30c866276c5659092b03614464f04dd772d86df88674c270ffc8747ea13e72da95e3594468f222c401676555de427abc409c4a394bc5426886302996919d4bf4bdd02236e14b36362067586885c3318eeffa1938c754fe3c60224ee5ae15e66af6b5104c47c8c5d80e841c2ac18a4003ac9326b9558380e0bc27fdd375e3605f96b819a358d34bde084f330485b09e866bc2f2ea2b897394deaf3f12aa31f28cb0552990967d470412c70e90e12b7874510cd1707e8856f71bf7f61d72631e268fca81000db9a1f5")
	bls := NewEngine()
	G1, G2 := bls.G1, bls.G2
	g1One, g2One := G1.One(), G2.One()
	bls.AddPair(g1One, g2One)
	GT := bls.GT()
	e := bls.Result()
	out := GT.ToBytes(e)
	if !bytes.Equal(out, expected) {
		t.Fatalf("expected pairing result is not met")
	}
}

func TestPairingNonDegeneracy(t *testing.T) {
	bls := NewEngine()
	G1, G2 := bls.G1, bls.G2
	g1Zero, g2Zero, g1One, g2One := G1.Zero(), G2.Zero(), G1.One(), G2.One()
	// GT := bls.GT()
	// e(g1^a, g2^b) != 1
	{
		bls.AddPair(g1One, g2One)
		e := bls.Result()
		if e.IsOne() {
			t.Fatal("pairing result is not expected to be one")
		}
	}
	// e(g1^a, 0) == 1
	bls.Reset()
	{
		bls.AddPair(g1One, g2Zero)
		e := bls.Result()
		if !e.IsOne() {
			t.Fatal("pairing result is expected to be one")
		}
	}
	// e(0, g2^b) == 1
	bls.Reset()
	{
		bls.AddPair(g1Zero, g2One)
		e := bls.Result()
		if !e.IsOne() {
			t.Fatal("pairing result is expected to be one")
		}
	}
	//
	bls.Reset()
	{
		bls.AddPair(g1Zero, g2One)
		bls.AddPair(g1One, g2Zero)
		bls.AddPair(g1Zero, g2Zero)
		e := bls.Result()
		if !e.IsOne() {
			t.Fatal("pairing result is expected to be one")
		}
	}
}

func TestPairingBilinearity(t *testing.T) {
	bls := NewEngine()
	g1, g2 := bls.G1, bls.G2
	gt := bls.GT()
	// e(a*G1, b*G2) = e(G1, G2)^c
	{
		a, b := big.NewInt(17), big.NewInt(117)
		c := new(big.Int).Mul(a, b)
		G1, G2 := g1.One(), g2.One()
		e0 := bls.AddPair(G1, G2).Result()
		P1, P2 := g1.New(), g2.New()
		g1.MulScalar(P1, G1, a)
		g2.MulScalar(P2, G2, b)
		e1 := bls.AddPair(P1, P2).Result()
		gt.Exp(e0, e0, c)
		if !e0.Equal(e1) {
			t.Fatal("bad pairing, 1")
		}
	}
	// e(a * G1, b * G2) = e((a + b) * G1, G2)
	{
		// scalars
		a, b := big.NewInt(17), big.NewInt(117)
		c := new(big.Int).Mul(a, b)
		// LHS
		G1, G2 := g1.One(), g2.One()
		g1.MulScalar(G1, G1, c)
		bls.AddPair(G1, G2)
		// RHS
		P1, P2 := g1.One(), g2.One()
		g1.MulScalar(P1, P1, a)
		g2.MulScalar(P2, P2, b)
		bls.AddPairInv(P1, P2)
		// should be one
		if !bls.Check() {
			t.Fatal("bad pairing, 2")
		}
	}
	// e(a * G1, b * G2) = e((a + b) * G1, G2)
	{
		// scalars
		a, b := big.NewInt(17), big.NewInt(117)
		c := new(big.Int).Mul(a, b)
		// LHS
		G1, G2 := g1.One(), g2.One()
		g2.MulScalar(G2, G2, c)
		bls.AddPair(G1, G2)
		// RHS
		H1, H2 := g1.One(), g2.One()
		g1.MulScalar(H1, H1, a)
		g2.MulScalar(H2, H2, b)
		bls.AddPairInv(H1, H2)
		// should be one
		if !bls.Check() {
			t.Fatal("bad pairing, 3")
		}
	}
}

func TestPairingMulti(t *testing.T) {
	// e(G1, G2) ^ t == e(a01 * G1, a02 * G2) * e(a11 * G1, a12 * G2) * ... * e(an1 * G1, an2 * G2)
	// where t = sum(ai1 * ai2)
	bls := NewEngine()
	g1, g2 := bls.G1, bls.G2
	numOfPair := 100
	targetExp := new(big.Int)
	// RHS
	for i := 0; i < numOfPair; i++ {
		// (ai1 * G1, ai2 * G2)
		a1, a2 := randScalar(q), randScalar(q)
		P1, P2 := g1.One(), g2.One()
		g1.MulScalar(P1, P1, a1)
		g2.MulScalar(P2, P2, a2)
		bls.AddPair(P1, P2)
		// accumulate targetExp
		// t += (ai1 * ai2)
		a1.Mul(a1, a2)
		targetExp.Add(targetExp, a1)
	}
	// LHS
	// e(t * G1, G2)
	T1, T2 := g1.One(), g2.One()
	g1.MulScalar(T1, T1, targetExp)
	bls.AddPairInv(T1, T2)
	if !bls.Check() {
		t.Fatal("fail multi pairing")
	}
}

func TestPairingEmpty(t *testing.T) {
	bls := NewEngine()
	if !bls.Check() {
		t.Fatal("empty check should be accepted")
	}
	if !bls.Result().IsOne() {
		t.Fatal("empty pairing result should be one")
	}
}

func BenchmarkPairing(t *testing.B) {
	bls := NewEngine()
	g1, g2, gt := bls.G1, bls.G2, bls.GT()
	bls.AddPair(g1.One(), g2.One())
	e := gt.New()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		e = bls.calculate()
	}
	_ = e
}
