package bls12_381

import (
	"io"
	"reflect"
	"unsafe"

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

// FromBellmanVerifyingKey Gnark Groth16 only needs vk.e, vk.G2.gammaNeg, vk.G2.deltaNeg and vk.G1.K
func FromBellmanVerifyingKey(bvk *BellmanVerifyingKeyBl12381) gnark.VerifyingKey {
	vk := gnark.NewVerifyingKey(ecc.BLS12_381)

	/* set unexported vk.e */
	gt, _ := curveBls12.Pair([]curveBls12.G1Affine{bvk.G1.Alpha}, []curveBls12.G2Affine{bvk.G2.Beta})

	pointerVal := reflect.ValueOf(vk)
	val := reflect.Indirect(pointerVal)
	member := val.FieldByName("e")
	ptrToY := unsafe.Pointer(member.UnsafeAddr())
	realPtrToGT := (*curveBls12.GT)(ptrToY)
	*realPtrToGT = gt
	/* */

	/* set unexported G2.gammaNeg and G2.deltaNeg */
	gammaNeg := curveBls12.G2Affine{}
	gammaNeg.Neg(&bvk.G2.Gamma)
	deltaNeg := curveBls12.G2Affine{}
	deltaNeg.Neg(&bvk.G2.Delta)

	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{1, 4}) // G2.gammaNeg
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToGammaNeg := (*curveBls12.G2Affine)(ptrToY)
	*realPtrToGammaNeg = gammaNeg

	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{1, 3}) // G2.deltaNeg
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToDeltaNeg := (*curveBls12.G2Affine)(ptrToY)
	*realPtrToDeltaNeg = deltaNeg

	/* */

	/* set unexported G1.K */
	K := make([]curveBls12.G1Affine, len(bvk.G1.Ic))
	copy(K, bvk.G1.Ic)
	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{0, 3}) // G1.K
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToK := (*[]curveBls12.G1Affine)(ptrToY)
	*realPtrToK = K

	/* */
	return vk
}
