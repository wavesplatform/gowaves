package crypto

import (
	"fmt"
	"os"

	"github.com/consensys/gnark/cs/groth16"
	"github.com/consensys/gnark/ecc/bls381"
	"github.com/consensys/gnark/ecc/bls381/fp"
	"github.com/pkg/errors"
)

func Groth16Verify(vk, proof, inputs []byte) (bool, error) {
	if len(vk)%48 != 0 {
		return false, errors.New("invalid vk length, should be multiple of 48")
	}
	if len(inputs)%32 != 0 {
		return false, errors.New("invalid inputs length, should be multiple of 32")
	}
	if len(vk)/48 != len(inputs)/32+8 {
		return false, errors.New("invalid vk or proof length")
	}
	if len(proof) != 192 {
		return false, errors.New("invalid proof length, should be 192 bytes")
	}

	//load verifying key
	key := &groth16.VerifyingKey{
		E: ,
		G2Aff: struct {
			GammaNeg, DeltaNeg interface{}
		}{},
		G1Jac: struct {
			K []interface{}
		}{},
		PublicInputsTracker: nil,
	}

	if len(key.PublicInputsTracker)-1 != len(r1csInput) {
		fmt.Printf("invalid input size. expected %d got %d\n", len(vk.PublicInputsTracker), len(r1csInput))
		os.Exit(-1)
	}

	// load proof
	prf := groth16.Proof{
		Ar:  bls381.G1Affine{
			X: fp.Element{}.SetUint64(0),
			Y: fp.Element{},
		},
		Krs: bls381.G1Affine{
			X: fp.Element{},
			Y: fp.Element{},
		},
		Bs:  bls381.G2Affine{
			X: {fp.Element{}, fp.Element{}},
			Y: {fp.Element{}, fp.Element{}},
		},
	}

	return groth16.Verify(&prf, key, r1csInput)
}
