package bls12381

import (
	"bytes"
	"errors"
	"io"

	"github.com/consensys/gnark-crypto/ecc"
	curveBls12 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	gnark "github.com/consensys/gnark/backend/groth16"
)

// BellmanVerifyingKeyBl12381 is a struct that holds the verifying key for the BLS12-381 curve in bellman form.
// It is used to unmarshal the key from the shor bellman format and marshal it into the gnark format.
// Bellman format stores the following points:
//   - G1.Alpha (48 bytes)
//   - G2.Beta (96 bytes)
//   - G2.Gamma (96 bytes)
//   - G2.Delta (96 bytes)
//   - And slice G1.Ic of points (48 bytes each), written one after another, no delimiters, no size prefix.
//
// On the other hand, gnark format stores the following points of the verifying key:
//   - G1.Alpha (48 bytes)
//   - G1.Beta (48 bytes) equal to G1.Alpha
//   - G2.Beta (96 bytes)
//   - G2.Gamma (96 bytes)
//   - G1.Delta (48 bytes) equal to G1.Alpha
//   - G2.Delta (96 bytes)
//   - Slice G1.Ic of points (48 bytes each), prefixed with 4 bytes of len(G1.Ic)
//   - Empty slice of public commitments, only 4 bytes of zero size written
//   - Number of commitments, always zero (4 bytes).
type BellmanVerifyingKeyBl12381 struct {
	G1 struct {
		Alpha curveBls12.G1Affine
		Ic    []curveBls12.G1Affine
	}
	G2 struct {
		Beta, Gamma, Delta curveBls12.G2Affine
	}
}

// ReadFrom reads the verifying key in bellman format from the reader r.
func (vk *BellmanVerifyingKeyBl12381) ReadFrom(r io.Reader) (int64, error) {
	dec := curveBls12.NewDecoder(r)

	// Read [α]1,[β]2,[γ]2,[δ]2
	toDecode := []any{
		&vk.G1.Alpha,
		&vk.G2.Beta,
		&vk.G2.Gamma,
		&vk.G2.Delta,
	}
	for _, v := range toDecode {
		if err := dec.Decode(v); err != nil {
			return dec.BytesRead(), err
		}
	}

	// Read G1Affine points while possible.
	var p curveBls12.G1Affine
	for {
		if err := dec.Decode(&p); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return dec.BytesRead(), err
		}
		vk.G1.Ic = append(vk.G1.Ic, p)
	}

	return dec.BytesRead(), nil
}

// WriteTo writes the verifying key in gnark format to the writer w.
func (vk *BellmanVerifyingKeyBl12381) WriteTo(w io.Writer) (int64, error) {
	enc := curveBls12.NewEncoder(w)

	// Write [α]1, [β]1([α]1), [β]2, [γ]2, [δ]1([α]1), [δ]2
	if err := enc.Encode(&vk.G1.Alpha); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&vk.G1.Alpha); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&vk.G2.Beta); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&vk.G2.Gamma); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&vk.G1.Alpha); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&vk.G2.Delta); err != nil {
		return enc.BytesWritten(), err
	}

	if err := enc.Encode(vk.G1.Ic); err != nil {
		return enc.BytesWritten(), err
	}

	var publicCommitted [][]uint64
	var nbCommitments uint32

	// Encode 0 as length of publicCommited.
	if err := enc.Encode(publicCommitted); err != nil {
		return enc.BytesWritten(), err
	}
	// Encode number of commitments.
	if err := enc.Encode(nbCommitments); err != nil {
		return enc.BytesWritten(), err
	}
	return enc.BytesWritten(), nil
}

// FromBytesToVerifyingKey un-marshals the gnark verifying key from the bytes in bellman format.
func FromBytesToVerifyingKey(vkBytes []byte) (gnark.VerifyingKey, error) {
	var bvk BellmanVerifyingKeyBl12381
	_, err := bvk.ReadFrom(bytes.NewReader(vkBytes))
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	_, err = bvk.WriteTo(&b)
	if err != nil {
		return nil, err
	}
	vk := gnark.NewVerifyingKey(ecc.BLS12_381)
	_, err = vk.ReadFrom(bytes.NewReader(b.Bytes()))
	if err != nil {
		return nil, err
	}
	return vk, nil
}
