package bn256

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bn256/utils/bn254" //nolint
)

const (
	g1ReprLen   = 32
	g2ReprLen   = 64
	minProofLen = g1ReprLen + g2ReprLen + g1ReprLen // len(A G1) + len(B G2) + len(C G1)
)

type Proof struct {
	A *bn254.PointG1
	B *bn254.PointG2
	C *bn254.PointG1
}

func GetProofFromCompressed(proof []byte) (*Proof, error) {
	if l := len(proof); l < minProofLen {
		return nil, errors.Errorf("insufficient proof len: wanted at least %d, got %d", minProofLen, l)
	}

	var (
		aG1Repr = proof[:g1ReprLen]
		bG2Repr = proof[g1ReprLen : g1ReprLen+g2ReprLen]
		cG1Repr = proof[g1ReprLen+g2ReprLen : minProofLen]
	)

	// A G1
	aG1, err := bn254.NewG1().FromCompressed(aG1Repr)
	if err != nil {
		return nil, err
	}

	// B G2
	bG2, err := bn254.NewG2().FromCompressed(bG2Repr)
	if err != nil {
		return nil, err
	}

	// C G1
	cG1, err := bn254.NewG1().FromCompressed(cG1Repr)
	if err != nil {
		return nil, err
	}

	return &Proof{
		A: aG1,
		B: bG2,
		C: cG1,
	}, nil
}

func (p *Proof) ToCompressed() []byte {
	var (
		g1  = bn254.NewG1()
		g2  = bn254.NewG2()
		out = make([]byte, 0, minProofLen)
	)
	out = append(out, g1.ToCompressed(p.A)...)
	out = append(out, g2.ToCompressed(p.B)...)
	out = append(out, g1.ToCompressed(p.C)...)
	return out
}
