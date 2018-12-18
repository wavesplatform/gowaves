package state

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var ErrNotFound = errors.New("Not found")

type State interface {
	TransactionByID([]byte) (proto.Transaction, error)
	TransactionHeightByID([]byte) (uint64, error)
	AssetBalance(proto.Recipient, *proto.OptionalAsset) uint64
}

type MockState struct {
	TransactionsByID       map[string]proto.Transaction
	TransactionsHeightByID map[string]uint64
	AssetsByID             map[string]uint64 // addr + asset
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

func (a MockState) AssetBalance(recp proto.Recipient, asset *proto.OptionalAsset) uint64 {
	return a.AssetsByID[recp.String()+asset.String()]
}
