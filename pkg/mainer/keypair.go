package mainer

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type KeyPair struct {
}

func NewKeyPairFromSeed(b []byte) KeyPair {
	panic("not implemented")
}

func (a *KeyPair) Public() crypto.PublicKey {
	panic("not implemented")
}

func (a *KeyPair) Private() crypto.SecretKey {
	panic("not implemented")
}

func (a *KeyPair) Addr() proto.Address {
	panic("not implemented")
}
