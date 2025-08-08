package bn256

import (
	"bytes"
	"io"

	"github.com/consensys/gnark-crypto/ecc"
	curveBn254 "github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/pedersen"
	gnark "github.com/consensys/gnark/backend/groth16"
)

type BellmanProofBn256 struct {
	Ar  curveBn254.G1Affine
	Bs  curveBn254.G2Affine
	Krs curveBn254.G1Affine
}

func (proof *BellmanProofBn256) ReadFrom(r io.Reader) (int64, error) {
	dec := curveBn254.NewDecoder(r)
	toDecode := []any{
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

func (proof *BellmanProofBn256) WriteTo(w io.Writer) (int64, error) {
	enc := curveBn254.NewEncoder(w)
	emptyG1List := make([]curveBn254.G1Affine, 0)
	var emptyG1Field curveBn254.G1Affine
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
	var bproof BellmanProofBn256
	proofBytes, err := changeFlagsInProofToGnarkType(proofBytes)
	if err != nil {
		return nil, err
	}
	_, err = bproof.ReadFrom(bytes.NewReader(proofBytes))
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	_, err = bproof.WriteTo(&b)
	if err != nil {
		return nil, err
	}

	proof := gnark.NewProof(ecc.BN254)
	_, err = proof.ReadFrom(bytes.NewReader(b.Bytes()))
	if err != nil {
		return nil, err
	}
	return proof, nil
}
