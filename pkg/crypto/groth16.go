package crypto

import (
	"bytes"
	"encoding/binary"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	gnark "github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	gnarkLog "github.com/consensys/gnark/logger"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bls12_381"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bn256"
)

const (
	sizeUint64 = 8
)

// Disable gnark logger because this lib use global logger and write to stdout
func init() {
	gnarkLog.Disable()
}

func Groth16Verify(vkBytes []byte, proofBytes []byte, inputsBytes []byte, curve ecc.ID) (bool, error) {

	var vk gnark.VerifyingKey

	switch curve {
	case ecc.BLS12_381:
		bls12381vk, err := bls12_381.FromBytesToVerifyingKey(vkBytes)
		if err != nil {
			return false, err
		}
		vk = bls12381vk
	case ecc.BN254:
		bn256vk, err := bn256.FromBytesToVerifyingKey(vkBytes)
		if err != nil {
			return false, err
		}
		vk = bn256vk

		// fix proof
		proofBytes, err = bn256.ChangeFlagsInProofToGnarkType(proofBytes)
		if err != nil {
			return false, err
		}
	default:
		return false, errors.Errorf("unknown eliptic curve")
	}

	proof := gnark.NewProof(curve)
	_, err := proof.ReadFrom(bytes.NewReader(proofBytes))
	if err != nil {
		return false, err
	}

	var buf bytes.Buffer
	buf.Grow(8 + 4 + len(inputsBytes))
	// Add 8 bytes for correct reading
	// Gnark witness has two addition number in the start
	// These numbers aren't used for verification
	buf.Write(make([]byte, 8))
	err = binary.Write(&buf, binary.BigEndian, uint32(len(inputsBytes)/(fr.Limbs*sizeUint64)))
	if err != nil {
		return false, err
	}
	buf.Write(inputsBytes)
	wit, err := witness.New(curve.ScalarField())
	if err != nil {
		return false, err
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
