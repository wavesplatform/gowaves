package wallet

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type seeder interface {
	AccountSeeds() [][]byte
}

type EmbeddedWalletImpl struct {
	loader Loader
	seeder seeder
	scheme proto.Scheme
	mu     sync.Mutex
}

func (a *EmbeddedWalletImpl) SignTransactionWith(pk crypto.PublicKey, tx proto.Transaction) error {
	seeds := a.seeder.AccountSeeds()
	for _, s := range seeds {
		secret, public, err := crypto.GenerateKeyPair(s)
		if err != nil {
			return err
		}
		if public == pk {
			return tx.Sign(a.scheme, secret)
		}
	}
	return ErrPublicKeyNotFound
}

func (a *EmbeddedWalletImpl) KeyPairsBLS() ([]bls.PublicKey, []bls.SecretKey, error) {
	seeds := a.seeder.AccountSeeds()
	var publicKeys []bls.PublicKey
	var secretKeys []bls.SecretKey
	for _, s := range seeds {
		secret, err := bls.GenerateSecretKey(s)
		if err != nil {
			continue
		}
		public, err := secret.PublicKey()
		if err != nil {
			return nil, nil, err
		}
		secretKeys = append(secretKeys, secret)
		publicKeys = append(publicKeys, public)
	}
	return publicKeys, secretKeys, nil
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

func (a *EmbeddedWalletImpl) AccountSeeds() [][]byte {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.seeder.AccountSeeds()
}

func NewEmbeddedWallet(path Loader, seeder seeder, scheme proto.Scheme) *EmbeddedWalletImpl {
	return &EmbeddedWalletImpl{
		loader: path,
		seeder: seeder,
		scheme: scheme,
	}
}
