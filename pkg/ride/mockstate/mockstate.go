package mockstate

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var ErrNotFound = errors.New("Not found")

type MockStateImpl struct {
	TransactionsByID       map[string]proto.Transaction
	TransactionsHeightByID map[string]uint64
	AccountsBalance        uint64
	DataEntry              proto.DataEntry
}

func (a MockStateImpl) NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	return a.AccountsBalance, nil
}

func (a MockStateImpl) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	panic("implement NewestAddrByAlias")
}

func (a MockStateImpl) RetrieveNewestEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	return a.DataEntry, nil
}

func (a MockStateImpl) TransactionByID(b []byte) (proto.Transaction, error) {
	t, ok := a.TransactionsByID[base58.Encode(b)]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (a MockStateImpl) TransactionHeightByID(b []byte) (uint64, error) {
	h, ok := a.TransactionsHeightByID[base58.Encode(b)]
	if !ok {
		return 0, ErrNotFound
	}
	return h, nil
}

func (a MockStateImpl) NewestHeight() (uint64, error) {
	return 0, nil
}

type MockAccount struct {
	Assets       map[string]uint64
	DataEntries  []proto.DataEntry
	AddressField proto.Address
}

func (a *MockAccount) Data() []proto.DataEntry {
	return a.DataEntries
}
func (a *MockAccount) AssetBalance(p *proto.OptionalAsset) uint64 {
	return a.Assets[p.String()]
}

func (a *MockAccount) Address() proto.Address {
	return a.AddressField
}
