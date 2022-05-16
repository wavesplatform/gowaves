package bn256

import (
	"bytes"
	"errors"
	"io"

	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bn256/utils/bn254" //nolint
)

type VerificationKey struct {
	AlphaG1 *bn254.PointG1
	BetaG2  *bn254.PointG2
	GammaG2 *bn254.PointG2
	DeltaG2 *bn254.PointG2
	Ic      []*bn254.PointG1
}

func GetVerificationKeyFromCompressed(vk []byte) (*VerificationKey, error) {
	var (
		g1Repr = [g1ReprLen]byte{}
		g2Repr = [g2ReprLen]byte{}
		r      = bytes.NewReader(vk)
	)

	// Alpha G1
	if _, err := io.ReadFull(r, g1Repr[:]); err != nil {
		return nil, err
	}
	alphaG1, err := bn254.NewG1().FromCompressed(g1Repr[:])
	if err != nil {
		return nil, err
	}

	// Beta G2
	if _, err := io.ReadFull(r, g2Repr[:]); err != nil {
		return nil, err
	}
	betaG2, err := bn254.NewG2().FromCompressed(g2Repr[:])
	if err != nil {
		return nil, err
	}

	// Gamma G2
	if _, err := io.ReadFull(r, g2Repr[:]); err != nil {
		return nil, err
	}
	gammaG2, err := bn254.NewG2().FromCompressed(g2Repr[:])
	if err != nil {
		return nil, err
	}

	// Delta G2
	if _, err := io.ReadFull(r, g2Repr[:]); err != nil {
		return nil, err
	}
	deltaG2, err := bn254.NewG2().FromCompressed(g2Repr[:])
	if err != nil {
		return nil, err
	}

	// IC []G1
	var ic []*bn254.PointG1
	for {
		if _, err := io.ReadFull(r, g1Repr[:]); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		g1, err := bn254.NewG1().FromCompressed(g1Repr[:])
		if err != nil {
			return nil, err
		}
		ic = append(ic, g1)
	}

	return &VerificationKey{
		AlphaG1: alphaG1,
		BetaG2:  betaG2,
		GammaG2: gammaG2,
		DeltaG2: deltaG2,
		Ic:      ic,
	}, nil

}

func (v *VerificationKey) ToCompressed() []byte {
	var (
		g1  = bn254.NewG1()
		g2  = bn254.NewG2()
		out = make([]byte, 0, g1ReprLen+3*g2ReprLen+g1ReprLen*len(v.Ic))
	)
	out = append(out, g1.ToCompressed(v.AlphaG1)...)
	out = append(out, g2.ToCompressed(v.BetaG2)...)
	out = append(out, g2.ToCompressed(v.GammaG2)...)
	out = append(out, g2.ToCompressed(v.DeltaG2)...)
	for _, p := range v.Ic {
		out = append(out, g1.ToCompressed(p)...)
	}
	return out
}
