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

func (a KeyPair) Public() crypto.PublicKey {
	_, pub := crypto.GenerateKeyPair(a.seed)
	return pub
}

func (a KeyPair) Private() crypto.SecretKey {
	sec, _ := crypto.GenerateKeyPair(a.seed)
	return sec
}

func (a KeyPair) Addr(scheme byte) Address {
	pub := a.Public()
	addr, _ := NewAddressFromPublicKey(scheme, pub)
	return addr
}
