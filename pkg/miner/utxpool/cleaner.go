package utxpool

import (
	"context"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Cleaner struct {
	inner BulkValidator
	state stateWrapper
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

func (a *Cleaner) Clean(ctx context.Context) {
	a.work(ctx)
}

func (a *Cleaner) work(ctx context.Context) {
	a.inner.Validate(ctx)
}

type stateWrapper interface {
	Height() (proto.Height, error)
	TopBlock() *proto.Block
	TxValidation(func(validation state.TxValidation) error) error
	ResetList()
	ResetListUnsafe(func(validation state.TxValidation) error) error
	Map(func(state state.NonThreadSafeState) error) error
	MapUnsafe(func(state state.NonThreadSafeState) error) error
	IsActivated(featureID int16) (bool, error)
}
