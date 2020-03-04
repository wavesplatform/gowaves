package utxpool

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
	"go.uber.org/zap"
)

type Cleaner struct {
	inner      BulkValidator
	lastHeight proto.Height
	state      stateWrapper
}

func NewCleaner(services services.Services) *Cleaner {
	return newCleaner(services.State, newBulkValidator(services.State, services.UtxPool, services.Time))
}

func newCleaner(state stateWrapper, validator BulkValidator) *Cleaner {
	return &Cleaner{
		state: state,
		inner: validator,
	}
}

// implements types.Handler
func (a *Cleaner) Handle() {
	a.work()
}

func (a *Cleaner) work() {
	locked := a.state.Mutex().RLock()
	height, err := a.state.Height()
	locked.Unlock()
	if err != nil {
		zap.S().Debug(err)
		return
	}

	if height != a.lastHeight {
		a.inner.Validate()
		a.lastHeight = height
	}
}

type stateWrapper interface {
	Height() (proto.Height, error)
	TopBlock() *proto.Block
	ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, version proto.BlockVersion) error
	ResetValidationList()
	Mutex() *lock.RwMutex
}
