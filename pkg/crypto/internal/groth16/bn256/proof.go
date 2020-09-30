package bn256

import (
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bn256/utils/bn254" //nolint
)

type Proof struct {
	A *bn254.PointG1
	B *bn254.PointG2
	C *bn254.PointG1
}

func GetProofFromCompressed(proof []byte) (*Proof, error) {
	reader := bytes.NewReader(proof)

	var g1Repr = make([]byte, 32)
	var g2Repr = make([]byte, 64)

	// A G1
	_, err := reader.Read(g1Repr)
	if err != nil {
		return nil, err
	}
	aG1, err := bn254.NewG1().FromCompressed(g1Repr)
	if err != nil {
		return nil, err
	}

	// B G2
	_, err = reader.Read(g2Repr)
	if err != nil {
		return nil, err
	}
	bG2, err := bn254.NewG2().FromCompressed(g2Repr)
	if err != nil {
		return nil, err
	}

	// C G1
	_, err = reader.Read(g1Repr)
	if err != nil {
		return nil, err
	}
	cG1, err := bn254.NewG1().FromCompressed(g1Repr)
	if err != nil {
		return nil, err
	}

	return &Proof{
		A: aG1,
		B: bG2,
		C: cG1,
	}, nil
}
