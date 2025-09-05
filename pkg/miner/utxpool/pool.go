package utxpool

import (
	"container/heap"
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type heapItem struct {
	tx    *types.TransactionWithBytes
	index int // The index of the item in the heap.
}

type transactionsHeap []*heapItem

func (a transactionsHeap) Len() int { return len(a) }

func (a transactionsHeap) Less(i, j int) bool {
	// skip division by zero, check it when we add transaction
	return a[i].tx.T.GetFee()/uint64(len(a[i].tx.B)) > a[j].tx.T.GetFee()/uint64(len(a[j].tx.B))
}

func (a transactionsHeap) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
	a[i].index = i
	a[j].index = j
}

func (a *transactionsHeap) Push(x any) {
	item, ok := x.(*heapItem)
	if !ok || item == nil {
		panic(fmt.Sprintf("transactionsHeap.Push: unexpected item type %T", x))
	}
	if item.index != -1 {
		panic("transactionsHeap.Push: item already in heap")
	}
	item.index = len(*a)
	*a = append(*a, item)
}

func (a *transactionsHeap) Pop() any {
	old := *a
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	item.index = -1 // For safety, mark as no longer in the heap.
	old[n-1] = nil  // Avoid holding stale pointer.
	*a = old[:n-1]
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

func (a *UtxImpl) AllTransactions() []proto.Transaction {
	a.mu.Lock()
	defer a.mu.Unlock()
	res := make([]proto.Transaction, a.transactions.Len())
	for i, it := range a.transactions {
		res[i] = it.tx.T
	}
	return res
}

func (a *UtxImpl) Clean(ctx context.Context, shouldDrop func(tx proto.Transaction) bool) (int, int) {
	// Take a snapshot of the current transactions to avoid holding the lock for too long.
	a.mu.Lock()
	snapshot := make([]*heapItem, a.transactions.Len())
	copy(snapshot, a.transactions)
	a.mu.Unlock()

	// Check which transactions should be dropped.
	var drop []*heapItem
	checked := 0
	for _, it := range snapshot {
		if ctx.Err() != nil {
			slog.Debug("UTX cleanup interrupted", logging.Error(context.Cause(ctx)),
				slog.Int("checked", checked), slog.Int("dropped", len(drop)))
			break
		}
		if shouldDrop(it.tx.T) {
			drop = append(drop, it)
		}
		checked++
	}

	// Now remove the dropped transactions from the heap.
	a.mu.Lock()
	for _, it := range drop {
		if it.index >= 0 {
			heap.Remove(&a.transactions, it.index)
		}
	}
	a.mu.Unlock()
	return checked, len(drop)
}

// Add Must only be called inside state Map or MapUnsafe.
func (a *UtxImpl) Add(st types.UtxPoolValidatorState, tx proto.Transaction) error {
	bts, err := proto.MarshalTx(a.settings.AddressSchemeCharacter, tx)
	if err != nil {
		return fmt.Errorf("failed to put transaction to UTX: %w", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.addWithBytes(st, tx, bts)
}

// AddWithBytes Must only be called inside state Map or MapUnsafe.
func (a *UtxImpl) AddWithBytes(st types.UtxPoolValidatorState, tx proto.Transaction, b []byte) error {
	// TODO: add flag here to distinguish adding using API and accepting
	//  through the network from other nodes.
	//  When API is used, we should check all scripts completely.
	//  When adding from the network, only free complexity limit is checked.
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.addWithBytes(st, tx, b)
}

func (a *UtxImpl) AddWithBytesRaw(tx proto.Transaction, b []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.addWithBytesRaw(tx, b)
}

// addWithBytesRaw has no tx validation. Can be called wherever.
func (a *UtxImpl) addWithBytesRaw(tx proto.Transaction, b []byte) error {
	var noValidation func(proto.Transaction) error
	return a.addWithBytesOptValidation(tx, b, noValidation)
}

// addWithBytes Must only be called inside state Map or MapUnsafe.
func (a *UtxImpl) addWithBytes(st types.UtxPoolValidatorState, tx proto.Transaction, b []byte) error {
	return a.addWithBytesOptValidation(tx, b, func(t proto.Transaction) error {
		return a.validator.Validate(st, t)
	})
}

// addWithBytesOptValidation has optional tx validation. User is responsible for the validator closure.
func (a *UtxImpl) addWithBytesOptValidation(
	tx proto.Transaction,
	b []byte,
	optionalTxValidator func(t proto.Transaction) error,
) error {
	if len(b) == 0 {
		return errors.New("transaction with empty bytes")
	}
	// exceed limit
	if a.curSize+uint64(len(b)) > a.sizeLimit {
		return errors.Errorf("size overflow, curSize: %d, limit: %d", a.curSize, a.sizeLimit)
	}
	if err := tx.GenerateID(a.settings.AddressSchemeCharacter); err != nil {
		return errors.Errorf("failed to generate ID: %v", err)
	}
	tID, err := tx.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	if a.exists(tx) {
		return proto.NewInfoMsg(errors.Errorf("transaction with id %s exists", base58.Encode(tID)))
	}
	if optionalTxValidator != nil {
		if vErr := optionalTxValidator(tx); vErr != nil {
			return errors.Wrapf(vErr, "transaction with id %s failed validation", base58.Encode(tID))
		}
	}
	it := &heapItem{
		tx: &types.TransactionWithBytes{
			T: tx,
			B: b,
		},
		index: -1, // Not in heap yet.
	}
	heap.Push(&a.transactions, it)
	idb, err := tx.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return fmt.Errorf("failed to get tx id: %w", err)
	}
	id, err := crypto.NewDigestFromBytes(idb)
	if err != nil {
		return fmt.Errorf("failed to create digest from tx id: %w", err)
	}
	a.transactionIds[id] = struct{}{}
	a.curSize += uint64(len(b))
	return nil
}

func (a *UtxImpl) exists(tx proto.Transaction) bool {
	idb, err := tx.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return false
	}
	id, err := crypto.NewDigestFromBytes(idb)
	if err != nil {
		return false
	}
	_, ok := a.transactionIds[id]
	return ok
}

func (a *UtxImpl) Exists(tx proto.Transaction) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.exists(tx)
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

	if a.transactions.Len() == 0 {
		return nil
	}
	it, ok := heap.Pop(&a.transactions).(*heapItem)
	if !ok {
		panic("UtxImpl Pop: unexpected type from heap.Pop")
	}
	idb, err := it.tx.T.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return nil
	}
	id, err := crypto.NewDigestFromBytes(idb)
	if err != nil {
		return nil
	}
	delete(a.transactionIds, id)
	if uint64(len(it.tx.B)) > a.curSize {
		panic(fmt.Sprintf("UtxImpl Pop: size of transaction %d > than current size %d", len(it.tx.B), a.curSize))
	}
	a.curSize -= uint64(len(it.tx.B))
	return it.tx
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
