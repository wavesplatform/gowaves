package wallet

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Stub struct {
	S [][]byte
}

func (s Stub) SignTransactionWith(_ crypto.PublicKey, _ proto.Transaction) error {
	panic("Stub.SignTransactionWith: Unsopported operation")
}

func (s Stub) FindPublicKeyByAddress(_ proto.WavesAddress, _ proto.Scheme) (crypto.PublicKey, error) {
	panic("Stub.FindPublicKeyByAddress: Unsupported operation")
}

func (s Stub) BlsPairByWavesPK(_ crypto.PublicKey) (bls.SecretKey, bls.PublicKey, error) {
	panic("Stub.BlsPairByWavesPK: Unsupported operation")
}

func (s Stub) Load(_ []byte) error {
	panic("Stub.Load: Unsupported operation")
}

func (s Stub) AccountSeeds() [][]byte {
	return s.S
}
