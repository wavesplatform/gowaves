package mockstate

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type State struct {
	TransactionsByID       map[string]proto.Transaction
	TransactionsHeightByID map[string]uint64
	AccountsBalance        uint64
	DataEntries            map[string]proto.DataEntry
	AssetIsSponsored       bool
	BlockHeaderByHeight    *proto.BlockHeader
}

func (a State) NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	return a.AccountsBalance, nil
}

func (a State) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	panic("implement NewestAddrByAlias")
}

func (a State) RetrieveNewestEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	v, ok := a.DataEntries[key]
	if !ok {
		return nil, errors.Errorf("key not found '%s'", key)
	}
	return v, nil
}

func (a State) TransactionByID(b []byte) (proto.Transaction, error) {
	t, ok := a.TransactionsByID[base58.Encode(b)]
	if !ok {
		return nil, state.NewNotFoundError(errors.New("not found"))
	}
	return t, nil
}

func (a State) TransactionHeightByID(b []byte) (uint64, error) {
	h, ok := a.TransactionsHeightByID[base58.Encode(b)]
	if !ok {
		return 0, state.NewNotFoundError(errors.New("not found"))
	}
	return h, nil
}

func (a State) NewestHeight() (uint64, error) {
	return 0, nil
}

func (a State) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	return a.AssetIsSponsored, nil
}

func (a State) HeaderByHeight(height proto.Height) (*proto.BlockHeader, error) {
	return a.BlockHeaderByHeight, nil
}
