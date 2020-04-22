package utxpool

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type Cleaner struct {
	inner      BulkValidator
	lastHeight proto.Height
	state      stateWrapper
}

func NewCleaner(state state.State, pool types.UtxPool, tm types.Time) *Cleaner {
	return newCleaner(state, newBulkValidator(state, pool, tm))
}

func newCleaner(state stateWrapper, validator BulkValidator) *Cleaner {
	return &Cleaner{
		state: state,
		inner: validator,
	}
}

func (a *Cleaner) Clean() {
	a.work()
}

func (a *Cleaner) work() {
	height, err := a.state.Height()
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
	BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error)
	TxValidation(func(validation state.TxValidation) error) error
}
