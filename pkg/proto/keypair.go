package proto

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type KeyPair struct {
	seed   []byte
	Public crypto.PublicKey
	Secret crypto.SecretKey
}

func NewKeyPair(seed []byte) (KeyPair, error) {
	sec, pub, err := crypto.GenerateKeyPair(seed)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{
		Public: pub,
		Secret: sec,
		seed:   seed,
	}, nil
}

func MustKeyPair(seed []byte) KeyPair {
	out, err := NewKeyPair(seed)
	if err != nil {
		panic(err)
	}
	return out
}

func (a KeyPair) Addr(scheme byte) (WavesAddress, error) {
	addr, err := NewAddressFromPublicKey(scheme, a.Public)
	if err != nil {
		return WavesAddress{}, err
	}
	return addr, nil
}
