package state

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var ErrNotFound = errors.New("Not found")

type Account interface {
	Data() []proto.DataEntry
	AssetBalance(*proto.OptionalAsset) uint64
	Address() proto.Address
}

type State interface {
	TransactionByID([]byte) (proto.Transaction, error)
	TransactionHeightByID([]byte) (uint64, error)
	Account(proto.Recipient) Account
}

type MockState struct {
	TransactionsByID       map[string]proto.Transaction
	TransactionsHeightByID map[string]uint64
	Accounts               map[string]Account // recipient to account
}

func (a MockState) TransactionByID(b []byte) (proto.Transaction, error) {
	t, ok := a.TransactionsByID[base58.Encode(b)]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (a MockState) TransactionHeightByID(b []byte) (uint64, error) {
	h, ok := a.TransactionsHeightByID[base58.Encode(b)]
	if !ok {
		return 0, ErrNotFound
	}
	return h, nil
}

func (a MockState) Account(r proto.Recipient) Account {
	return a.Accounts[r.String()]
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
