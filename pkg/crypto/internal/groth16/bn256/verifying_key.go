package bn256

import (
	"bytes"
	"errors"
	"io"

	"github.com/consensys/gnark-crypto/ecc"
	curveBn254 "github.com/consensys/gnark-crypto/ecc/bn254"
	gnark "github.com/consensys/gnark/backend/groth16"
)

// BellmanVerifyingKeyBn256 is a struct that holds the verifying key for the BN256 curve in bellman form.
// It is used to unmarshal the key from the short bellman format and marshal it into the gnark format.
// See the documentation of the BellmanVerifyingKeyBl12381 for more details.
type BellmanVerifyingKeyBn256 struct {
	G1 struct {
		Alpha curveBn254.G1Affine
		Ic    []curveBn254.G1Affine
	}
	G2 struct {
		Beta, Gamma, Delta curveBn254.G2Affine
	}
}

// ReadFrom reads the verifying key in bellman format from the reader r.
func (vk *BellmanVerifyingKeyBn256) ReadFrom(r io.Reader) (int64, error) {
	dec := curveBn254.NewDecoder(r)
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

	var p curveBn254.G1Affine
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
func (vk *BellmanVerifyingKeyBn256) WriteTo(w io.Writer) (int64, error) {
	enc := curveBn254.NewEncoder(w)

	// [α]1, [β]1 ([α]1), [β]2, [γ]2, [δ]1 ([α]1), [δ]2
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

	// uint32(len(Kvk)),[Kvk]1
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

// FromBytesToVerifyingKey un-marshals the verifying key from the bytes in the bellman format to the gnark format.
func FromBytesToVerifyingKey(vkBytes []byte) (gnark.VerifyingKey, error) {
	var bvk BellmanVerifyingKeyBn256
	vkBytes, err := changeFlagsInVKToGnarkType(vkBytes)
	if err != nil {
		return nil, err
	}
	_, err = bvk.ReadFrom(bytes.NewReader(vkBytes))
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	_, err = bvk.WriteTo(&b)
	if err != nil {
		return nil, err
	}

	vk := gnark.NewVerifyingKey(ecc.BN254)
	_, err = vk.ReadFrom(bytes.NewReader(b.Bytes()))
	if err != nil {
		return nil, err
	}
	return vk, nil
}
