package wallet

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
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

func (a *EmbeddedWalletImpl) FindPublicKeyByAddress(address proto.WavesAddress,
	scheme proto.Scheme) (crypto.PublicKey, error) {
	seeds := a.seeder.AccountSeeds()
	for _, s := range seeds {
		_, public, err := crypto.GenerateKeyPair(s)
		if err != nil {
			return crypto.PublicKey{}, err
		}
		retrievedAddress, err := proto.NewAddressFromPublicKey(scheme, public)
		if err != nil {
			return crypto.PublicKey{}, err
		}
		if retrievedAddress == address {
			return public, nil
		}
	}
	return crypto.PublicKey{}, ErrPublicKeyNotFound
}

func (a *EmbeddedWalletImpl) BLSPairByWavesPK(publicKey crypto.PublicKey) (bls.SecretKey, bls.PublicKey, error) {
	seeds := a.seeder.AccountSeeds()
	for _, s := range seeds {
		_, publicKeyRetrieved, err := crypto.GenerateKeyPair(s)
		if err != nil {
			return bls.SecretKey{}, bls.PublicKey{}, err
		}
		if publicKeyRetrieved == publicKey {
			secretKeyBls, genErr := bls.GenerateSecretKey(s)
			if genErr != nil {
				return bls.SecretKey{}, bls.PublicKey{}, genErr
			}
			publicKeyBls, retrieveErr := secretKeyBls.PublicKey()
			if retrieveErr != nil {
				return bls.SecretKey{}, bls.PublicKey{}, retrieveErr
			}
			return secretKeyBls, publicKeyBls, nil
		}
	}
	return bls.SecretKey{}, bls.PublicKey{}, ErrPublicKeyNotFound
}
func (a *EmbeddedWalletImpl) TopPkSkPairBLS() (bls.PublicKey,
	bls.SecretKey, error) {
	seeds := a.seeder.AccountSeeds()
	for _, s := range seeds {
		secret, err := bls.GenerateSecretKey(s)
		if err != nil {
			continue
		}
		public, err := secret.PublicKey()
		if err != nil {
			return bls.PublicKey{}, bls.SecretKey{}, err
		}
		return public, secret, nil
	}
	return bls.PublicKey{}, bls.SecretKey{}, errors.New("failed to find bls key pair")
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
