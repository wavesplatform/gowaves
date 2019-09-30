package mockstate

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type State struct {
	TransactionsByID       map[string]proto.Transaction
	TransactionsHeightByID map[string]uint64
	AccountsBalance        uint64
	DataEntries            map[string]proto.DataEntry
	AssetIsSponsored       bool
	BlockHeaderByHeight    *proto.BlockHeader
	NewestHeightVal        proto.Height
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

func (a State) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	v, ok := a.DataEntries[key]
	if !ok {
		return nil, errors.Errorf("key not found '%s'", key)
	}
	iv, ok := v.(*proto.IntegerDataEntry)
	if !ok {
		return nil, errors.Errorf("unexpected entry type %T", v)
	}
	return iv, nil
}

func (a State) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	v, ok := a.DataEntries[key]
	if !ok {
		return nil, errors.Errorf("key not found '%s'", key)
	}
	bv, ok := v.(*proto.BooleanDataEntry)
	if !ok {
		return nil, errors.Errorf("unexpected entry type %T", v)
	}
	return bv, nil
}

func (a State) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	v, ok := a.DataEntries[key]
	if !ok {
		return nil, errors.Errorf("key not found '%s'", key)
	}
	sv, ok := v.(*proto.StringDataEntry)
	if !ok {
		return nil, errors.Errorf("unexpected entry type %T", v)
	}
	return sv, nil
}

func (a State) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	v, ok := a.DataEntries[key]
	if !ok {
		return nil, errors.Errorf("key not found '%s'", key)
	}
	bv, ok := v.(*proto.BinaryDataEntry)
	if !ok {
		return nil, errors.Errorf("unexpected entry type %T", v)
	}
	return bv, nil
}

func (a State) NewestTransactionByID(b []byte) (proto.Transaction, error) {
	t, ok := a.TransactionsByID[base58.Encode(b)]
	if !ok {
		return nil, proto.ErrNotFound
	}
	return t, nil
}

func (a State) NewestTransactionHeightByID(b []byte) (uint64, error) {
	h, ok := a.TransactionsHeightByID[base58.Encode(b)]
	if !ok {
		return 0, proto.ErrNotFound
	}
	return h, nil
}

func (a State) NewestHeight() (uint64, error) {
	return a.NewestHeightVal, nil
}

func (a State) AddingBlockHeight() (uint64, error) {
	return 0, nil
}

func (a State) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	return a.AssetIsSponsored, nil
}

func (a State) HeaderByHeight(height proto.Height) (*proto.BlockHeader, error) {
	return a.BlockHeaderByHeight, nil
}

func (a State) IsNotFound(err error) bool {
	return err == proto.ErrNotFound
}
