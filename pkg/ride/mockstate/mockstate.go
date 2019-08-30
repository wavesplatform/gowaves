package mockstate

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

var ErrNotFound = errors.New("Not found")

type MockStateImpl struct {
	TransactionsByID       map[string]proto.Transaction
	TransactionsHeightByID map[string]uint64
	Accounts               map[string]types.Account // recipient to account
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

func (a MockStateImpl) Account(r proto.Recipient) types.Account {
	return a.Accounts[r.String()]
}

func (a MockStateImpl) NewLastHeight() (uint64, error) {
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
