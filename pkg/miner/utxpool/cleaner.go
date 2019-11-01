package utxpool

import (
	"context"
	"time"

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
		inner: newInner(&stateWrapperImpl{services.State}, services.UtxPool),
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
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-a.ch:
				a.work()
			}
		}
	}()
}

type inner struct {
	state stateWrapper
	utx   types.UtxPool
}

func newInner(state stateWrapper, utx types.UtxPool) *inner {
	return &inner{
		state: state,
		utx:   utx,
	}
}

func (a inner) Handle() {
	transactions, err := a.handle()
	if err != nil {
		zap.S().Error(err)
		return
	}
	for _, t := range transactions {
		a.utx.AddWithBytes(t.T, t.B)
	}
}

func (a inner) handle() ([]*types.TransactionWithBytes, error) {
	var transactions []*types.TransactionWithBytes
	currentTimestamp := proto.NewTimestampFromTime(time.Now())
	mu := a.state.Mutex()
	locked := mu.Lock()
	defer locked.Unlock()

	lastKnownBlock, err := a.state.lastHeader()
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
	lastHeader() (*proto.BlockHeader, error)
	ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, version proto.BlockVersion) error
	ResetValidationList()
	Mutex() *lock.RwMutex
}

type stateWrapperImpl struct {
	state state.State
}

func (a stateWrapperImpl) lastHeader() (*proto.BlockHeader, error) {
	height, err := a.state.Height()
	if err != nil {
		return nil, err
	}
	return a.state.HeaderByHeight(height)
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
