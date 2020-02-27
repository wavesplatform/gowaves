package wallet

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type seeder interface {
	Seeds() [][]byte
}

type EmbeddedWalletImpl struct {
	loader Loader
	seeder seeder
	scheme proto.Scheme
	mu     sync.Mutex
}

func (a *EmbeddedWalletImpl) SignTransactionWith(pk crypto.PublicKey, tx proto.Transaction) error {
	seeds := a.seeder.Seeds()
	for _, s := range seeds {
		secret, public, err := crypto.GenerateKeyPair(s)
		if err != nil {
			return err
		}
		if public == pk {
			return tx.Sign(secret)
		}
	}
	return PublicKeyNotFound
}

func (a *EmbeddedWalletImpl) Load(password []byte) error {
	bts, err := a.loader.Load()
	if err != nil {
		return err
	}
	w, err := Decode(bts, password)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.seeder = w
	a.mu.Unlock()
	return nil
}

func (a *EmbeddedWalletImpl) Seeds() [][]byte {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.seeder.Seeds()
}

func NewEmbeddedWallet(path Loader, seeder seeder, scheme proto.Scheme) *EmbeddedWalletImpl {
	return &EmbeddedWalletImpl{
		loader: path,
		seeder: seeder,
		scheme: scheme,
	}
}
