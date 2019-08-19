package proto

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type KeyPair struct {
	seed []byte
}

func NewKeyPair(seed []byte) KeyPair {
	return KeyPair{
		seed: seed,
	}
}

func (a KeyPair) Public() (crypto.PublicKey, error) {
	_, pub, err := crypto.GenerateKeyPair(a.seed)
	if err != nil {
		return pub, nil
	}
	return pub, nil
}

func (a KeyPair) Private() (crypto.SecretKey, error) {
	sec, _, err := crypto.GenerateKeyPair(a.seed)
	if err != nil {
		return sec, err
	}
	return sec, nil
}

func (a KeyPair) Addr(scheme byte) (Address, error) {
	pub, err := a.Public()
	if err != nil {
		return Address{}, err
	}
	addr, err := NewAddressFromPublicKey(scheme, pub)
	if err != nil {
		return Address{}, err
	}
	return addr, nil
}
