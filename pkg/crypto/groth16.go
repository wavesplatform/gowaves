package crypto

import (
	"errors"
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bls12381"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bn256"
)

type Bls12381 struct{}

type Bn256 struct{}

type Dummy struct{}

const (
	sizeUint64      = 8
	lenOneFrElement = 4
	frReprSize      = sizeUint64 * lenOneFrElement
)

func ReadInputs(inputs []byte) ([]*big.Int, error) {
	var result []*big.Int

	if len(inputs)%frReprSize != 0 {
		return nil, errors.New("inputs should be % 32 = 0")
	}

	var (
		currentOffset int
		oldOffSet     int
		lenFrElements = len(inputs) / frReprSize
	)
	// Appending every 32 bytes [0..32], [32..64], ...
	for i := 0; i < lenFrElements; i++ {
		currentOffset += frReprSize
		elem := new(big.Int)
		elem.SetBytes((inputs)[oldOffSet:currentOffset])
		oldOffSet += frReprSize

		result = append(result, elem)
	}

	return result, nil
}

func SerializeInputs(inputs []*big.Int) []byte {
	if len(inputs) == 0 {
		return nil
	}

	out := make([]byte, frReprSize*len(inputs))

	outInput := out[:0]
	for _, input := range inputs {
		outInput = outInput[:frReprSize]
		input.FillBytes(outInput)
		outInput = outInput[frReprSize:]
	}
	return out
}

func (Bls12381) Groth16Verify(vk []byte, proof []byte, inputs []byte) (bool, error) {
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

	vkT, err := bls12381.GetVerificationKeyFromCompressed(vk)
	if err != nil {
		return false, err
	}
	proofT, err := bls12381.GetProofFromCompressed(proof)
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
	return bls12381.ProofVerify(vkT, proofT, inputsFr)
}

func (Bn256) Groth16Verify(vk []byte, proof []byte, inputs []byte) (bool, error) {
	if len(vk)%32 != 0 {
		return false, errors.New("invalid vk length, should be multiple of 32")
	}
	if len(inputs)%32 != 0 {
		return false, errors.New("invalid inputs length, should be multiple of 32")
	}
	if len(vk)/32 != len(inputs)/32+8 {
		return false, errors.New("invalid vk or proof length")
	}
	if len(proof) != 128 {
		return false, errors.New("invalid proof length, should be 128 bytes")
	}

	vkT, err := bn256.GetVerificationKeyFromCompressed(vk)
	if err != nil {
		return false, err
	}
	proofT, err := bn256.GetProofFromCompressed(proof)
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

	return bn256.ProofVerify(vkT, proofT, inputsFr)

}

func (Dummy) Groth16Verify(vk, proof, inputs []byte) (bool, error) {
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

	return true, nil
}
