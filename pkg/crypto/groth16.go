package crypto

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"unsafe"

	"github.com/consensys/gnark-crypto/ecc"
	curve "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	gnark "github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
)

const (
	sizeUint64      = 8
	lenOneFrElement = 4
	frReprSize      = sizeUint64 * lenOneFrElement
)

type BellmanVerifyingKey struct {
	G1 struct {
		Alpha curve.G1Affine
		Ic    []curve.G1Affine
	}
	G2 struct {
		Beta, Gamma, Delta curve.G2Affine
	}
}

func (vk *BellmanVerifyingKey) ReadFrom(r io.Reader) (n int64, err error) {
	{
		dec := curve.NewDecoder(r)
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
		dec := curve.NewDecoder(r)
		var p curve.G1Affine
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
func FromBellmanVerifyingKey(bvk *BellmanVerifyingKey) gnark.VerifyingKey {
	vk := gnark.NewVerifyingKey(ecc.BLS12_381)

	/* set unexported vk.e */
	gt, _ := curve.Pair([]curve.G1Affine{bvk.G1.Alpha}, []curve.G2Affine{bvk.G2.Beta})

	pointerVal := reflect.ValueOf(vk)
	val := reflect.Indirect(pointerVal)
	member := val.FieldByName("e")
	ptrToY := unsafe.Pointer(member.UnsafeAddr())
	realPtrToGT := (*curve.GT)(ptrToY)
	*realPtrToGT = gt
	/* */

	/* set unexported G2.gammaNeg and G2.deltaNeg */
	gammaNeg := curve.G2Affine{}
	gammaNeg.Neg(&bvk.G2.Gamma)
	deltaNeg := curve.G2Affine{}
	deltaNeg.Neg(&bvk.G2.Delta)

	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{1, 4}) // G2.gammaNeg
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToGammaNeg := (*curve.G2Affine)(ptrToY)
	*realPtrToGammaNeg = gammaNeg

	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{1, 3}) // G2.deltaNeg
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToDeltaNeg := (*curve.G2Affine)(ptrToY)
	*realPtrToDeltaNeg = deltaNeg

	/* */

	/* set unexported G1.K */
	K := make([]curve.G1Affine, len(bvk.G1.Ic))
	copy(K, bvk.G1.Ic)
	pointerVal = reflect.ValueOf(vk)
	val = reflect.Indirect(pointerVal)
	member = val.FieldByIndex([]int{0, 3}) // G1.K
	ptrToY = unsafe.Pointer(member.UnsafeAddr())
	realPtrToK := (*[]curve.G1Affine)(ptrToY)
	*realPtrToK = K

	/* */
	return vk
}

func Groth16Verify(vkBytes []byte, proofBytes []byte, inputsBytes []byte, curve ecc.ID) (bool, error) {

	var bvk BellmanVerifyingKey
	_, err := bvk.ReadFrom(bytes.NewReader(vkBytes))
	if err != nil {
		return false, err
	}
	vk := FromBellmanVerifyingKey(&bvk)

	proof := gnark.NewProof(curve)
	_, err = proof.ReadFrom(bytes.NewReader(proofBytes))
	if err != nil {
		return false, err
	}

	var buf bytes.Buffer
	err = binary.Write(&buf, binary.BigEndian, uint32(len(inputsBytes)/(fr.Limbs*sizeUint64)))
	if err != nil {
		return false, err
	}
	buf.Write(inputsBytes)

	wit := &witness.Witness{
		CurveID: curve,
	}
	err = wit.UnmarshalBinary(buf.Bytes())
	if err != nil {
		return false, err
	}
	err = gnark.Verify(proof, vk, wit)
	if err != nil {
		return false, nil
	}
	return true, nil
}
