package crypto

import (
	"bytes"
	"encoding/binary"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	gnark "github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bls12_381"
	"github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bn256"
)

const (
	sizeUint64      = 8
	lenOneFrElement = 4
	frReprSize      = sizeUint64 * lenOneFrElement
)

func Groth16Verify(vkBytes []byte, proofBytes []byte, inputsBytes []byte, curve ecc.ID) (bool, error) {

	var vk gnark.VerifyingKey

	switch curve {
	case ecc.BLS12_381:
		var bvk bls12_381.BellmanVerifyingKeyBl12381
		_, err := bvk.ReadFrom(bytes.NewReader(vkBytes))
		if err != nil {
			return false, err
		}
		vk = bls12_381.FromBellmanVerifyingKey(&bvk)
	case ecc.BN254:
		var bvk bn256.BellmanVerifyingKeyBn256
		_, err := bvk.ReadFrom(bytes.NewReader(vkBytes))
		if err != nil {
			return false, err
		}
		vk = bn256.FromBellmanVerifyingKey(&bvk)
	default:
		return false, errors.Errorf("unknown eliptic curve")
	}

	proof := gnark.NewProof(curve)
	_, err := proof.ReadFrom(bytes.NewReader(proofBytes))
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
