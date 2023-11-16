package bn256

import (
	"bytes"
	"io"

	"github.com/consensys/gnark-crypto/ecc/bn254"
)

func changeFlags(buf [bn254.SizeOfG2AffineCompressed]byte) [bn254.SizeOfG2AffineCompressed]byte {
	flags := buf[0] & (0b11 << 6)
	// check inf flag
	if flags&(0b01<<6) != 0 {
		buf[0] |= 0b01 << 6
		return buf
	}
	// check smallest
	if flags == 0 {
		buf[0] |= 0b10 << 6
		return buf
	}

	// check largest
	if flags&(0b10<<6) != 0 {
		buf[0] |= 0b11 << 6
		return buf
	}
	return buf
}

func changeFlagsInVKToGnarkType(buf []byte) ([]byte, error) {
	fixBuffer := make([]byte, 0, len(buf))
	var tmpBuf [bn254.SizeOfG2AffineCompressed]byte
	r := bytes.NewReader(buf)

	// G1.Alpha
	_, err := io.ReadFull(r, tmpBuf[:bn254.SizeOfG1AffineCompressed])
	if err != nil {
		return nil, err
	}
	tmpBuf = changeFlags(tmpBuf)
	fixBuffer = append(fixBuffer, tmpBuf[:bn254.SizeOfG1AffineCompressed]...)

	// G2.Beta
	_, err = io.ReadFull(r, tmpBuf[:bn254.SizeOfG2AffineCompressed])
	if err != nil {
		return nil, err
	}
	tmpBuf = changeFlags(tmpBuf)
	fixBuffer = append(fixBuffer, tmpBuf[:bn254.SizeOfG2AffineCompressed]...)

	// G2.Gamma
	_, err = io.ReadFull(r, tmpBuf[:bn254.SizeOfG2AffineCompressed])
	if err != nil {
		return nil, err
	}
	tmpBuf = changeFlags(tmpBuf)
	fixBuffer = append(fixBuffer, tmpBuf[:bn254.SizeOfG2AffineCompressed]...)

	// G2.Delta
	_, err = io.ReadFull(r, tmpBuf[:bn254.SizeOfG2AffineCompressed])
	if err != nil {
		return nil, err
	}
	tmpBuf = changeFlags(tmpBuf)
	fixBuffer = append(fixBuffer, tmpBuf[:bn254.SizeOfG2AffineCompressed]...)

	// G1.K
	for {
		_, err = io.ReadFull(r, tmpBuf[:bn254.SizeOfG1AffineCompressed])
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		tmpBuf = changeFlags(tmpBuf)
		fixBuffer = append(fixBuffer, tmpBuf[:bn254.SizeOfG1AffineCompressed]...)
	}
	return fixBuffer, nil
}

func changeFlagsInProofToGnarkType(buf []byte) ([]byte, error) {
	fixBuffer := make([]byte, 0, len(buf))
	var tmpBuf [bn254.SizeOfG2AffineCompressed]byte
	r := bytes.NewReader(buf)

	// G1.Ar
	_, err := io.ReadFull(r, tmpBuf[:bn254.SizeOfG1AffineCompressed])
	if err != nil {
		return nil, err
	}
	tmpBuf = changeFlags(tmpBuf)
	fixBuffer = append(fixBuffer, tmpBuf[:bn254.SizeOfG1AffineCompressed]...)

	// G2.Bs
	_, err = io.ReadFull(r, tmpBuf[:bn254.SizeOfG2AffineCompressed])
	if err != nil {
		return nil, err
	}
	tmpBuf = changeFlags(tmpBuf)
	fixBuffer = append(fixBuffer, tmpBuf[:bn254.SizeOfG2AffineCompressed]...)

	// G1.Krs
	_, err = io.ReadFull(r, tmpBuf[:bn254.SizeOfG1AffineCompressed])
	if err != nil {
		return nil, err
	}
	tmpBuf = changeFlags(tmpBuf)
	fixBuffer = append(fixBuffer, tmpBuf[:bn254.SizeOfG1AffineCompressed]...)

	return fixBuffer, nil
}
