package bls12381

import (
	"errors"
	"github.com/consensys/gurvy/bls381/fr"
	Proof "github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bls12381/proof"
	VerificationKey "github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bls12381/verificationKey"
	Verifier "github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bls12381/verifier"
	"math/big"
)

func ReadInputs(inputs []byte) ([]fr.Element, error) {
	var result []fr.Element
	const sizeUint64 = 8
	const lenOneFrElement = 4

	if len(inputs)%32 != 0 {
		return nil, errors.New("inputs should be % 32 = 0")
	}

	lenFrElements := len(inputs) / 32
	frReprSize := sizeUint64 * lenOneFrElement

	var currentOffset int
	var oldOffSet int

	// Put to []fr.Element every 32 bytes: [0:32], [32:64], ...
	for i := 0; i < lenFrElements; i++ {
		currentOffset += frReprSize
		elem := fr.One()
		elem.SetBytes(inputs[oldOffSet:currentOffset])
		oldOffSet += frReprSize

		result = append(result, elem)
	}

	return result, nil
}

func makeSliceBigInt(inputs []fr.Element) []*big.Int {
	publicInput := make([]*big.Int, 0)
	for _, v := range inputs {
		z := new(big.Int)
		z.SetBytes(v.Bytes())
		publicInput = append(publicInput, z)
	}
	return publicInput
}

func Groth16Verify(vk []byte, proof []byte, inputs []byte) (bool, error) {
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

	vkT, err := VerificationKey.GetVerificationKeyFromCompressed(vk)
	if err != nil {
		return false, err
	}
	proofT, err := Proof.GetProofFromCompressed(proof)
	if err != nil {
		return false, err
	}
	inputsFr, err := ReadInputs(inputs)
	if err != nil {
		return false, err
	}

	if len(inputsFr) != len(inputs)/32 || len(vkT.Ic) != len(inputs)/32+1 {
		return false, err
	}
	return Verifier.ProofVerify(vkT, proofT, makeSliceBigInt(inputsFr))
}
