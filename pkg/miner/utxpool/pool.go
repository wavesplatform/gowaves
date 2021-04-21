package utxpool

import (
	"container/heap"
	"fmt"
	"sync"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type transactionsHeap []*types.TransactionWithBytes

func (a transactionsHeap) Len() int { return len(a) }

func (a transactionsHeap) Less(i, j int) bool {
	// skip division by zero, check it when we add transaction
	return a[i].T.GetFee()/uint64(len(a[i].B)) > a[j].T.GetFee()/uint64(len(a[j].B))
}

func (a transactionsHeap) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a *transactionsHeap) Push(x interface{}) {
	item := x.(*types.TransactionWithBytes)
	*a = append(*a, item)
}

func (a *transactionsHeap) Pop() interface{} {
	old := *a
	n := len(old)
	item := old[n-1]
	*a = old[0 : n-1]
	return item
}

type UtxImpl struct {
	mu             sync.Mutex
	transactions   transactionsHeap
	transactionIds map[crypto.Digest]struct{}
	sizeLimit      uint64 // max transaction size in bytes
	curSize        uint64
	validator      Validator
	settings       *settings.BlockchainSettings
}

func New(sizeLimit uint64, validator Validator, settings *settings.BlockchainSettings) *UtxImpl {
	return &UtxImpl{
		transactionIds: make(map[crypto.Digest]struct{}),
		sizeLimit:      sizeLimit,
		validator:      validator,
		settings:       settings,
	}
}

func (a *UtxImpl) AllTransactions() []*types.TransactionWithBytes {
	a.mu.Lock()
	defer a.mu.Unlock()

	res := make([]*types.TransactionWithBytes, len(a.transactions))
	copy(res, a.transactions)
	return res
}

func (a *UtxImpl) Add(t proto.Transaction) error {
	bts, err := proto.MarshalTx(a.settings.AddressSchemeCharacter, t)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.addWithBytes(t, bts)
}

func (a *UtxImpl) AddBytes(bts []byte) error {
	t, err := proto.BytesToTransaction(bts, a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.addWithBytes(t, bts)
}

func (a *UtxImpl) AddWithBytes(t proto.Transaction, b []byte) error {
	// TODO: add flag here to distinguish adding using API and accepting
	//  through the network from other nodes.
	//  When API is used, we should check all scripts completely.
	//  When adding from the network, only free complexity limit is checked.
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.addWithBytes(t, b)
}

func (a *UtxImpl) addWithBytes(t proto.Transaction, b []byte) error {
	if len(b) == 0 {
		return errors.New("transaction with empty bytes")
	}
	// exceed limit
	if a.curSize+uint64(len(b)) > a.sizeLimit {
		return errors.Errorf("size overflow, curSize: %d, limit: %d", a.curSize, a.sizeLimit)
	}
	if err := t.GenerateID(a.settings.AddressSchemeCharacter); err != nil {
		return errors.Errorf("failed to generate ID: %v", err)
	}
	tID, err := t.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	if a.exists(t) {
		return proto.NewInfoMsg(errors.Errorf("transaction with id %s exists", base58.Encode(tID)))
	}
	err = a.validator.Validate(t)
	if err != nil {
		return err
	}
	tb := &types.TransactionWithBytes{
		T: t,
		B: b,
	}
	heap.Push(&a.transactions, tb)
	id := makeDigest(t.GetID(a.settings.AddressSchemeCharacter))
	a.transactionIds[id] = struct{}{}
	a.curSize += uint64(len(b))
	return nil
}

func (a *UtxImpl) Count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.transactions)
}

func makeDigest(b []byte, _ error) crypto.Digest {
	d := crypto.Digest{}
	copy(d[:], b)
	return d
}

func (a *UtxImpl) exists(t proto.Transaction) bool {
	_, ok := a.transactionIds[makeDigest(t.GetID(a.settings.AddressSchemeCharacter))]
	return ok
}

func (a *UtxImpl) Exists(t proto.Transaction) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.exists(t)
}

func (a *UtxImpl) ExistsByID(id []byte) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	digest, err := crypto.NewDigestFromBytes(id)
	if err != nil {
		return false
	}
	_, ok := a.transactionIds[digest]
	return ok
}

func (a *UtxImpl) Pop() *types.TransactionWithBytes {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.transactions.Len() > 0 {
		tb := heap.Pop(&a.transactions).(*types.TransactionWithBytes)
		delete(a.transactionIds, makeDigest(tb.T.GetID(a.settings.AddressSchemeCharacter)))
		if uint64(len(tb.B)) > a.curSize {
			panic(fmt.Sprintf("UtxImpl Pop: size of transaction %d > than current size %d", len(tb.B), a.curSize))
		}
		a.curSize -= uint64(len(tb.B))
		return tb
	}
	return nil
}

func (a *UtxImpl) CurSize() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.curSize
}

func (a *UtxImpl) Len() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.transactions.Len()
}
