package bn256

import (
	"io"
	"reflect"
	"unsafe"

	"github.com/consensys/gnark-crypto/ecc"
	curveBn254 "github.com/consensys/gnark-crypto/ecc/bn254"
	gnark "github.com/consensys/gnark/backend/groth16"
)

type BellmanVerifyingKeyBn256 struct {
	G1 struct {
		Alpha curveBn254.G1Affine
		Ic    []curveBn254.G1Affine
	}
	G2 struct {
		Beta, Gamma, Delta curveBn254.G2Affine
	}
}

func (vk *BellmanVerifyingKeyBn256) ReadFrom(r io.Reader) (n int64, err error) {
	{
		dec := curveBn254.NewDecoder(r)
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
		dec := curveBn254.NewDecoder(r)
		var p curveBn254.G1Affine
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
func FromBellmanVerifyingKey(bvk *BellmanVerifyingKeyBn256) gnark.VerifyingKey {
	vk := gnark.NewVerifyingKey(ecc.BLS12_381)

	/* set unexported vk.e */
	gt, _ := curveBn254.Pair([]curveBn254.G1Affine{bvk.G1.Alpha}, []curveBn254.G2Affine{bvk.G2.Beta})

	pointerVal := reflect.ValueOf(vk)
	val := reflect.Indirect(pointerVal)
	member := val.FieldByName("e")
	ptrToY := unsafe.Pointer(member.UnsafeAddr())
	realPtrToGT := (*curveBn254.GT)(ptrToY)
	*realPtrToGT = gt
	/* */

	/* set unexported G2.gammaNeg and G2.deltaNeg */
	gammaNeg := curveBn254.G2Affine{}
	gammaNeg.Neg(&bvk.G2.Gamma)
	deltaNeg := curveBn254.G2Affine{}
	deltaNeg.Neg(&bvk.G2.Delta)

	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{1, 4}) // G2.gammaNeg
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToGammaNeg := (*curveBn254.G2Affine)(ptrToY)
	*realPtrToGammaNeg = gammaNeg

	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{1, 3}) // G2.deltaNeg
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToDeltaNeg := (*curveBn254.G2Affine)(ptrToY)
	*realPtrToDeltaNeg = deltaNeg

	/* */

	/* set unexported G1.K */
	K := make([]curveBn254.G1Affine, len(bvk.G1.Ic))
	copy(K, bvk.G1.Ic)
	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{0, 3}) // G1.K
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToK := (*[]curveBn254.G1Affine)(ptrToY)
	*realPtrToK = K

	/* */
	return vk
}
