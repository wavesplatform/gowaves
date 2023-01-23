package wallet

import (
	"encoding/binary"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const curVersion = 1

type WalletFormat struct {
	Seed [][]byte `json:"seeds"`
}

type Wallet interface {
	AccountSeeds() [][]byte
	AddAccountSeed([]byte) error
	Encode(pass []byte) ([]byte, error)
}

type WalletImpl struct {
	Version uint32
	format  WalletFormat
}

func (a *WalletImpl) AccountSeeds() [][]byte {
	return a.format.Seed
}

func NewWallet() *WalletImpl {
	return &WalletImpl{
		format: WalletFormat{},
	}
}

func (a *WalletImpl) AddAccountSeed(seed []byte) error {
	s := common.Dup(seed)
	a.format.Seed = append(a.format.Seed, s)
	return nil
}

func (a *WalletImpl) Encode(password []byte) ([]byte, error) {

	crypt := NewCrypt(password)
	walletData, err := json.Marshal(a.format)
	if err != nil {
		return nil, err
	}

	rs, err := crypt.Encrypt(walletData)
	if err != nil {
		return nil, err
	}
	rs = append(make([]byte, 4), rs...)
	binary.BigEndian.PutUint32(rs[:4], curVersion)
	return rs, nil
}

func Decode(walletData []byte, password []byte) (Wallet, error) {
	version := binary.BigEndian.Uint32(walletData[:4])
	walletData = walletData[4:]
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
	return &WalletImpl{
		Version: version,
		format:  format,
	}, nil
}
