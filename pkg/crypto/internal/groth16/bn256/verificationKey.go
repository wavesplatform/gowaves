package bn256

import (
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bn256/utils/bn254" //nolint
	"io"
)

type VerificationKey struct {
	AlphaG1 *bn254.PointG1
	BetaG2  *bn254.PointG2
	GammaG2 *bn254.PointG2
	DeltaG2 *bn254.PointG2
	Ic      []*bn254.PointG1
}

func GetVerificationKeyFromCompressed(vk []byte) (*VerificationKey, error) {
	reader := bytes.NewReader(vk)

	var g1Repr = make([]byte, 32)
	var g2Repr = make([]byte, 64)

	// Alpha G1
	_, err := reader.Read(g1Repr)
	if err != nil {
		return nil, err
	}
	alphaG1, err := bn254.NewG1().FromCompressed(g1Repr)
	if err != nil {
		return nil, err
	}

	// Beta G2
	_, err = reader.Read(g2Repr)
	if err != nil {
		return nil, err
	}
	betaG2, err := bn254.NewG2().FromCompressed(g2Repr)
	if err != nil {
		return nil, err
	}

	// Gamma G2
	_, err = reader.Read(g2Repr)
	if err != nil {
		return nil, err
	}
	gammaG2, err := bn254.NewG2().FromCompressed(g2Repr)
	if err != nil {
		return nil, err
	}

	// Delta G2
	_, err = reader.Read(g2Repr)
	if err != nil {
		return nil, err
	}
	deltaG2, err := bn254.NewG2().FromCompressed(g2Repr)
	if err != nil {
		return nil, err
	}

	// IC []G1
	var ic []*bn254.PointG1
	for {
		_, err := reader.Read(g1Repr)
		if err == io.EOF {
			break
		} else if err != nil && err != io.EOF {
			return nil, err
		}

		g1, err := bn254.NewG1().FromCompressed(g1Repr)
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
