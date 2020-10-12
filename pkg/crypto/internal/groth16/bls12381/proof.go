package bls12381

import (
	"bytes"
	bls "github.com/kilic/bls12-381"
)

type Proof struct {
	A *bls.PointG1
	B *bls.PointG2
	C *bls.PointG1
}

func GetProofFromCompressed(proof []byte) (*Proof, error) {
	reader := bytes.NewReader(proof)

	var g1Repr = make([]byte, 48)
	var g2Repr = make([]byte, 96)

	// A G1
	_, err := reader.Read(g1Repr)
	if err != nil {
		return nil, err
	}
	aG1, err := bls.NewG1().FromCompressed(g1Repr)
	if err != nil {
		return nil, err
	}

	// B G2
	_, err = reader.Read(g2Repr)
	if err != nil {
		return nil, err
	}
	bG2, err := bls.NewG2().FromCompressed(g2Repr)
	if err != nil {
		return nil, err
	}

	// C G1
	_, err = reader.Read(g1Repr)
	if err != nil {
		return nil, err
	}
	cG1, err := bls.NewG1().FromCompressed(g1Repr)
	if err != nil {
		return nil, err
	}

	return &Proof{
		A: aG1,
		B: bG2,
		C: cG1,
	}, nil
}
