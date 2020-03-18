package crypto

import (
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

	//TODO: implement function
	return true, nil
	//prf, err := loadProof(proof)
	//if err != nil {
	//	return false, err
	//}
	//key, err := loadKey(vk)
	//if err != nil {
	//	return false, err
	//}
	//
	//r1csInput, err := loadInputs(inputs)
	//if err != nil {
	//	return false, err
	//}
	//if len(key.PublicInputsTracker)-1 != len(r1csInput) {
	//	return false, errors.Errorf("invalid input size. expected %d got %d\n", len(key.PublicInputsTracker), len(r1csInput))
	//}
	//
	//return groth16.Verify(prf, key, r1csInput)
}

//func loadInputs(data []byte) (cs.Assignments, error) {
//	return nil, errors.New("not implemented")
//}
//
//func loadProof(data []byte) (*groth16.Proof, error) {
//	return nil, errors.New("not implemented")
//}
//
//func loadKey(data []byte) (*groth16.VerifyingKey, error) {
//	return nil, errors.New("not implemented")
//}
//
//func decompressG1(point []byte) (bls381.G1Affine, error) {
//	if len(point) != 48 {
//		return bls381.G1Affine{}, errors.New("invalid G1 point length")
//	}
//	if point[0]&(1<<7) == 0 {
//		return bls381.G1Affine{}, errors.New("invalid compression of G1 point")
//	}
//	if point[0]&(1<<6) != 0 {
//		// This is the point at infinity, which means that if we mask away
//		// the first two bits, the entire representation should consist
//		// of zeroes.
//		point[0] &= 0x3f
//		allZeros := true
//		for _, b := range point {
//			if b != 0 {
//				allZeros = false
//				break
//			}
//		}
//		if !allZeros {
//			return bls381.G1Affine{}, errors.New("unexpected G1 point")
//		}
//		return bls381.G1Affine{X: fp.Element{}, Y: fp.Element{}}, nil
//
//	}
//	// Determine if the intended y coordinate must be greater
//	// lexicographically.
//	greatest := point[0]&(1<<5) != 0
//	// Unset the three most significant bits.
//	point[0] &= 0x1f
//	x, err := readElement(point)
//	if err != nil {
//		return bls381.G1Affine{}, err
//	}
//
//	//	let greatest = copy[0] & (1 << 5) != 0;
//	//
//	//
//	//	copy[0] &= 0x1f;
//	//
//	//	let mut x = FqRepr([0; 6]);
//	//
//	//{
//	//let mut reader = &copy[..];
//	//
//	//x.read_be(&mut reader).unwrap();
//	//}
//	//
//	//// Interpret as Fq element.
//	//let x = Fq::from_repr(x)
//	//.map_err(|e| GroupDecodingError::CoordinateDecodingError("x coordinate", e))?;
//	//
//	//G1Affine::get_point_from_x(x, greatest).ok_or(GroupDecodingError::NotOnCurve)
//	//}
//
//}
//
//func readElement(data []byte) (*fp.Element, error) {
//	if len(data) < 6*8 {
//		return nil, errors.New("insufficient data")
//	}
//	z := &fp.Element{}
//	z[0] = binary.BigEndian.Uint64(data[0:])
//	z[1] = binary.BigEndian.Uint64(data[8:])
//	z[2] = binary.BigEndian.Uint64(data[16:])
//	z[3] = binary.BigEndian.Uint64(data[24:])
//	z[4] = binary.BigEndian.Uint64(data[32:])
//	z[5] = binary.BigEndian.Uint64(data[40:])
//	z[5] %= 1873798617647539866
//
//	// if z > q --> z -= q
//	if !(z[5] < 1873798617647539866 || (z[5] == 1873798617647539866 && (z[4] < 5412103778470702295 || (z[4] == 5412103778470702295 && (z[3] < 7239337960414712511 || (z[3] == 7239337960414712511 && (z[2] < 7435674573564081700 || (z[2] == 7435674573564081700 && (z[1] < 2210141511517208575 || (z[1] == 2210141511517208575 && (z[0] < 13402431016077863595))))))))))) {
//		var b uint64
//		z[0], b = bits.Sub64(z[0], 13402431016077863595, 0)
//		z[1], b = bits.Sub64(z[1], 2210141511517208575, b)
//		z[2], b = bits.Sub64(z[2], 7435674573564081700, b)
//		z[3], b = bits.Sub64(z[3], 7239337960414712511, b)
//		z[4], b = bits.Sub64(z[4], 5412103778470702295, b)
//		z[5], _ = bits.Sub64(z[5], 1873798617647539866, b)
//	}
//	return z, nil
//}
//
//func pointFromX(x *fp.Element, greates bool) (bls381.G1Affine, error) {
//	z := fp.Element{}
//	z.Square(x)
//	z.MulAssign(x)
//	z.AddAssign(&bls381.BLS381().B)
//	z.
//}
