package mockstate

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

//TODO: Get rid of this error
var ErrNotFound = errors.New("Not found")

type MockStateImpl struct {
	TransactionsByID       map[string]proto.Transaction
	TransactionsHeightByID map[string]uint64
	AccountsBalance        uint64
	DataEntry              proto.DataEntry
	AssetIsSponsored       bool
	BlockHeaderByHeight    *proto.BlockHeader
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
		return nil, state.NewNotFoundError(errors.New("not found"))
	}
	return t, nil
}

func (a MockStateImpl) TransactionHeightByID(b []byte) (uint64, error) {
	h, ok := a.TransactionsHeightByID[base58.Encode(b)]
	if !ok {
		return 0, ErrNotFound //FIXME: return proper error
	}
	return h, nil
}

func (a MockStateImpl) NewestHeight() (uint64, error) {
	return 0, nil
}

func (a MockStateImpl) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	return a.AssetIsSponsored, nil
}

func (a MockStateImpl) HeaderByHeight(height proto.Height) (*proto.BlockHeader, error) {
	return a.BlockHeaderByHeight, nil
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
