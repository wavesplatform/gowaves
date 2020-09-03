package deserializer

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type Deserializer struct {
	b []byte
}

func NewDeserializer(b []byte) *Deserializer {
	return &Deserializer{
		b: b,
	}
}

func (a *Deserializer) Byte() (byte, error) {
	if len(a.b) > 0 {
		out := a.b[0]
		a.b = a.b[1:]
		return out, nil
	}
	return 0, errors.Errorf("not enough bytes, expected at least 1, found 0")
}

func (a *Deserializer) Digest() (crypto.Digest, error) {
	if len(a.b) < crypto.DigestSize {
		return crypto.Digest{},
			errors.Errorf(
				"not enough bytes to deserialize digest, expected at least %d, found %d",
				crypto.DigestSize,
				len(a.b))
	}
	rs, err := crypto.NewDigestFromBytes(a.b[:crypto.DigestSize])
	if err != nil {
		return crypto.Digest{}, errors.Wrap(err, "failed to parse Digest")
	}
	a.b = a.b[crypto.DigestSize:]
	return rs, nil
}

func (a *Deserializer) Signature() (crypto.Signature, error) {
	if len(a.b) < crypto.SignatureSize {
		return crypto.Signature{},
			errors.Errorf(
				"not enough bytes to deserialize signature, expected at least %d, found %d",
				crypto.SignatureSize,
				len(a.b))
	}
	rs, err := crypto.NewSignatureFromBytes(a.b[:crypto.SignatureSize])
	if err != nil {
		return crypto.Signature{}, errors.Wrap(err, "failed to parse signature")
	}
	a.b = a.b[crypto.SignatureSize:]
	return rs, nil
}

func (a *Deserializer) Uint32() (uint32, error) {
	if len(a.b) < 4 {
		return 0, errors.Errorf(
			"not enough bytes to deserialize uint32, expected at least %d, found %d",
			4,
			len(a.b))
	}
	out := binary.BigEndian.Uint32(a.b[:4])
	a.b = a.b[4:]
	return out, nil
}

func (a *Deserializer) Uint64() (uint64, error) {
	l := 8
	if len(a.b) < l {
		return 0, errors.Errorf(
			"not enough bytes to deserialize uint32, expected at least %d, found %d",
			l,
			len(a.b))
	}
	out := binary.BigEndian.Uint64(a.b[:l])
	a.b = a.b[l:]
	return out, nil
}

func (a *Deserializer) Uint16() (uint16, error) {
	l := 2
	if len(a.b) < l {
		return 0, errors.Errorf(
			"not enough bytes to deserialize uint32, expected at least %d, found %d",
			l,
			len(a.b))
	}
	out := binary.BigEndian.Uint16(a.b[:l])
	a.b = a.b[l:]
	return out, nil
}

// Length of the rest bytes.
func (a *Deserializer) Len() int {
	return len(a.b)
}

func (a *Deserializer) Bytes(length uint) ([]byte, error) {
	if length > uint(len(a.b)) {
		return nil, errors.Errorf(
			"not enough bytes to deserialize Bytes, expected %d, found %d",
			length,
			len(a.b))
	}
	out := a.b[:length]
	a.b = a.b[length:]
	return out, nil
}

func (a *Deserializer) PublicKey() (crypto.PublicKey, error) {
	l := len(a.b)
	bts, err := a.Bytes(crypto.PublicKeySize)
	if err != nil {
		return crypto.PublicKey{}, errors.Errorf(
			"not enough bytes to deserialize PublicKey, expected %d, found %d",
			crypto.PublicKeySize,
			l)
	}
	return crypto.NewPublicKeyFromBytes(bts)
}

func (a *Deserializer) ByteStringWithUint16Len() ([]byte, error) {
	l, err := a.Uint16()
	if err != nil {
		return nil, err
	}
	return a.Bytes(uint(l))
}
