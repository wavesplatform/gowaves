package storage

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	ROLLBACK_MAX_BLOCKS = 2000
)

var (
	heightKey = []byte("height")
	lastKey   = []byte("last") // For addr2Index, asset2Index.
)

type Iterator interface {
	Key() []byte
	Value() []byte
	Next() bool
	Erorr() error
	Release()
}

type AccountKeyVal interface {
	KeyValue
	NewKeyIterator(prefix []byte) (Iterator, error)
}

type AccountsStorage struct {
	globalStor  AccountKeyVal // AddrIndex+AssetIndex -> balance.
	addr2Index  KeyValue
	asset2Index KeyValue
}

func NewAccountsStorage(globalStor AccountKeyVal, addr2Index, asset2Index KeyValue) (*AccountsStorage, error) {
	return &AccountsStorage{
		globalStor:  globalStor,
		addr2Index:  addr2Index,
		asset2Index: asset2Index,
	}, nil
}

func (s *AccountsStorage) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	return 0, nil
}

func (s *AccountsStorage) SetAccountBalance(addr proto.Address, asset []byte, balance uint64) error {
	return nil
}

func (s *AccountsStorage) RollbackTo(newLast crypto.Signature) error {
	return nil
}
