package nullable

import "github.com/wavesplatform/gowaves/pkg/crypto"

type Signature struct {
	sig  crypto.Signature
	null bool
}

func NewNullSignature() Signature {
	return Signature{null: true}
}

func NewSignature(sig crypto.Signature) Signature {
	return Signature{
		sig: sig,
	}
}

func (a Signature) Null() bool {
	return a.null
}

func (a Signature) Sig() crypto.Signature {
	return a.sig
}
