package bls12381

import (
	"errors"
	"math/big"
)

func ReadInputs(inputs []byte) ([]*big.Int, error) {
	var result []*big.Int
	const sizeUint64 = 8
	const lenOneFrElement = 4

	if len(inputs)%32 != 0 {
		return nil, errors.New("inputs should be % 32 = 0")
	}

	lenFrElements := len(inputs) / 32
	frReprSize := sizeUint64 * lenOneFrElement

	var currentOffset int
	var oldOffSet int

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
