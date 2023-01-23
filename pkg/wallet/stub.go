package wallet

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Stub struct {
	S [][]byte
}

func (s Stub) SignTransactionWith(pk crypto.PublicKey, tx proto.Transaction) error {
	panic("Stub.SignTransactionWith: Unsopported operation")
}

func (s Stub) Load(password []byte) error {
	panic("Stub.Load: Unsopported operation")
}

func (s Stub) AccountSeeds() [][]byte {
	return s.S
}
