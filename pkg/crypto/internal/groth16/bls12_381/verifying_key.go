package bls12_381

import (
	"bytes"
	"io"

	"github.com/consensys/gnark-crypto/ecc"
	curveBls12 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	gnark "github.com/consensys/gnark/backend/groth16"
)

type BellmanVerifyingKeyBl12381 struct {
	G1 struct {
		Alpha curveBls12.G1Affine
		Ic    []curveBls12.G1Affine
	}
	G2 struct {
		Beta, Gamma, Delta curveBls12.G2Affine
	}
}

func (vk *BellmanVerifyingKeyBl12381) ReadFrom(r io.Reader) (n int64, err error) {
	{
		dec := curveBls12.NewDecoder(r)
		toDecode := []interface{}{
			&vk.G1.Alpha,
			// &vk.G1.Beta,
			&vk.G2.Beta,
			&vk.G2.Gamma,
			// &vk.G1.Delta,
			&vk.G2.Delta,
		}
		for _, v := range toDecode {
			if err := dec.Decode(v); err != nil {
				return dec.BytesRead(), err
			}
		}
		n += dec.BytesRead()
	}
	{
		dec := curveBls12.NewDecoder(r)
		var p curveBls12.G1Affine
		for {
			err := dec.Decode(&p)
			if err == io.EOF {
				break
			}
			if err != nil {
				return n + dec.BytesRead(), err
			}
			vk.G1.Ic = append(vk.G1.Ic, p)
		}
		n += dec.BytesRead()
	}
	return
}

func (vk *BellmanVerifyingKeyBl12381) WriteTo(w io.Writer) (n int64, err error) {
	enc := curveBls12.NewEncoder(w)
	var emptyG1Field curveBls12.G1Affine
	// [α]1,[β]1,[β]2,[γ]2,[δ]1,[δ]2
	if err := enc.Encode(&vk.G1.Alpha); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&emptyG1Field); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&vk.G2.Beta); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&vk.G2.Gamma); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&emptyG1Field); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&vk.G2.Delta); err != nil {
		return enc.BytesWritten(), err
	}

	// uint32(len(Kvk)),[Kvk]1
	if err := enc.Encode(vk.G1.Ic); err != nil {
		return enc.BytesWritten(), err
	}
	return enc.BytesWritten(), nil
}

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
