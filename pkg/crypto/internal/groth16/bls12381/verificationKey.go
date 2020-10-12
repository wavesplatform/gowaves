package bls12381

import (
	"bytes"
	bls "github.com/kilic/bls12-381"
	"io"
)

type VerificationKey struct {
	AlphaG1 *bls.PointG1
	BetaG2  *bls.PointG2
	GammaG2 *bls.PointG2
	DeltaG2 *bls.PointG2
	Ic      []*bls.PointG1
}

func GetVerificationKeyFromCompressed(vk []byte) (*VerificationKey, error) {
	reader := bytes.NewReader(vk)

	var g1Repr = make([]byte, 48)
	var g2Repr = make([]byte, 96)

	// Alpha G1
	_, err := reader.Read(g1Repr)
	if err != nil {
		return nil, err
	}
	alphaG1, err := bls.NewG1().FromCompressed(g1Repr)
	if err != nil {
		return nil, err
	}

	// Beta G2
	_, err = reader.Read(g2Repr)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	betaG2, err := bls.NewG2().FromCompressed(g2Repr)
	if err != nil {
		return nil, err
	}

	// Gamma G2
	_, err = reader.Read(g2Repr)
	if err != nil {
		return nil, err
	}
	gammaG2, err := bls.NewG2().FromCompressed(g2Repr)
	if err != nil {
		return nil, err
	}

	// Delta G2
	_, err = reader.Read(g2Repr)
	if err != nil {
		return nil, err
	}
	deltaG2, err := bls.NewG2().FromCompressed(g2Repr)
	if err != nil {
		return nil, err
	}

	// IC []G1
	var ic []*bls.PointG1
	for {
		_, err := reader.Read(g1Repr)
		if err == io.EOF {
			break
		} else if err != nil && err != io.EOF {
			return nil, err
		}

		g1, err := bls.NewG1().FromCompressed(g1Repr)
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
