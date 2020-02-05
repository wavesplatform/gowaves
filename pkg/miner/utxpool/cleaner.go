package utxpool

import (
	"context"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
	"go.uber.org/zap"
)

type Cleaner struct {
	ch         chan struct{}
	inner      types.Handler
	lastHeight proto.Height
	state      state.State
}

func NewCleaner(services services.Services) *Cleaner {
	return &Cleaner{
		state: services.State,
		inner: newInner(services.State, services.UtxPool, services.Time),
		ch:    make(chan struct{}, 1),
	}
}

// implements types.Handler
func (a *Cleaner) Handle() {
	select {
	case a.ch <- struct{}{}:
	default:
	}
}

func (a *Cleaner) work() {
	locked := a.state.Mutex().RLock()
	height, err := a.state.Height()
	locked.Unlock()
	if err != nil {
		zap.S().Error(err)
		return
	}

	if height != a.lastHeight {
		a.inner.Handle()
		a.lastHeight = height
	}
}

func (a *Cleaner) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.ch:
			a.work()
		}
	}
}

type inner struct {
	state stateWrapper
	utx   types.UtxPool
	tm    types.Time
}

func newInner(state stateWrapper, utx types.UtxPool, tm types.Time) *inner {
	return &inner{
		state: state,
		utx:   utx,
		tm:    tm,
	}
}

func (a inner) Handle() {
	transactions, err := a.handle()
	if err != nil {
		zap.S().Error(err)
		return
	}
	for _, t := range transactions {
		_ = a.utx.AddWithBytes(t.T, t.B)
	}
}

func (a inner) handle() ([]*types.TransactionWithBytes, error) {
	var transactions []*types.TransactionWithBytes
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	mu := a.state.Mutex()
	locked := mu.Lock()
	defer locked.Unlock()

	lastKnownBlock, err := a.state.TopBlock()
	if err != nil {
		return nil, err
	}

	for {
		t := a.utx.Pop()
		if t == nil {
			break
		}
		if err := a.state.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version); err == nil {
			transactions = append(transactions, t)
		}
	}
	a.state.ResetValidationList()
	return transactions, nil
}

type stateWrapper interface {
	TopBlock() (*proto.Block, error)
	ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, version proto.BlockVersion) error
	ResetValidationList()
	Mutex() *lock.RwMutex
}

type stateWrapperImpl struct {
	state state.State
}

func (a stateWrapperImpl) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, version proto.BlockVersion) error {
	return a.state.ValidateNextTx(tx, currentTimestamp, parentTimestamp, version)
}

func (a stateWrapperImpl) ResetValidationList() {
	a.state.ResetValidationList()
}

func (a stateWrapperImpl) Mutex() *lock.RwMutex {
	return a.state.Mutex()
}
func (a stateWrapperImpl) TopBlock() (*proto.Block, error) {
	return a.state.TopBlock()
}
