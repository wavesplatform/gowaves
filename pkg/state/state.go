package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type State interface {
	TransactionByID(*crypto.Signature) (proto.Transaction, error)
	TransactionHeightByID(*crypto.Signature) (uint64, error)
}

type MockState struct {
	TransactionsByID       map[crypto.Signature]proto.Transaction
	TransactionsHeightByID map[crypto.Signature]uint64
}

func (a MockState) TransactionByID(s *crypto.Signature) (proto.Transaction, error) {
	t, ok := a.TransactionsByID[*s]
	if !ok {
		return nil, errors.New("transaction not found")
	}
	return t, nil
}

func (a MockState) TransactionHeightByID(s *crypto.Signature) (uint64, error) {
	h, ok := a.TransactionsHeightByID[*s]
	if !ok {
		return 0, errors.New("transaction not found")
	}
	return h, nil
}
