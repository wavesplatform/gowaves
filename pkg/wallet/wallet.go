package wallet

import (
	"encoding/binary"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type WalletFormat struct {
	Version int32  `json:"version"`
	Seed    []byte `json:"seed"`
	Index   uint32 `json:"index"`
}

type Wallet struct {
	format WalletFormat
}

func NewWalletFromSeed(seed []byte) (*Wallet, error) {
	s := make([]byte, len(seed))
	copy(s, seed)
	return &Wallet{
		format: WalletFormat{
			Version: 0,
			Seed:    s,
			Index:   0,
		},
	}, nil
}

func (a *Wallet) Encode(password []byte) ([]byte, error) {
	crypt := NewCrypt(password)
	walletData, err := json.Marshal(a.format)
	if err != nil {
		return nil, err
	}

	return crypt.Encrypt(walletData)
}

func (a *Wallet) GenPair() (crypto.SecretKey, crypto.PublicKey, error) {
	prefix := make([]byte, 4)
	binary.BigEndian.PutUint32(prefix, a.format.Index)

	s := append(prefix, a.format.Seed...)

	d, err := crypto.SecureHash(s)
	if err != nil {
		return crypto.SecretKey{}, crypto.PublicKey{}, err
	}

	priv, pub, err := crypto.GenerateKeyPair(d.Bytes())
	if err != nil {
		return crypto.SecretKey{}, crypto.PublicKey{}, err
	}

	return priv, pub, nil
}

func Decode(walletData []byte, password []byte) (*Wallet, error) {
	crypt := NewCrypt(password)
	bts, err := crypt.Decrypt(walletData)
	if err != nil {
		return nil, err
	}

	format := WalletFormat{}
	err = json.Unmarshal(bts, &format)
	if err != nil {
		return nil, errors.New("invalid password")
	}
	return &Wallet{
		format: format,
	}, nil
}

func (a *Wallet) Seed() []byte {
	out := make([]byte, len(a.format.Seed))
	copy(out, a.format.Seed)
	return out
}
