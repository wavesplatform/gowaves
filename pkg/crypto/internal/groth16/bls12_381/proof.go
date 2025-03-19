package bls12381

import (
	"bytes"
	"io"

	"github.com/consensys/gnark-crypto/ecc"
	curveBls12 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr/pedersen"
	gnark "github.com/consensys/gnark/backend/groth16"
)

type BellmanProofBl12381 struct {
	Ar  curveBls12.G1Affine
	Bs  curveBls12.G2Affine
	Krs curveBls12.G1Affine
}

func (proof *BellmanProofBl12381) ReadFrom(r io.Reader) (int64, error) {
	dec := curveBls12.NewDecoder(r)
	toDecode := []interface{}{
		&proof.Ar,
		&proof.Bs,
		&proof.Krs,
	}

	for _, v := range toDecode {
		if err := dec.Decode(v); err != nil {
			return dec.BytesRead(), err
		}
	}

	return dec.BytesRead(), nil
}

func (proof *BellmanProofBl12381) WriteTo(w io.Writer) (int64, error) {
	enc := curveBls12.NewEncoder(w)
	emptyG1List := make([]curveBls12.G1Affine, 0)
	var emptyG1Field curveBls12.G1Affine
	// Ar | Krs | bs
	if err := enc.Encode(&proof.Ar); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&proof.Bs); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&proof.Krs); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(emptyG1List); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&emptyG1Field); err != nil {
		return enc.BytesWritten(), err
	}

	commitmentKey := pedersen.VerifyingKey{}
	m, err := commitmentKey.WriteTo(w)
	return enc.BytesWritten() + m, err
}

func FromBytesToProof(proofBytes []byte) (gnark.Proof, error) {
	var bproof BellmanProofBl12381
	_, err := bproof.ReadFrom(bytes.NewReader(proofBytes))
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	_, err = bproof.WriteTo(&b)
	if err != nil {
		return nil, err
	}

	proof := gnark.NewProof(ecc.BLS12_381)
	_, err = proof.ReadFrom(bytes.NewReader(b.Bytes()))
	if err != nil {
		return nil, err
	}
	return proof, nil
}
